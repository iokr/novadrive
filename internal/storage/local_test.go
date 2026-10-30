package storage

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"io"
	"strings"
	"testing"

	"github.com/iokr/novadrive/internal/config"
)

// newTestLocal 在临时目录上构建存储实现。
func newTestLocal(t *testing.T) *Local {
	t.Helper()
	base := t.TempDir()
	cfg := &config.Config{}
	cfg.Storage.Local.Root = base + "/blobs"
	cfg.Storage.Local.ThumbRoot = base + "/thumbs"
	cfg.Storage.Local.UploadRoot = base + "/uploads"
	cfg.Storage.Chunk.MergeBufferSize = 1024
	l, err := NewLocal(cfg)
	if err != nil {
		t.Fatalf("NewLocal: %v", err)
	}
	return l
}

func TestSaveAndMergeChunks(t *testing.T) {
	l := newTestLocal(t)
	const uploadID = "b3b6d0f0-2a15-4f1c-9a21-123456789abc"
	parts := []string{"hello ", "novadrive ", "chunked"}
	for i, p := range parts {
		if _, err := l.SaveChunk(uploadID, i, strings.NewReader(p)); err != nil {
			t.Fatalf("SaveChunk(%d): %v", i, err)
		}
	}
	// 重传覆盖应保持幂等。
	if _, err := l.SaveChunk(uploadID, 1, strings.NewReader(parts[1])); err != nil {
		t.Fatalf("SaveChunk retry: %v", err)
	}
	indices, err := l.ListChunks(uploadID)
	if err != nil {
		t.Fatalf("ListChunks: %v", err)
	}
	if len(indices) != len(parts) {
		t.Fatalf("分片数量期望 %d，实际 %d", len(parts), len(indices))
	}

	var buf bytes.Buffer
	hash, size, err := l.MergeChunks(uploadID, len(parts), &buf)
	if err != nil {
		t.Fatalf("MergeChunks: %v", err)
	}
	joined := strings.Join(parts, "")
	sum := md5.Sum([]byte(joined))
	if want := hex.EncodeToString(sum[:]); hash != want {
		t.Fatalf("MD5 期望 %s，实际 %s", want, hash)
	}
	if size != int64(len(joined)) || buf.String() != joined {
		t.Fatalf("合并结果不符，size=%d body=%q", size, buf.String())
	}

	if err := l.DeleteChunkDir(uploadID); err != nil {
		t.Fatalf("DeleteChunkDir: %v", err)
	}
	// 目录被删除后不应残留分片。
	indices, err = l.ListChunks(uploadID)
	if err != nil {
		t.Fatalf("ListChunks after delete: %v", err)
	}
	if len(indices) != 0 {
		t.Fatalf("期望清空分片，实际 %v", indices)
	}
}

func TestMissingChunkMergeFails(t *testing.T) {
	l := newTestLocal(t)
	const uploadID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	if _, err := l.SaveChunk(uploadID, 0, strings.NewReader("only-first")); err != nil {
		t.Fatalf("SaveChunk: %v", err)
	}
	if _, _, err := l.MergeChunks(uploadID, 2, io.Discard); err == nil {
		t.Fatal("缺失分片时应当返回错误")
	}
}

func TestTempAndBlobRoundtrip(t *testing.T) {
	l := newTestLocal(t)
	tmpPath, size, err := l.SaveTemp(strings.NewReader("payload"))
	if err != nil {
		t.Fatalf("SaveTemp: %v", err)
	}
	if size != int64(len("payload")) {
		t.Fatalf("临时文件大小期望 7，实际 %d", size)
	}
	reader, err := l.OpenTemp(tmpPath)
	if err != nil {
		t.Fatalf("OpenTemp: %v", err)
	}
	data, err := io.ReadAll(reader)
	reader.(io.Closer).Close()
	if err != nil || string(data) != "payload" {
		t.Fatalf("读取临时文件失败: %v data=%q", err, string(data))
	}

	sum := md5.Sum(data)
	hash := hex.EncodeToString(sum[:])
	relPath, err := l.MoveToBlob(tmpPath, hash, "txt")
	if err != nil {
		t.Fatalf("MoveToBlob: %v", err)
	}
	if want := hash[:2] + "/" + hash; relPath != want {
		t.Fatalf("blob 相对路径期望 %s，实际 %s", want, relPath)
	}
	if !l.BlobExists(relPath) {
		t.Fatal("BlobExists 应为 true")
	}
	blob, err := l.Open(relPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	blob.(io.Closer).Close()
	if err := l.Delete(relPath); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if l.BlobExists(relPath) {
		t.Fatal("删除后 BlobExists 应为 false")
	}
	// 删除不存在的 blob 应静默成功。
	if err := l.Delete(relPath); err != nil {
		t.Fatalf("重复 Delete: %v", err)
	}
}

func TestThumbRoundtrip(t *testing.T) {
	l := newTestLocal(t)
	hash := strings.Repeat("ab", 16)
	relPath := l.ThumbPath(hash)
	if relPath == "" {
		t.Fatal("ThumbPath 不应为空")
	}
	w, err := l.CreateThumb(relPath)
	if err != nil {
		t.Fatalf("CreateThumb: %v", err)
	}
	if _, err := w.Write([]byte("jpeg-bytes")); err != nil {
		t.Fatalf("写入缩略图: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("关闭缩略图: %v", err)
	}
	reader, err := l.OpenThumb(relPath)
	if err != nil {
		t.Fatalf("OpenThumb: %v", err)
	}
	reader.(io.Closer).Close()
	if err := l.DeleteThumb(relPath); err != nil {
		t.Fatalf("DeleteThumb: %v", err)
	}
	if _, err := l.OpenThumb(relPath); err == nil {
		t.Fatal("删除后 OpenThumb 应失败")
	}
}

func TestRejectPathTraversal(t *testing.T) {
	l := newTestLocal(t)
	if _, err := l.Open("../etc/passwd"); err == nil {
		t.Fatal("越界路径应当被拒绝")
	}
	if _, err := l.ChunkDir("../uploads"); err == nil {
		t.Fatal("非法 upload_id 应当被拒绝")
	}
}

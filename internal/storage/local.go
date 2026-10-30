package storage

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/iokr/novadrive/internal/config"
)

// tmpSubDir 临时文件子目录名。
const tmpSubDir = "_tmp"

// uploadIDPattern 上传会话 ID 合法字符（服务端生成的 UUID）。
var uploadIDPattern = regexp.MustCompile(`^[A-Za-z0-9-]{8,64}$`)

// chunkFilePattern 分片文件名模式。
var chunkFilePattern = regexp.MustCompile(`^(\d+)\.part$`)

// Local 本地文件系统存储实现。
type Local struct {
	blobRoot   string
	thumbRoot  string
	uploadRoot string
	tmpRoot    string
	bufSize    int
}

// NewLocal 创建本地存储实现，自动创建所需目录。
func NewLocal(cfg *config.Config) (*Local, error) {
	blobRoot, err := filepath.Abs(cfg.Storage.Local.Root)
	if err != nil {
		return nil, fmt.Errorf("解析 blob 根目录失败: %w", err)
	}
	thumbRoot, err := filepath.Abs(cfg.Storage.Local.ThumbRoot)
	if err != nil {
		return nil, fmt.Errorf("解析缩略图根目录失败: %w", err)
	}
	uploadRoot, err := filepath.Abs(cfg.Storage.Local.UploadRoot)
	if err != nil {
		return nil, fmt.Errorf("解析分片根目录失败: %w", err)
	}
	l := &Local{
		blobRoot:   blobRoot,
		thumbRoot:  thumbRoot,
		uploadRoot: uploadRoot,
		tmpRoot:    filepath.Join(uploadRoot, tmpSubDir),
		bufSize:    cfg.Storage.Chunk.MergeBufferSize,
	}
	for _, dir := range []string{l.blobRoot, l.thumbRoot, l.uploadRoot, l.tmpRoot} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("创建存储目录 %s 失败: %w", dir, err)
		}
	}
	return l, nil
}

// Roots 返回各存储根目录（便于运维排查）。
func (l *Local) Roots() (blob, thumb, upload string) {
	return l.blobRoot, l.thumbRoot, l.uploadRoot
}

// withinRoot 判断 target 是否位于 root 内部（防目录穿越）。
func withinRoot(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return !strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel)
}

// blobAbs 由相对路径得到绝对路径并校验边界。
func (l *Local) blobAbs(relPath string) (string, error) {
	if relPath == "" {
		return "", errors.New("非法的存储路径")
	}
	clean := filepath.Clean(filepath.FromSlash(relPath))
	if filepath.IsAbs(clean) || strings.HasPrefix(clean, "..") {
		return "", errors.New("非法的存储路径")
	}
	abs := filepath.Join(l.blobRoot, clean)
	if !withinRoot(l.blobRoot, abs) {
		return "", errors.New("非法的存储路径")
	}
	return abs, nil
}

// SaveTemp 写入临时文件。
func (l *Local) SaveTemp(reader io.Reader) (string, int64, error) {
	f, err := l.CreateTempFile()
	if err != nil {
		return "", 0, err
	}
	n, err := io.Copy(f, reader)
	closeErr := f.Close()
	if err != nil {
		_ = os.Remove(f.Name())
		return "", 0, err
	}
	if closeErr != nil {
		_ = os.Remove(f.Name())
		return "", 0, closeErr
	}
	return f.Name(), n, nil
}

// CreateTempFile 在临时目录创建文件。
func (l *Local) CreateTempFile() (*os.File, error) {
	if err := os.MkdirAll(l.tmpRoot, 0o755); err != nil {
		return nil, err
	}
	return os.CreateTemp(l.tmpRoot, "tmp-*.dat")
}

// MoveToBlob 把临时文件移动到 Blob 位置（按 hash 前两位分片）。
func (l *Local) MoveToBlob(tmpPath, hash, ext string) (string, error) {
	if len(hash) < 2 {
		return "", errors.New("非法的 hash")
	}
	relPath := filepath.ToSlash(filepath.Join(hash[:2], hash))
	dst, err := l.blobAbs(relPath)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return "", err
	}
	if err := os.Rename(tmpPath, dst); err != nil {
		// 跨设备等场景回退为复制 + 删除。
		if copyErr := l.copyFile(tmpPath, dst); copyErr != nil {
			return "", err
		}
		_ = os.Remove(tmpPath)
	}
	return relPath, nil
}

// copyFile 复制文件内容。
func (l *Local) copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		_ = os.Remove(dst)
		return err
	}
	return out.Close()
}

// Open 打开 Blob 文件。
func (l *Local) Open(relPath string) (io.ReadSeeker, error) {
	abs, err := l.blobAbs(relPath)
	if err != nil {
		return nil, err
	}
	return os.Open(abs)
}

// OpenTemp 打开临时文件。
func (l *Local) OpenTemp(tmpPath string) (io.ReadSeeker, error) {
	abs := filepath.Clean(tmpPath)
	if !withinRoot(l.tmpRoot, abs) {
		return nil, errors.New("非法的临时文件路径")
	}
	return os.Open(abs)
}

// DeleteTemp 删除临时文件。
func (l *Local) DeleteTemp(tmpPath string) error {
	if tmpPath == "" {
		return nil
	}
	abs := filepath.Clean(tmpPath)
	if !withinRoot(l.tmpRoot, abs) {
		return errors.New("非法的临时文件路径")
	}
	if err := os.Remove(abs); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Delete 删除 Blob 文件。
func (l *Local) Delete(relPath string) error {
	abs, err := l.blobAbs(relPath)
	if err != nil {
		return err
	}
	if err := os.Remove(abs); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// BlobExists 判断 Blob 是否存在。
func (l *Local) BlobExists(relPath string) bool {
	abs, err := l.blobAbs(relPath)
	if err != nil {
		return false
	}
	info, err := os.Stat(abs)
	return err == nil && !info.IsDir()
}

// ThumbPath 计算缩略图相对路径。
func (l *Local) ThumbPath(hash string) string {
	if len(hash) < 2 {
		return ""
	}
	return filepath.ToSlash(filepath.Join(hash[:2], hash+".jpg"))
}

// thumbAbs 缩略图绝对路径。
func (l *Local) thumbAbs(relPath string) (string, error) {
	if relPath == "" {
		return "", errors.New("非法的缩略图路径")
	}
	clean := filepath.Clean(filepath.FromSlash(relPath))
	if filepath.IsAbs(clean) || strings.HasPrefix(clean, "..") {
		return "", errors.New("非法的缩略图路径")
	}
	abs := filepath.Join(l.thumbRoot, clean)
	if !withinRoot(l.thumbRoot, abs) {
		return "", errors.New("非法的缩略图路径")
	}
	return abs, nil
}

// CreateThumb 创建缩略图文件。
func (l *Local) CreateThumb(relPath string) (io.WriteCloser, error) {
	abs, err := l.thumbAbs(relPath)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return nil, err
	}
	return os.Create(abs)
}

// OpenThumb 打开缩略图。
func (l *Local) OpenThumb(relPath string) (io.ReadSeeker, error) {
	abs, err := l.thumbAbs(relPath)
	if err != nil {
		return nil, err
	}
	return os.Open(abs)
}

// DeleteThumb 删除缩略图。
func (l *Local) DeleteThumb(relPath string) error {
	abs, err := l.thumbAbs(relPath)
	if err != nil {
		return err
	}
	if err := os.Remove(abs); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// ChunkDir 返回（并创建）分片目录。
func (l *Local) ChunkDir(uploadID string) (string, error) {
	if !uploadIDPattern.MatchString(uploadID) {
		return "", errors.New("非法的上传会话 ID")
	}
	dir := filepath.Join(l.uploadRoot, uploadID)
	if !withinRoot(l.uploadRoot, dir) {
		return "", errors.New("非法的上传会话 ID")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// chunkPath 分片文件绝对路径。
func (l *Local) chunkPath(uploadID string, index int) (string, error) {
	dir, err := l.ChunkDir(uploadID)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, strconv.Itoa(index)+".part"), nil
}

// SaveChunk 覆盖写入分片：先写临时文件再原子替换。
func (l *Local) SaveChunk(uploadID string, index int, reader io.Reader) (int64, error) {
	dir, err := l.ChunkDir(uploadID)
	if err != nil {
		return 0, err
	}
	final := filepath.Join(dir, strconv.Itoa(index)+".part")
	tmp := filepath.Join(dir, strconv.Itoa(index)+".part.tmp")
	f, err := os.Create(tmp)
	if err != nil {
		return 0, err
	}
	n, copyErr := io.Copy(f, reader)
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return n, copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return n, closeErr
	}
	if err := os.Rename(tmp, final); err != nil {
		_ = os.Remove(tmp)
		return n, err
	}
	return n, nil
}

// ListChunks 扫描已上传分片索引。
func (l *Local) ListChunks(uploadID string) ([]int, error) {
	dir, err := l.ChunkDir(uploadID)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	indices := make([]int, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		m := chunkFilePattern.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		idx, convErr := strconv.Atoi(m[1])
		if convErr != nil {
			continue
		}
		indices = append(indices, idx)
	}
	sort.Ints(indices)
	return indices, nil
}

// MergeChunks 按序合并分片，边合并边计算 MD5。
func (l *Local) MergeChunks(uploadID string, totalChunks int, dst io.Writer) (string, int64, error) {
	h := md5.New()
	writer := io.MultiWriter(dst, h)
	bufSize := l.bufSize
	if bufSize <= 0 {
		bufSize = 4 << 20
	}
	buf := make([]byte, bufSize)
	var total int64
	for i := 0; i < totalChunks; i++ {
		path, err := l.chunkPath(uploadID, i)
		if err != nil {
			return "", total, err
		}
		f, err := os.Open(path)
		if err != nil {
			return "", total, fmt.Errorf("分片 %d 缺失: %w", i, err)
		}
		n, copyErr := io.CopyBuffer(writer, f, buf)
		closeErr := f.Close()
		total += n
		if copyErr != nil {
			return "", total, copyErr
		}
		if closeErr != nil {
			return "", total, closeErr
		}
	}
	return hex.EncodeToString(h.Sum(nil)), total, nil
}

// DeleteChunkDir 删除分片目录。
func (l *Local) DeleteChunkDir(uploadID string) error {
	dir, err := l.ChunkDir(uploadID)
	if err != nil {
		return err
	}
	return os.RemoveAll(dir)
}

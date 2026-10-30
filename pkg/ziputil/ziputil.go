// Package ziputil 提供流式 zip 写入工具（不落临时文件）。
package ziputil

import (
	"archive/zip"
	"io"
	"io/fs"
	"time"
)

// Streamer 包装 archive/zip.Writer，限制条目数量并统一 UTF-8 文件名。
type Streamer struct {
	zw         *zip.Writer
	maxEntries int
	count      int
}

// NewStreamer 创建流式 zip 写入器，maxEntries <= 0 表示不限制条目数。
func NewStreamer(w io.Writer, maxEntries int) *Streamer {
	return &Streamer{zw: zip.NewWriter(w), maxEntries: maxEntries}
}

// Count 已写入的条目数。
func (s *Streamer) Count() int { return s.count }

// addHeader 写入条目头并累计计数。
func (s *Streamer) addHeader(name string, mode fs.FileMode, modTime time.Time) (io.Writer, error) {
	if s.maxEntries > 0 && s.count >= s.maxEntries {
		return nil, ErrTooManyEntries
	}
	if modTime.IsZero() {
		modTime = time.Now()
	}
	h := &zip.FileHeader{
		Name:     name,
		Method:   zip.Deflate,
		Modified: modTime,
	}
	h.SetMode(mode)
	// 强制按 UTF-8 标注文件名，保证中文名在各平台正确解压。
	h.NonUTF8 = false
	w, err := s.zw.CreateHeader(h)
	if err != nil {
		return nil, err
	}
	s.count++
	return w, nil
}

// AddDir 写入目录条目（name 需为已带结尾斜杠的相对路径）。
func (s *Streamer) AddDir(name string, modTime time.Time) error {
	_, err := s.addHeader(name, fs.ModeDir|0o755, modTime)
	return err
}

// AddFile 写入文件条目并拷贝内容。
func (s *Streamer) AddFile(name string, r io.Reader, modTime time.Time) error {
	w, err := s.addHeader(name, 0o644, modTime)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, r)
	return err
}

// Close 结束 zip 写入（必须调用以写入中央目录）。
func (s *Streamer) Close() error {
	return s.zw.Close()
}

// ErrTooManyEntries 条目数超限。
var ErrTooManyEntries = errTooManyEntries{}

type errTooManyEntries struct{}

func (errTooManyEntries) Error() string { return "压缩包条目数超限" }

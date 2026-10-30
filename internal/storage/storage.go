// Package storage 定义物理存储抽象，屏蔽底层实现（当前仅本地文件系统）。
package storage

import (
	"io"
	"os"
)

// Storage 物理存储接口：Blob、分片与缩略图。
type Storage interface {
	// SaveTemp 将数据写入临时目录，返回临时文件绝对路径与字节数。
	SaveTemp(reader io.Reader) (tmpPath string, size int64, err error)
	// CreateTempFile 在临时目录创建可写文件（用于分片合并产物）。
	CreateTempFile() (*os.File, error)
	// MoveToBlob 把临时文件落库到 Blob 位置，返回相对路径。
	MoveToBlob(tmpPath, hash, ext string) (relPath string, err error)
	// Open 打开 Blob 文件。
	Open(relPath string) (io.ReadSeeker, error)
	// OpenTemp 打开临时文件（仅允许临时目录内路径）。
	OpenTemp(tmpPath string) (io.ReadSeeker, error)
	// DeleteTemp 删除临时文件。
	DeleteTemp(tmpPath string) error
	// Delete 删除 Blob 文件。
	Delete(relPath string) error
	// BlobExists 判断 Blob 文件是否存在。
	BlobExists(relPath string) bool

	// ThumbPath 计算缩略图相对路径（不含磁盘前缀）。
	ThumbPath(hash string) string
	// CreateThumb 创建缩略图文件写入器。
	CreateThumb(relPath string) (io.WriteCloser, error)
	// OpenThumb 打开缩略图文件。
	OpenThumb(relPath string) (io.ReadSeeker, error)
	// DeleteThumb 删除缩略图。
	DeleteThumb(relPath string) error

	// ChunkDir 返回（并创建）会话分片目录的绝对路径。
	ChunkDir(uploadID string) (absDir string, err error)
	// SaveChunk 写入单个分片（覆盖写入，幂等）。
	SaveChunk(uploadID string, index int, reader io.Reader) (size int64, err error)
	// ListChunks 扫描已上传分片索引（升序）。
	ListChunks(uploadID string) (indices []int, err error)
	// MergeChunks 按序把分片写入 dst，返回合并内容的 MD5 与总字节数。
	MergeChunks(uploadID string, totalChunks int, dst io.Writer) (md5 string, size int64, err error)
	// DeleteChunkDir 删除会话分片目录。
	DeleteChunkDir(uploadID string) error
}

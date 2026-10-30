package handler

import (
	"bufio"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/iokr/novadrive/internal/model"
	"github.com/iokr/novadrive/internal/service"
	"github.com/iokr/novadrive/pkg/fileutil"
	"github.com/iokr/novadrive/pkg/ziputil"
	"go.uber.org/zap"
)

// blobStream 描述一次文件流式响应。
type blobStream struct {
	name    string
	mime    string
	reader  io.ReadSeeker
	modTime time.Time
	etag    string
	cache   string
	inline  bool
}

// serveBlob 以 http.ServeContent 语义输出内容，自动支持 Range / ETag / Last-Modified。
func serveBlob(c *gin.Context, s blobStream) {
	if closer, ok := s.reader.(io.Closer); ok {
		defer closer.Close()
	}
	mimeType := s.mime
	if mimeType == "" {
		mimeType = fileutil.MimeByExt(fileutil.Ext(s.name))
	}
	header := c.Writer.Header()
	header.Set("Content-Type", mimeType)
	header.Set("Content-Disposition", contentDisposition(s.name, s.inline))
	header.Set("X-Content-Type-Options", "nosniff")
	if s.etag != "" {
		header.Set("ETag", s.etag)
	}
	if s.cache != "" {
		header.Set("Cache-Control", s.cache)
	}
	http.ServeContent(c.Writer, c.Request, s.name, s.modTime, s.reader)
}

// contentDisposition 生成同时兼容 ASCII 与 UTF-8 文件名的响应头。
func contentDisposition(name string, inline bool) string {
	kind := "attachment"
	if inline {
		kind = "inline"
	}
	if name == "" {
		name = "download"
	}
	return kind + `; filename="` + fileutil.FallbackName(name) + `"; filename*=UTF-8''` + url.PathEscape(name)
}

// zipSource 打开 Blob 内容，供 zip 流读取。
type zipSource func(*model.Blob) (io.ReadSeeker, error)

// streamZip 流式输出 zip：边打包边写响应，不落临时文件。
func streamZip(c *gin.Context, entries []service.ZipEntry, filename string, open zipSource, bufferSize, maxEntries int, log *zap.Logger) {
	if log == nil {
		log = zap.NewNop()
	}
	header := c.Writer.Header()
	header.Set("Content-Type", "application/zip")
	header.Set("Content-Disposition", contentDisposition(filename, false))
	header.Set("X-Content-Type-Options", "nosniff")
	c.Status(http.StatusOK)

	if bufferSize <= 0 {
		bufferSize = 32 << 10
	}
	buffered := bufio.NewWriterSize(c.Writer, bufferSize)
	streamer := ziputil.NewStreamer(buffered, maxEntries)
	for i := range entries {
		entry := entries[i]
		if entry.Dir || entry.Blob == nil {
			if err := streamer.AddDir(entry.Name, entry.ModTime); err != nil {
				log.Warn("写入 zip 目录条目失败", zap.String("name", entry.Name), zap.Error(err))
				return
			}
			continue
		}
		reader, err := open(entry.Blob)
		if err != nil {
			log.Warn("读取 zip 条目内容失败", zap.String("name", entry.Name), zap.Error(err))
			continue
		}
		err = streamer.AddFile(entry.Name, reader, entry.ModTime)
		if closer, ok := reader.(io.Closer); ok {
			_ = closer.Close()
		}
		if err != nil {
			log.Warn("写入 zip 文件条目失败", zap.String("name", entry.Name), zap.Error(err))
			return
		}
	}
	if err := streamer.Close(); err != nil {
		log.Warn("关闭 zip 流失败", zap.Error(err))
		return
	}
	if err := buffered.Flush(); err != nil {
		log.Warn("刷新 zip 响应缓冲失败", zap.Error(err))
	}
}

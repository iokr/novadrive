package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Logger 访问日志中间件：记录方法、路径、状态码、耗时与客户端 IP。
func Logger(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		method := c.Request.Method
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		if log == nil {
			return
		}
		status := c.Writer.Status()
		fields := []zap.Field{
			zap.Int("status", status),
			zap.String("method", method),
			zap.String("path", path),
			zap.String("ip", c.ClientIP()),
			zap.Int("size", c.Writer.Size()),
			zap.Duration("latency", time.Since(start)),
		}
		if query != "" {
			fields = append(fields, zap.String("query", query))
		}
		if uid := CurrentUserID(c); uid != 0 {
			fields = append(fields, zap.Uint64("user_id", uid))
		}
		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("errors", c.Errors.String()))
		}
		switch {
		case status >= 500:
			log.Error("http", fields...)
		case status >= 400:
			log.Warn("http", fields...)
		default:
			log.Info("http", fields...)
		}
	}
}

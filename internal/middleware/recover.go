package middleware

import (
	"errors"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/iokr/novadrive/pkg/resp"
	"go.uber.org/zap"
)

// Recover 捕获 panic，记录堆栈并返回统一内部错误响应。
func Recover(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			r := recover()
			if r == nil {
				return
			}
			// http.ErrAbortHandler 是标准库用于静默中断响应的信号，需继续向外抛出。
			if err, ok := r.(error); ok && errors.Is(err, http.ErrAbortHandler) {
				panic(r)
			}
			if log != nil {
				log.Error("panic recovered",
					zap.Any("error", r),
					zap.String("method", c.Request.Method),
					zap.String("path", c.Request.URL.Path),
					zap.ByteString("stack", debug.Stack()))
			}
			if c.Writer.Written() {
				c.Abort()
				return
			}
			resp.Abort(c, resp.CodeInternal, "服务器内部错误")
		}()
		c.Next()
	}
}

package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORS 跨域中间件：回显请求来源，预检请求直接返回 204。
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			origin = "*"
		}
		header := c.Writer.Header()
		header.Set("Access-Control-Allow-Origin", origin)
		header.Set("Vary", "Origin")
		header.Set("Access-Control-Allow-Credentials", "true")
		header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		header.Set("Access-Control-Allow-Headers",
			"Authorization, Content-Type, X-Requested-With, Range, X-Share-Key")
		header.Set("Access-Control-Expose-Headers",
			"Content-Length, Content-Range, Content-Disposition, ETag")
		header.Set("Access-Control-Max-Age", "86400")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

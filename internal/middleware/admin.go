package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/iokr/novadrive/internal/constants"
	"github.com/iokr/novadrive/pkg/resp"
)

// Admin 管理员权限校验（需在 Auth 之后使用）。
func Admin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if CurrentRole(c) != constants.RoleAdmin {
			resp.Abort(c, resp.CodeNotAdmin, "非管理员")
			return
		}
		c.Next()
	}
}

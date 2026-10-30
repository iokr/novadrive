// Package middleware 提供认证、权限、跨域、日志与异常恢复中间件。
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/iokr/novadrive/internal/constants"
	"github.com/iokr/novadrive/internal/repository"
	"github.com/iokr/novadrive/pkg/jwt"
	"github.com/iokr/novadrive/pkg/resp"
)

const (
	// ctxUserID 上下文中的用户 ID 键。
	ctxUserID = "novadrive_user_id"
	// ctxUserRole 上下文中的角色键。
	ctxUserRole = "novadrive_user_role"
)

// Auth JWT 认证中间件：支持 Authorization: Bearer 与 ?token= 两种方式。
func Auth(jwtMgr *jwt.Manager, userRepo *repository.UserRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			resp.Abort(c, resp.CodeUnauthorized, "未登录或 token 失效")
			return
		}
		claims, err := jwtMgr.Parse(token)
		if err != nil {
			resp.Abort(c, resp.CodeUnauthorized, "未登录或 token 失效")
			return
		}
		user, err := userRepo.FindByID(claims.UserID)
		if err != nil {
			resp.Abort(c, resp.CodeUnauthorized, "未登录或 token 失效")
			return
		}
		if user.Status != constants.UserStatusNormal {
			resp.Abort(c, resp.CodeForbidden, "账号已被禁用")
			return
		}
		c.Set(ctxUserID, user.ID)
		c.Set(ctxUserRole, user.Role)
		c.Next()
	}
}

// extractToken 从请求头或查询参数中提取 JWT。
func extractToken(c *gin.Context) string {
	if header := c.GetHeader("Authorization"); header != "" {
		if len(header) > 7 && strings.EqualFold(header[:7], "bearer ") {
			return strings.TrimSpace(header[7:])
		}
		return strings.TrimSpace(header)
	}
	return strings.TrimSpace(c.Query(constants.AccessTokenQueryKey))
}

// CurrentUserID 获取当前登录用户 ID。
func CurrentUserID(c *gin.Context) uint64 {
	if v, ok := c.Get(ctxUserID); ok {
		if id, ok := v.(uint64); ok {
			return id
		}
	}
	return 0
}

// CurrentRole 获取当前登录用户角色。
func CurrentRole(c *gin.Context) int8 {
	if v, ok := c.Get(ctxUserRole); ok {
		if role, ok := v.(int8); ok {
			return role
		}
	}
	return constants.RoleUser
}

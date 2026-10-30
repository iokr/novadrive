// Package handler 实现 HTTP 接口层：参数绑定、业务调用与统一响应。
package handler

import (
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/iokr/novadrive/internal/middleware"
	"github.com/iokr/novadrive/pkg/errs"
	"github.com/iokr/novadrive/pkg/resp"
	"go.uber.org/zap"
)

// fail 统一错误响应：业务错误按原业务码返回，未知错误记录日志后返回 5000。
func fail(c *gin.Context, log *zap.Logger, err error) {
	code, message := errs.From(err)
	if code == resp.CodeInternal && log != nil {
		log.Error("请求处理失败",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Error(err))
	}
	resp.Fail(c, code, message)
}

// bindJSON 解析 JSON 请求体，失败时写出 1001。
func bindJSON(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		resp.Fail(c, resp.CodeBadParam, "参数错误："+err.Error())
		return false
	}
	return true
}

// bindJSONOptional 解析可选 JSON 请求体（允许空 body）。
func bindJSONOptional(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil && !errors.Is(err, io.EOF) {
		resp.Fail(c, resp.CodeBadParam, "参数错误："+err.Error())
		return false
	}
	return true
}

// pathID 解析路径参数中的无符号整数 ID。
func pathID(c *gin.Context, name string) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, resp.CodeBadParam, "非法的 ID 参数")
		return 0, false
	}
	return id, true
}

// parseUint 解析无符号整数，失败时 ok 为 false。
func parseUint(raw string) (uint64, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}
	v, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

// queryUint64 读取查询参数中的无符号整数，缺省或非法时返回 def。
func queryUint64(c *gin.Context, key string, def uint64) uint64 {
	if v, ok := parseUint(c.Query(key)); ok {
		return v
	}
	return def
}

// queryInt 读取查询参数中的整数，缺省或非法时返回 def。
func queryInt(c *gin.Context, key string, def int) int {
	raw := c.Query(key)
	if raw == "" {
		return def
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return v
}

// queryInt8 读取查询参数中的 8 位整数，缺省或非法时返回 def。
func queryInt8(c *gin.Context, key string, def int8) int8 {
	raw := c.Query(key)
	if raw == "" {
		return def
	}
	v, err := strconv.ParseInt(raw, 10, 8)
	if err != nil {
		return def
	}
	return int8(v)
}

// formUint64 读取表单字段中的无符号整数，缺省或非法时返回 def。
func formUint64(c *gin.Context, key string, def uint64) uint64 {
	if v, ok := parseUint(c.PostForm(key)); ok {
		return v
	}
	return def
}

// userID 返回当前登录用户 ID（未登录时为 0）。
func userID(c *gin.Context) uint64 { return middleware.CurrentUserID(c) }

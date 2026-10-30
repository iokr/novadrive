// Package resp 定义统一响应结构与业务错误码。
package resp

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// 业务错误码。
const (
	CodeOK                 = 0
	CodeBadParam           = 1001
	CodeUnauthorized       = 1002
	CodeForbidden          = 1003
	CodeNotAdmin           = 1004
	CodeUserExists         = 2001
	CodeCredentialInvalid  = 2002
	CodeFileNotFound       = 3001
	CodeQuotaExceeded      = 3002
	CodeNameConflict       = 3003
	CodeUploadLimit        = 3004
	CodeZipEntriesExceeded = 3005
	CodeSessionNotFound    = 3006
	CodeChunkInvalid       = 3007
	CodeChunksIncomplete   = 3008
	CodeMD5Mismatch        = 3009
	CodeShareInvalid       = 4001
	CodeSharePasswordWrong = 4002
	CodeInternal           = 5000
)

// Body 统一响应体 {code, message, data}。
type Body struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// OK 返回成功响应。
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Body{Code: CodeOK, Message: "ok", Data: data})
}

// Fail 返回业务失败响应（HTTP 200，业务码在 body 中）。
func Fail(c *gin.Context, code int, message string) {
	c.JSON(http.StatusOK, Body{Code: code, Message: message})
}

// Abort 返回失败响应并中断后续处理。
func Abort(c *gin.Context, code int, message string) {
	c.AbortWithStatusJSON(http.StatusOK, Body{Code: code, Message: message})
}

// Raw 直接写出响应体，便于 NoRoute 等场景。
func Raw(c *gin.Context, status, code int, message string) {
	c.JSON(status, Body{Code: code, Message: message})
}

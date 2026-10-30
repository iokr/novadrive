// Package errs 定义携带业务码的错误类型与常用错误值。
package errs

import (
	"errors"
	"fmt"

	"github.com/iokr/novadrive/pkg/resp"
)

// Error 业务错误：Code 为对外业务码，Message 为提示文案。
type Error struct {
	Code    int
	Message string
}

// Error 实现 error 接口。
func (e *Error) Error() string { return e.Message }

// WithMessage 基于已有错误生成自定义文案的副本。
func (e *Error) WithMessage(format string, args ...any) *Error {
	return &Error{Code: e.Code, Message: fmt.Sprintf(format, args...)}
}

// New 创建业务错误。
func New(code int, message string) *Error {
	return &Error{Code: code, Message: message}
}

// From 提取错误中的业务码，非业务错误统一映射为内部错误。
func From(err error) (int, string) {
	if err == nil {
		return resp.CodeOK, "ok"
	}
	var e *Error
	if errors.As(err, &e) {
		return e.Code, e.Message
	}
	return resp.CodeInternal, "服务器内部错误"
}

// 预定义业务错误。
var (
	ErrBadParam          = New(resp.CodeBadParam, "参数错误")
	ErrUnauthorized      = New(resp.CodeUnauthorized, "未登录或 token 失效")
	ErrForbidden         = New(resp.CodeForbidden, "无权限")
	ErrNotAdmin          = New(resp.CodeNotAdmin, "非管理员")
	ErrUserExists        = New(resp.CodeUserExists, "用户名或邮箱已存在")
	ErrCredential        = New(resp.CodeCredentialInvalid, "用户名或密码错误")
	ErrUserDisabled      = New(resp.CodeForbidden, "账号已被禁用")
	ErrFileNotFound      = New(resp.CodeFileNotFound, "文件不存在")
	ErrQuotaExceeded     = New(resp.CodeQuotaExceeded, "存储空间不足")
	ErrNameConflict      = New(resp.CodeNameConflict, "同名文件已存在")
	ErrUploadLimit       = New(resp.CodeUploadLimit, "上传数量或大小超限")
	ErrZipEntries        = New(resp.CodeZipEntriesExceeded, "压缩包条目数超限")
	ErrSessionNotFound   = New(resp.CodeSessionNotFound, "上传会话不存在或已失效")
	ErrChunkInvalid      = New(resp.CodeChunkInvalid, "分片索引越界或分片大小不匹配")
	ErrChunksIncomplete  = New(resp.CodeChunksIncomplete, "分片不完整，无法合并")
	ErrMD5Mismatch       = New(resp.CodeMD5Mismatch, "合并后 MD5 与声明不一致")
	ErrShareInvalid      = New(resp.CodeShareInvalid, "分享不存在或已失效")
	ErrSharePassword     = New(resp.CodeSharePasswordWrong, "提取密码错误")
	ErrInternal          = New(resp.CodeInternal, "服务器内部错误")
	ErrExtNotAllowed     = New(resp.CodeBadParam, "扩展名不允许")
	ErrFileTooLarge      = New(resp.CodeUploadLimit, "文件大小超出限制")
	ErrInvalidTarget     = New(resp.CodeBadParam, "非法的目标目录")
	ErrInvalidName       = New(resp.CodeBadParam, "非法的名称")
	ErrSameName          = New(resp.CodeNameConflict, "原名与新名相同")
	ErrInvalidMove       = New(resp.CodeBadParam, "不能移动到自身或其子目录")
	ErrPasswordIncorrect = New(resp.CodeBadParam, "原密码错误")
)

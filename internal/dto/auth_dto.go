// Package dto 定义接口出入参结构。
package dto

import (
	"strings"
	"time"

	"github.com/iokr/novadrive/internal/model"
)

// RegisterReq 注册请求。
type RegisterReq struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	Email    string `json:"email" binding:"required,email,max=128"`
	Password string `json:"password" binding:"required,min=6,max=64"`
}

// RegisterResp 注册响应。
type RegisterResp struct {
	ID       uint64 `json:"id"`
	Username string `json:"username"`
}

// LoginReq 登录请求：支持 username / email / account 三种字段名。
type LoginReq struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Account  string `json:"account"`
	Password string `json:"password" binding:"required"`
}

// Identifier 返回用户输入的登录标识（用户名或邮箱）。
func (r *LoginReq) Identifier() string {
	for _, v := range []string{r.Username, r.Email, r.Account} {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// UserInfo 用户信息。
type UserInfo struct {
	ID           uint64     `json:"id"`
	Username     string     `json:"username"`
	Email        string     `json:"email"`
	Nickname     string     `json:"nickname"`
	Avatar       string     `json:"avatar"`
	Role         int8       `json:"role"`
	Status       int8       `json:"status"`
	StorageUsed  int64      `json:"storage_used"`
	StorageQuota int64      `json:"storage_quota"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// LoginResp 登录响应。
type LoginResp struct {
	Token     string   `json:"token"`
	ExpiresIn int64    `json:"expires_in"`
	User      UserInfo `json:"user"`
}

// UpdateProfileReq 修改昵称 / 头像。
type UpdateProfileReq struct {
	Nickname *string `json:"nickname"`
	Avatar   *string `json:"avatar"`
}

// ChangePasswordReq 修改密码。
type ChangePasswordReq struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6,max=64"`
}

// NewUserInfo 由模型转换用户信息 DTO。
func NewUserInfo(u *model.User) UserInfo {
	return UserInfo{
		ID:           u.ID,
		Username:     u.Username,
		Email:        u.Email,
		Nickname:     u.Nickname,
		Avatar:       u.Avatar,
		Role:         u.Role,
		Status:       u.Status,
		StorageUsed:  u.StorageUsed,
		StorageQuota: u.StorageQuota,
		LastLoginAt:  u.LastLoginAt,
		CreatedAt:    u.CreatedAt,
	}
}

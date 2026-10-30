package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/iokr/novadrive/internal/dto"
	"github.com/iokr/novadrive/internal/service"
	"github.com/iokr/novadrive/pkg/resp"
	"go.uber.org/zap"
)

// AuthHandler 认证与个人资料接口。
type AuthHandler struct {
	svc *service.AuthService
	log *zap.Logger
}

// NewAuthHandler 创建认证 handler。
func NewAuthHandler(svc *service.AuthService, log *zap.Logger) *AuthHandler {
	return &AuthHandler{svc: svc, log: log}
}

// Register 注册新用户。
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterReq
	if !bindJSON(c, &req) {
		return
	}
	res, err := h.svc.Register(&req)
	if err != nil {
		fail(c, h.log, err)
		return
	}
	resp.OK(c, res)
}

// Login 登录并签发 token。
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginReq
	if !bindJSON(c, &req) {
		return
	}
	res, err := h.svc.Login(&req)
	if err != nil {
		fail(c, h.log, err)
		return
	}
	resp.OK(c, res)
}

// Profile 当前用户信息。
func (h *AuthHandler) Profile(c *gin.Context) {
	res, err := h.svc.Profile(userID(c))
	if err != nil {
		fail(c, h.log, err)
		return
	}
	resp.OK(c, res)
}

// UpdateProfile 修改昵称 / 头像。
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	var req dto.UpdateProfileReq
	if !bindJSON(c, &req) {
		return
	}
	res, err := h.svc.UpdateProfile(userID(c), &req)
	if err != nil {
		fail(c, h.log, err)
		return
	}
	resp.OK(c, res)
}

// ChangePassword 修改密码。
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req dto.ChangePasswordReq
	if !bindJSON(c, &req) {
		return
	}
	if err := h.svc.ChangePassword(userID(c), &req); err != nil {
		fail(c, h.log, err)
		return
	}
	resp.OK(c, gin.H{"changed": true})
}

package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/iokr/novadrive/internal/service"
	"github.com/iokr/novadrive/pkg/resp"
	"go.uber.org/zap"
)

// UserHandler 用户空间相关接口。
type UserHandler struct {
	svc *service.UserService
	log *zap.Logger
}

// NewUserHandler 创建用户 handler。
func NewUserHandler(svc *service.UserService, log *zap.Logger) *UserHandler {
	return &UserHandler{svc: svc, log: log}
}

// Storage 侧边栏空间条数据（已用 / 配额 / 文件数）。
func (h *UserHandler) Storage(c *gin.Context) {
	res, err := h.svc.Space(userID(c))
	if err != nil {
		fail(c, h.log, err)
		return
	}
	resp.OK(c, res)
}

// Overview 空间分析（按类型聚合）。
func (h *UserHandler) Overview(c *gin.Context) {
	res, err := h.svc.Overview(userID(c))
	if err != nil {
		fail(c, h.log, err)
		return
	}
	resp.OK(c, res)
}

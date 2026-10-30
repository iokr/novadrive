package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/iokr/novadrive/internal/dto"
	"github.com/iokr/novadrive/internal/service"
	"github.com/iokr/novadrive/pkg/resp"
	"go.uber.org/zap"
)

// AdminHandler 管理员后台接口。
type AdminHandler struct {
	svc *service.AdminService
	log *zap.Logger
}

// NewAdminHandler 创建管理员 handler。
func NewAdminHandler(svc *service.AdminService, log *zap.Logger) *AdminHandler {
	return &AdminHandler{svc: svc, log: log}
}

// Users 用户列表（分页 + 关键字）。
func (h *AdminHandler) Users(c *gin.Context) {
	res, err := h.svc.ListUsers(
		queryInt(c, "page", 1),
		queryInt(c, "size", 20),
		c.Query("keyword"),
	)
	if err != nil {
		fail(c, h.log, err)
		return
	}
	resp.OK(c, res)
}

// SetStatus 启用 / 禁用用户。
func (h *AdminHandler) SetStatus(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var req dto.SetStatusReq
	if !bindJSON(c, &req) {
		return
	}
	if err := h.svc.SetStatus(id, req.Status); err != nil {
		fail(c, h.log, err)
		return
	}
	resp.OK(c, gin.H{"status": req.Status})
}

// SetQuota 调整用户配额。
func (h *AdminHandler) SetQuota(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var req dto.SetQuotaReq
	if !bindJSON(c, &req) {
		return
	}
	if err := h.svc.SetQuota(id, req.StorageQuota); err != nil {
		fail(c, h.log, err)
		return
	}
	resp.OK(c, gin.H{"storage_quota": req.StorageQuota})
}

// Stats 系统统计。
func (h *AdminHandler) Stats(c *gin.Context) {
	res, err := h.svc.Stats()
	if err != nil {
		fail(c, h.log, err)
		return
	}
	resp.OK(c, res)
}

// UploadSessions 上传会话监控（status 缺省表示全部状态）。
func (h *AdminHandler) UploadSessions(c *gin.Context) {
	var status *int8
	if raw := c.Query("status"); raw != "" {
		value := queryInt8(c, "status", -1)
		if value >= 0 {
			status = &value
		}
	}
	res, err := h.svc.UploadSessions(status, queryInt(c, "limit", 200))
	if err != nil {
		fail(c, h.log, err)
		return
	}
	resp.OK(c, res)
}

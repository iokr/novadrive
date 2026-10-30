package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/iokr/novadrive/internal/config"
	"github.com/iokr/novadrive/internal/constants"
	"github.com/iokr/novadrive/internal/dto"
	"github.com/iokr/novadrive/internal/model"
	"github.com/iokr/novadrive/internal/service"
	"github.com/iokr/novadrive/pkg/errs"
	"github.com/iokr/novadrive/pkg/resp"
	"go.uber.org/zap"
)

// ShareHandler 分享相关接口（含公开访问）。
type ShareHandler struct {
	cfg      *config.Config
	log      *zap.Logger
	shareSvc *service.ShareService
	fileSvc  *service.FileService
	blobSvc  *service.BlobService
}

// NewShareHandler 创建分享 handler。
func NewShareHandler(
	cfg *config.Config,
	log *zap.Logger,
	shareSvc *service.ShareService,
	fileSvc *service.FileService,
	blobSvc *service.BlobService,
) *ShareHandler {
	return &ShareHandler{cfg: cfg, log: log, shareSvc: shareSvc, fileSvc: fileSvc, blobSvc: blobSvc}
}

// Create 创建分享。
func (h *ShareHandler) Create(c *gin.Context) {
	var req dto.CreateShareReq
	if !bindJSON(c, &req) {
		return
	}
	res, err := h.shareSvc.Create(userID(c), &req)
	if err != nil {
		fail(c, h.log, err)
		return
	}
	resp.OK(c, res)
}

// List 我的分享列表。
func (h *ShareHandler) List(c *gin.Context) {
	res, err := h.shareSvc.List(userID(c))
	if err != nil {
		fail(c, h.log, err)
		return
	}
	resp.OK(c, res)
}

// Cancel 取消分享。
func (h *ShareHandler) Cancel(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	if err := h.shareSvc.Cancel(userID(c), id); err != nil {
		fail(c, h.log, err)
		return
	}
	resp.OK(c, gin.H{"canceled": true})
}

// PublicInfo 公开分享基本信息（无需鉴权）。
func (h *ShareHandler) PublicInfo(c *gin.Context) {
	res, err := h.shareSvc.PublicInfo(c.Param("token"))
	if err != nil {
		fail(c, h.log, err)
		return
	}
	resp.OK(c, res)
}

// Verify 校验提取码并返回 access_key。
func (h *ShareHandler) Verify(c *gin.Context) {
	var req struct {
		Password string `json:"password"`
	}
	if !bindJSONOptional(c, &req) {
		return
	}
	res, err := h.shareSvc.Verify(c.Param("token"), req.Password)
	if err != nil {
		fail(c, h.log, err)
		return
	}
	resp.OK(c, res)
}

// PublicFiles 目录分享内容浏览。
func (h *ShareHandler) PublicFiles(c *gin.Context) {
	res, err := h.shareSvc.PublicFiles(
		c.Param("token"),
		accessKey(c),
		queryUint64(c, "parent_id", 0),
	)
	if err != nil {
		fail(c, h.log, err)
		return
	}
	resp.OK(c, res)
}

// Preview 分享内容预览（单文件，支持 Range）。
func (h *ShareHandler) Preview(c *gin.Context) {
	fileID, ok := pathID(c, "file_id")
	if !ok {
		return
	}
	_, file, blob, err := h.shareSvc.PublicTarget(c.Param("token"), accessKey(c), fileID)
	if err != nil {
		fail(c, h.log, err)
		return
	}
	if blob == nil {
		fail(c, h.log, errs.ErrBadParam.WithMessage("目录不支持预览"))
		return
	}
	h.streamBlobContent(c, file.Name, blob, true)
}

// Download 分享下载：单文件直接输出，目录流式 zip。
func (h *ShareHandler) Download(c *gin.Context) {
	info, file, blob, err := h.shareSvc.RecordDownload(
		c.Param("token"),
		accessKey(c),
		queryUint64(c, "file_id", 0),
	)
	if err != nil {
		fail(c, h.log, err)
		return
	}
	if blob != nil {
		h.streamBlobContent(c, file.Name, blob, false)
		return
	}
	entries, filename, err := h.fileSvc.CollectDirZip(file.UserID, file.ID)
	if err != nil {
		fail(c, h.log, err)
		return
	}
	if info != nil && info.FileName != "" {
		filename = info.FileName + ".zip"
	}
	streamZip(c, entries, filename, h.blobSvc.OpenBlob,
		h.cfg.Download.ZipBufferSize, h.cfg.Download.ZipMaxEntries, h.log)
}

// streamBlobContent 输出分享文件内容（inline 预览 / attachment 下载）。
func (h *ShareHandler) streamBlobContent(c *gin.Context, name string, blob *model.Blob, inline bool) {
	reader, err := h.blobSvc.OpenBlob(blob)
	if err != nil {
		fail(c, h.log, errs.ErrFileNotFound.WithMessage("物理文件不可用"))
		return
	}
	serveBlob(c, blobStream{
		name:    name,
		mime:    blob.MimeType,
		reader:  reader,
		modTime: blob.CreatedAt,
		etag:    `"` + blob.HashMD5 + `"`,
		cache:   "private, max-age=3600",
		inline:  inline,
	})
}

// accessKey 读取分享访问凭证：优先请求头，其次查询参数。
func accessKey(c *gin.Context) string {
	if key := c.GetHeader(constants.ShareKeyHeader); key != "" {
		return key
	}
	return c.Query("access_key")
}

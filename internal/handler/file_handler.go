package handler

import (
	"encoding/json"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/iokr/novadrive/internal/config"
	"github.com/iokr/novadrive/internal/dto"
	"github.com/iokr/novadrive/internal/service"
	"github.com/iokr/novadrive/pkg/errs"
	"github.com/iokr/novadrive/pkg/resp"
	"go.uber.org/zap"
)

// FileHandler 文件 / 目录 / 上传接口。
type FileHandler struct {
	cfg       *config.Config
	log       *zap.Logger
	fileSvc   *service.FileService
	uploadSvc *service.UploadService
	thumbSvc  *service.ThumbService
	blobSvc   *service.BlobService
}

// NewFileHandler 创建文件 handler。
func NewFileHandler(
	cfg *config.Config,
	log *zap.Logger,
	fileSvc *service.FileService,
	uploadSvc *service.UploadService,
	thumbSvc *service.ThumbService,
	blobSvc *service.BlobService,
) *FileHandler {
	return &FileHandler{
		cfg:       cfg,
		log:       log,
		fileSvc:   fileSvc,
		uploadSvc: uploadSvc,
		thumbSvc:  thumbSvc,
		blobSvc:   blobSvc,
	}
}

// List 目录浏览 / 关键字搜索。
func (h *FileHandler) List(c *gin.Context) {
	res, err := h.fileSvc.List(
		userID(c),
		queryUint64(c, "parent_id", 0),
		c.Query("keyword"),
		c.DefaultQuery("sort", "name"),
		c.DefaultQuery("order", "asc"),
	)
	if err != nil {
		fail(c, h.log, err)
		return
	}
	resp.OK(c, res)
}

// Breadcrumb 面包屑导航。
func (h *FileHandler) Breadcrumb(c *gin.Context) {
	res, err := h.fileSvc.Breadcrumb(userID(c), queryUint64(c, "parent_id", 0))
	if err != nil {
		fail(c, h.log, err)
		return
	}
	resp.OK(c, res)
}

// Recent 最近访问。
func (h *FileHandler) Recent(c *gin.Context) {
	res, err := h.fileSvc.Recent(userID(c), queryInt(c, "limit", 0))
	if err != nil {
		fail(c, h.log, err)
		return
	}
	resp.OK(c, res)
}

// Favorites 收藏列表。
func (h *FileHandler) Favorites(c *gin.Context) {
	res, err := h.fileSvc.Favorites(userID(c))
	if err != nil {
		fail(c, h.log, err)
		return
	}
	resp.OK(c, res)
}

// Stats 空间统计（按类型聚合）。
func (h *FileHandler) Stats(c *gin.Context) {
	res, err := h.fileSvc.Stats(userID(c))
	if err != nil {
		fail(c, h.log, err)
		return
	}
	resp.OK(c, res)
}

// Mkdir 新建目录。
func (h *FileHandler) Mkdir(c *gin.Context) {
	var req dto.MkdirReq
	if !bindJSON(c, &req) {
		return
	}
	res, err := h.fileSvc.Mkdir(userID(c), req.ParentID, req.Name)
	if err != nil {
		fail(c, h.log, err)
		return
	}
	resp.OK(c, res)
}

// Upload 单文件上传（可带 relative_path 逐级创建目录）。
func (h *FileHandler) Upload(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		resp.Fail(c, resp.CodeBadParam, "缺少上传文件字段 file")
		return
	}
	res, err := h.fileSvc.Upload(
		userID(c),
		formUint64(c, "parent_id", 0),
		c.PostForm("relative_path"),
		fileHeader,
	)
	if err != nil {
		fail(c, h.log, err)
		return
	}
	resp.OK(c, res)
}

// UploadBatch 多文件 / 文件夹上传（普通通道，部分成功可接受）。
func (h *FileHandler) UploadBatch(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		resp.Fail(c, resp.CodeBadParam, "解析 multipart 表单失败："+err.Error())
		return
	}
	files := form.File["files"]
	if len(files) == 0 {
		resp.Fail(c, resp.CodeBadParam, "缺少上传文件字段 files")
		return
	}
	var manifest []dto.ManifestItem
	if raw := c.PostForm("manifest"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &manifest); err != nil {
			resp.Fail(c, resp.CodeBadParam, "manifest 不是合法的 JSON 数组")
			return
		}
	}
	var parentID uint64
	if values := form.Value["parent_id"]; len(values) > 0 {
		if parsed, ok := parseUint(values[0]); ok {
			parentID = parsed
		}
	}
	res, err := h.fileSvc.UploadBatch(userID(c), parentID, files, manifest)
	if err != nil {
		fail(c, h.log, err)
		return
	}
	resp.OK(c, res)
}

// Check 秒传预检。
func (h *FileHandler) Check(c *gin.Context) {
	var req dto.CheckReq
	if !bindJSON(c, &req) {
		return
	}
	res, err := h.fileSvc.Check(userID(c), req.HashMD5, req.Size)
	if err != nil {
		fail(c, h.log, err)
		return
	}
	resp.OK(c, res)
}

// InitUpload 分片上传：初始化会话（命中秒传时直接入库）。
func (h *FileHandler) InitUpload(c *gin.Context) {
	var req dto.InitUploadReq
	if !bindJSON(c, &req) {
		return
	}
	res, err := h.uploadSvc.Init(userID(c), &req)
	if err != nil {
		fail(c, h.log, err)
		return
	}
	resp.OK(c, res)
}

// UploadChunk 分片上传：接收单个分片。
func (h *FileHandler) UploadChunk(c *gin.Context) {
	res, err := h.uploadSvc.UploadChunk(
		userID(c),
		c.Param("upload_id"),
		queryInt(c, "index", -1),
		io.Reader(c.Request.Body),
		c.Request.ContentLength,
	)
	if err != nil {
		fail(c, h.log, err)
		return
	}
	resp.OK(c, res)
}

// UploadStatus 分片上传：查询进度（断点续传）。
func (h *FileHandler) UploadStatus(c *gin.Context) {
	res, err := h.uploadSvc.Status(userID(c), c.Param("upload_id"))
	if err != nil {
		fail(c, h.log, err)
		return
	}
	resp.OK(c, res)
}

// CompleteUpload 分片上传：合并入库。
func (h *FileHandler) CompleteUpload(c *gin.Context) {
	var req dto.CompleteReq
	if !bindJSONOptional(c, &req) {
		return
	}
	res, err := h.uploadSvc.Complete(userID(c), c.Param("upload_id"), req.HashMD5)
	if err != nil {
		fail(c, h.log, err)
		return
	}
	resp.OK(c, res)
}

// CancelUpload 分片上传：取消会话。
func (h *FileHandler) CancelUpload(c *gin.Context) {
	if err := h.uploadSvc.Cancel(userID(c), c.Param("upload_id")); err != nil {
		fail(c, h.log, err)
		return
	}
	resp.OK(c, gin.H{"canceled": true})
}

// Detail 文件详情（顺带刷新最近访问时间）。
func (h *FileHandler) Detail(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	res, err := h.fileSvc.Detail(userID(c), id)
	if err != nil {
		fail(c, h.log, err)
		return
	}
	resp.OK(c, res)
}

// Thumb 缩略图：命中返回 JPEG，未生成时回落原图并触发异步生成。
func (h *FileHandler) Thumb(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	target, err := h.fileSvc.ThumbTarget(userID(c), id)
	if err != nil {
		fail(c, h.log, err)
		return
	}
	if relPath := h.thumbSvc.ThumbRelPath(target.Blob); relPath != "" {
		if file, err := h.thumbSvc.OpenThumb(relPath); err == nil {
			serveBlob(c, blobStream{
				name:    target.File.Name,
				mime:    "image/jpeg",
				reader:  file,
				modTime: target.Blob.CreatedAt,
				etag:    `"` + target.Blob.HashMD5 + `-thumb"`,
				cache:   "private, max-age=86400",
				inline:  true,
			})
			return
		}
	}
	h.thumbSvc.GenerateAsync(target.Blob)
	h.streamTarget(c, target, true, "private, max-age=3600")
}

// Raw 内联预览（支持 Range / ETag）。
func (h *FileHandler) Raw(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	target, err := h.fileSvc.Raw(userID(c), id)
	if err != nil {
		fail(c, h.log, err)
		return
	}
	h.streamTarget(c, target, true, "private, max-age=3600")
}

// Rename 重命名。
func (h *FileHandler) Rename(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var req dto.RenameReq
	if !bindJSON(c, &req) {
		return
	}
	res, err := h.fileSvc.Rename(userID(c), id, req.Name)
	if err != nil {
		fail(c, h.log, err)
		return
	}
	resp.OK(c, res)
}

// Move 移动。
func (h *FileHandler) Move(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var req dto.MoveReq
	if !bindJSON(c, &req) {
		return
	}
	if err := h.fileSvc.Move(userID(c), id, req.TargetParentID); err != nil {
		fail(c, h.log, err)
		return
	}
	resp.OK(c, gin.H{"moved": true})
}

// SetFavorite 收藏 / 取消收藏。
func (h *FileHandler) SetFavorite(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var req dto.FavoriteReq
	if !bindJSON(c, &req) {
		return
	}
	if err := h.fileSvc.SetFavorite(userID(c), id, req.IsFavorite); err != nil {
		fail(c, h.log, err)
		return
	}
	resp.OK(c, gin.H{"is_favorite": req.IsFavorite})
}

// Copy 复制（目录递归）。
func (h *FileHandler) Copy(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var req dto.CopyReq
	if !bindJSON(c, &req) {
		return
	}
	res, err := h.fileSvc.Copy(userID(c), id, req.TargetParentID, req.NewName)
	if err != nil {
		fail(c, h.log, err)
		return
	}
	resp.OK(c, res)
}

// Delete 删除到回收站。
func (h *FileHandler) Delete(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	if err := h.fileSvc.Delete(userID(c), id); err != nil {
		fail(c, h.log, err)
		return
	}
	resp.OK(c, gin.H{"deleted": true})
}

// RecycleList 回收站列表。
func (h *FileHandler) RecycleList(c *gin.Context) {
	res, err := h.fileSvc.RecycleList(userID(c))
	if err != nil {
		fail(c, h.log, err)
		return
	}
	resp.OK(c, res)
}

// Restore 从回收站还原。
func (h *FileHandler) Restore(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	if err := h.fileSvc.Restore(userID(c), id); err != nil {
		fail(c, h.log, err)
		return
	}
	resp.OK(c, gin.H{"restored": true})
}

// ForceDelete 彻底删除。
func (h *FileHandler) ForceDelete(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	if err := h.fileSvc.ForceDelete(userID(c), id); err != nil {
		fail(c, h.log, err)
		return
	}
	resp.OK(c, gin.H{"deleted": true})
}

// EmptyRecycle 清空回收站。
func (h *FileHandler) EmptyRecycle(c *gin.Context) {
	if err := h.fileSvc.EmptyRecycle(userID(c)); err != nil {
		fail(c, h.log, err)
		return
	}
	resp.OK(c, gin.H{"emptied": true})
}

// Download 下载：单文件 attachment，目录流式 zip。
func (h *FileHandler) Download(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	uid := userID(c)
	target, err := h.fileSvc.Download(uid, id)
	if err != nil {
		fail(c, h.log, err)
		return
	}
	if target.Blob != nil {
		h.streamTarget(c, target, false, "")
		return
	}
	entries, filename, err := h.fileSvc.CollectDirZip(uid, id)
	if err != nil {
		fail(c, h.log, err)
		return
	}
	h.streamZipEntries(c, entries, filename)
}

// BatchDownload 批量下载（合并为一个 zip 流）。
func (h *FileHandler) BatchDownload(c *gin.Context) {
	var req dto.BatchDownloadReq
	if !bindJSON(c, &req) {
		return
	}
	entries, filename, err := h.fileSvc.CollectBatchZip(userID(c), req.IDs)
	if err != nil {
		fail(c, h.log, err)
		return
	}
	h.streamZipEntries(c, entries, filename)
}

// streamTarget 输出单文件内容（inline 预览或 attachment 下载）。
func (h *FileHandler) streamTarget(c *gin.Context, target *service.DownloadTarget, inline bool, cache string) {
	if target.Blob == nil {
		fail(c, h.log, errs.ErrFileNotFound.WithMessage("文件缺少物理数据"))
		return
	}
	reader, err := h.blobSvc.OpenBlob(target.Blob)
	if err != nil {
		fail(c, h.log, errs.ErrFileNotFound.WithMessage("物理文件不可用"))
		return
	}
	serveBlob(c, blobStream{
		name:    target.File.Name,
		mime:    target.Blob.MimeType,
		reader:  reader,
		modTime: target.Blob.CreatedAt,
		etag:    `"` + target.Blob.HashMD5 + `"`,
		cache:   cache,
		inline:  inline,
	})
}

// streamZipEntries 以统一配置流式输出 zip。
func (h *FileHandler) streamZipEntries(c *gin.Context, entries []service.ZipEntry, filename string) {
	streamZip(c, entries, filename, h.blobSvc.OpenBlob,
		h.cfg.Download.ZipBufferSize, h.cfg.Download.ZipMaxEntries, h.log)
}

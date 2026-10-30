package service

import (
	"image"
	"image/jpeg"
	"os"

	// 注册图片解码器：jpg / png / gif / bmp / webp。
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/iokr/novadrive/internal/config"
	"github.com/iokr/novadrive/internal/model"
	"github.com/iokr/novadrive/internal/repository"
	"github.com/iokr/novadrive/internal/storage"
	"github.com/iokr/novadrive/pkg/fileutil"
	"go.uber.org/zap"
	_ "golang.org/x/image/bmp"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// ThumbService 缩略图生成服务。
type ThumbService struct {
	cfg      *config.Config
	blobRepo *repository.BlobRepo
	storage  storage.Storage
	log      *zap.Logger
}

// NewThumbService 创建缩略图服务。
func NewThumbService(cfg *config.Config, blobRepo *repository.BlobRepo, st storage.Storage, log *zap.Logger) *ThumbService {
	return &ThumbService{cfg: cfg, blobRepo: blobRepo, storage: st, log: log}
}

// GenerateAsync 异步生成缩略图，失败不影响主流程。
func (s *ThumbService) GenerateAsync(blob *model.Blob) {
	if blob == nil || !s.cfg.Thumbnail.Enabled {
		return
	}
	if blob.ThumbPath != "" || !fileutil.IsImageExt(blob.Ext) {
		return
	}
	go func() {
		if err := s.generate(blob); err != nil {
			s.log.Warn("生成缩略图失败",
				zap.Uint64("blob_id", blob.ID),
				zap.String("hash", blob.HashMD5),
				zap.Error(err))
		}
	}()
}

// ThumbRelPath 返回 blob 对应的缩略图相对路径（可能未生成）。
func (s *ThumbService) ThumbRelPath(blob *model.Blob) string {
	if blob == nil {
		return ""
	}
	if blob.ThumbPath != "" {
		return blob.ThumbPath
	}
	return s.storage.ThumbPath(blob.HashMD5)
}

// generate 读取原图 → 等比缩放 → 写入 JPEG。
func (s *ThumbService) generate(blob *model.Blob) error {
	src, err := s.storage.Open(blob.StoragePath)
	if err != nil {
		return err
	}
	defer func() {
		if closer, ok := src.(interface{ Close() error }); ok {
			_ = closer.Close()
		}
	}()
	img, _, err := image.Decode(src)
	if err != nil {
		return err
	}
	thumb := resizeFit(img, s.cfg.Thumbnail.MaxWidth, s.cfg.Thumbnail.MaxHeight)
	relPath := s.storage.ThumbPath(blob.HashMD5)
	writer, err := s.storage.CreateThumb(relPath)
	if err != nil {
		return err
	}
	encodeErr := jpeg.Encode(writer, thumb, &jpeg.Options{Quality: s.cfg.Thumbnail.Quality})
	closeErr := writer.Close()
	if encodeErr != nil {
		_ = s.storage.DeleteThumb(relPath)
		return encodeErr
	}
	if closeErr != nil {
		return closeErr
	}
	return s.blobRepo.UpdateThumbPath(blob.ID, relPath)
}

// resizeFit 等比缩放到不超过 maxW×maxH（不放大小图）。
func resizeFit(src image.Image, maxW, maxH int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return src
	}
	if maxW <= 0 {
		maxW = 400
	}
	if maxH <= 0 {
		maxH = 400
	}
	scale := 1.0
	if w > maxW {
		scale = float64(maxW) / float64(w)
	}
	if h > maxH {
		if hs := float64(maxH) / float64(h); hs < scale {
			scale = hs
		}
	}
	if scale >= 1.0 {
		return src
	}
	dstW := int(float64(w)*scale + 0.5)
	dstH := int(float64(h)*scale + 0.5)
	if dstW < 1 {
		dstW = 1
	}
	if dstH < 1 {
		dstH = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), src, b, draw.Over, nil)
	return dst
}

// OpenThumb 打开缩略图文件，不存在时返回 os.ErrNotExist。
func (s *ThumbService) OpenThumb(relPath string) (*os.File, error) {
	reader, err := s.storage.OpenThumb(relPath)
	if err != nil {
		return nil, err
	}
	f, ok := reader.(*os.File)
	if !ok {
		return nil, os.ErrInvalid
	}
	return f, nil
}

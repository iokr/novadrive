// Package jobs 提供后台定时任务：上传会话 GC 与物理文件 GC。
package jobs

import (
	"context"
	"time"

	"github.com/iokr/novadrive/internal/config"
	"github.com/iokr/novadrive/internal/repository"
	"github.com/iokr/novadrive/internal/storage"
	"go.uber.org/zap"
)

// gcBatchSize 单轮处理的最大记录数，避免长时间占用数据库与磁盘。
const gcBatchSize = 100

// BlobGC 清理 ref_count = 0 的物理文件与缩略图。
type BlobGC struct {
	cfg      *config.Config
	log      *zap.Logger
	blobRepo *repository.BlobRepo
	storage  storage.Storage
}

// NewBlobGC 创建物理文件 GC 任务。
func NewBlobGC(cfg *config.Config, log *zap.Logger, blobRepo *repository.BlobRepo, st storage.Storage) *BlobGC {
	return &BlobGC{cfg: cfg, log: log, blobRepo: blobRepo, storage: st}
}

// Start 启动后台 GC，ctx 取消时退出。
func (j *BlobGC) Start(ctx context.Context) {
	if !j.cfg.BlobGC.Enabled {
		j.log.Info("物理文件 GC 未启用")
		return
	}
	go func() {
		ticker := time.NewTicker(j.cfg.BlobGCInterval())
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				j.log.Info("物理文件 GC 已停止")
				return
			case <-ticker.C:
				j.RunOnce()
			}
		}
	}()
}

// RunOnce 执行一轮清理，返回本轮删除的物理文件数量。
func (j *BlobGC) RunOnce() int {
	blobs, err := j.blobRepo.ListZeroRef(gcBatchSize)
	if err != nil {
		j.log.Error("查询待回收物理文件失败", zap.Error(err))
		return 0
	}
	removed := 0
	for i := range blobs {
		blob := blobs[i]
		// 先用带 ref_count = 0 条件的删除占位，避免并发复活后被误删物理内容。
		deleted, err := j.blobRepo.DeleteIfZeroRef(blob.ID)
		if err != nil {
			j.log.Error("删除物理文件记录失败",
				zap.Uint64("blob_id", blob.ID), zap.Error(err))
			continue
		}
		if deleted == 0 {
			continue
		}
		if err := j.storage.Delete(blob.StoragePath); err != nil {
			j.log.Error("删除物理文件失败，需人工清理",
				zap.Uint64("blob_id", blob.ID),
				zap.String("path", blob.StoragePath),
				zap.Error(err))
			continue
		}
		if blob.ThumbPath != "" {
			if err := j.storage.DeleteThumb(blob.ThumbPath); err != nil {
				j.log.Warn("删除缩略图失败",
					zap.Uint64("blob_id", blob.ID),
					zap.String("path", blob.ThumbPath),
					zap.Error(err))
			}
		}
		removed++
	}
	if removed > 0 {
		j.log.Info("物理文件 GC 完成", zap.Int("removed", removed))
	}
	return removed
}

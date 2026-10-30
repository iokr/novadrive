package jobs

import (
	"context"
	"time"

	"github.com/iokr/novadrive/internal/config"
	"github.com/iokr/novadrive/internal/constants"
	"github.com/iokr/novadrive/internal/repository"
	"github.com/iokr/novadrive/internal/storage"
	"go.uber.org/zap"
)

// UploadGC 清理超时未完成的分片上传会话，并标记过期分享。
type UploadGC struct {
	cfg        *config.Config
	log        *zap.Logger
	uploadRepo *repository.UploadRepo
	shareRepo  *repository.ShareRepo
	storage    storage.Storage
}

// NewUploadGC 创建会话 GC 任务。
func NewUploadGC(
	cfg *config.Config,
	log *zap.Logger,
	uploadRepo *repository.UploadRepo,
	shareRepo *repository.ShareRepo,
	st storage.Storage,
) *UploadGC {
	return &UploadGC{cfg: cfg, log: log, uploadRepo: uploadRepo, shareRepo: shareRepo, storage: st}
}

// Start 启动后台 GC，ctx 取消时退出。
func (j *UploadGC) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(j.cfg.SessionGCInterval())
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				j.log.Info("上传会话 GC 已停止")
				return
			case <-ticker.C:
				j.RunOnce()
			}
		}
	}()
}

// RunOnce 执行一轮清理：删除过期会话的分片目录并置为已过期，同时标记过期分享。
func (j *UploadGC) RunOnce() {
	sessions, err := j.uploadRepo.ListExpired(gcBatchSize)
	if err != nil {
		j.log.Error("查询过期上传会话失败", zap.Error(err))
	} else {
		for i := range sessions {
			session := sessions[i]
			if err := j.storage.DeleteChunkDir(session.UploadID); err != nil {
				j.log.Warn("删除分片目录失败",
					zap.String("upload_id", session.UploadID), zap.Error(err))
				continue
			}
			if err := j.uploadRepo.UpdateStatus(session.ID, constants.SessionExpired); err != nil {
				j.log.Warn("标记上传会话过期失败",
					zap.Uint64("session_id", session.ID), zap.Error(err))
			}
		}
		if len(sessions) > 0 {
			j.log.Info("上传会话 GC 完成", zap.Int("expired", len(sessions)))
		}
	}
	if err := j.shareRepo.MarkExpired(time.Now()); err != nil {
		j.log.Warn("标记过期分享失败", zap.Error(err))
	}
}

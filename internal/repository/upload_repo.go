package repository

import (
	"time"

	"github.com/iokr/novadrive/internal/constants"
	"github.com/iokr/novadrive/internal/model"
	"gorm.io/gorm"
)

// UploadRepo 上传会话仓储。
type UploadRepo struct {
	db *gorm.DB
}

// NewUploadRepo 创建上传会话仓储。
func NewUploadRepo(db *gorm.DB) *UploadRepo { return &UploadRepo{db: db} }

// WithTx 基于事务创建仓储副本。
func (r *UploadRepo) WithTx(tx *gorm.DB) *UploadRepo { return &UploadRepo{db: tx} }

// DB 返回底层连接。
func (r *UploadRepo) DB() *gorm.DB { return r.db }

// Create 新建会话。
func (r *UploadRepo) Create(s *model.UploadSession) error {
	return r.db.Create(s).Error
}

// FindByUploadID 按 upload_id 查询。
func (r *UploadRepo) FindByUploadID(uploadID string) (*model.UploadSession, error) {
	var s model.UploadSession
	if err := r.db.Where("upload_id = ?", uploadID).First(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

// FindResumable 查找可复用的上传中会话（断点续传）。
func (r *UploadRepo) FindResumable(userID, parentID uint64, fileName string, fileSize int64, hashMD5 string) (*model.UploadSession, error) {
	q := r.db.Where(
		"user_id = ? AND parent_id = ? AND file_name = ? AND file_size = ? AND status = ? AND expire_at > ?",
		userID, parentID, fileName, fileSize, constants.SessionUploading, time.Now(),
	)
	if hashMD5 != "" {
		q = q.Where("hash_md5 = ?", hashMD5)
	}
	var s model.UploadSession
	if err := q.Order("id ASC").First(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

// MarkMerged 乐观锁标记已合并（仅 status=0 时生效）。
func (r *UploadRepo) MarkMerged(id uint64) (int64, error) {
	res := r.db.Model(&model.UploadSession{}).
		Where("id = ? AND status = ?", id, constants.SessionUploading).
		UpdateColumn("status", constants.SessionMerged)
	return res.RowsAffected, res.Error
}

// UpdateStatus 更新会话状态。
func (r *UploadRepo) UpdateStatus(id uint64, status int8) error {
	return r.db.Model(&model.UploadSession{}).Where("id = ?", id).
		UpdateColumn("status", status).Error
}

// ListExpired 查询超时未完成的会话。
func (r *UploadRepo) ListExpired(limit int) ([]model.UploadSession, error) {
	if limit <= 0 {
		limit = 100
	}
	var sessions []model.UploadSession
	err := r.db.Where("status = ? AND expire_at < ?", constants.SessionUploading, time.Now()).
		Order("id ASC").Limit(limit).Find(&sessions).Error
	return sessions, err
}

// ListByStatus 按状态查询会话（管理端监控）。
func (r *UploadRepo) ListByStatus(status *int8, limit int) ([]model.UploadSession, error) {
	if limit <= 0 {
		limit = 200
	}
	q := r.db.Model(&model.UploadSession{})
	if status != nil {
		q = q.Where("status = ?", *status)
	}
	var sessions []model.UploadSession
	err := q.Order("id DESC").Limit(limit).Find(&sessions).Error
	return sessions, err
}

// CountByStatus 统计指定状态会话数。
func (r *UploadRepo) CountByStatus(status int8) (int64, error) {
	var count int64
	err := r.db.Model(&model.UploadSession{}).Where("status = ?", status).Count(&count).Error
	return count, err
}

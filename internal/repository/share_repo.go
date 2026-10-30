package repository

import (
	"github.com/iokr/novadrive/internal/model"
	"gorm.io/gorm"
)

// ShareRepo 分享仓储。
type ShareRepo struct {
	db *gorm.DB
}

// NewShareRepo 创建分享仓储。
func NewShareRepo(db *gorm.DB) *ShareRepo { return &ShareRepo{db: db} }

// WithTx 基于事务创建仓储副本。
func (r *ShareRepo) WithTx(tx *gorm.DB) *ShareRepo { return &ShareRepo{db: tx} }

// DB 返回底层连接。
func (r *ShareRepo) DB() *gorm.DB { return r.db }

// Create 新建分享。
func (r *ShareRepo) Create(s *model.Share) error {
	return r.db.Create(s).Error
}

// FindByID 按主键查询。
func (r *ShareRepo) FindByID(id uint64) (*model.Share, error) {
	var s model.Share
	if err := r.db.First(&s, id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

// FindByToken 按访问码查询。
func (r *ShareRepo) FindByToken(token string) (*model.Share, error) {
	var s model.Share
	if err := r.db.Where("token = ?", token).First(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

// ListByUser 查询用户创建的分享（按创建时间倒序）。
func (r *ShareRepo) ListByUser(userID uint64) ([]model.Share, error) {
	var shares []model.Share
	err := r.db.Where("user_id = ?", userID).Order("id DESC").Find(&shares).Error
	return shares, err
}

// UpdateFields 更新指定字段。
func (r *ShareRepo) UpdateFields(id uint64, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	return r.db.Model(&model.Share{}).Where("id = ?", id).Updates(fields).Error
}

// IncView 访问次数 +1。
func (r *ShareRepo) IncView(id uint64) error {
	return r.db.Model(&model.Share{}).Where("id = ?", id).
		UpdateColumn("view_count", gorm.Expr("view_count + 1")).Error
}

// IncDownload 下载次数 +1，原子校验下载上限；返回是否成功计数。
func (r *ShareRepo) IncDownload(id uint64) (bool, error) {
	res := r.db.Model(&model.Share{}).
		Where("id = ? AND status = 1 AND (max_downloads = 0 OR download_count < max_downloads)", id).
		UpdateColumn("download_count", gorm.Expr("download_count + 1"))
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// Count 分享总数。
func (r *ShareRepo) Count() (int64, error) {
	var count int64
	err := r.db.Model(&model.Share{}).Count(&count).Error
	return count, err
}

// MarkExpired 标记已过期的分享。
func (r *ShareRepo) MarkExpired(now any) error {
	return r.db.Model(&model.Share{}).
		Where("status = 1 AND expire_at IS NOT NULL AND expire_at < ?", now).
		UpdateColumn("status", 2).Error
}

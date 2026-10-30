package repository

import (
	"github.com/iokr/novadrive/internal/model"
	"gorm.io/gorm"
)

// BlobRepo 物理文件仓储。
type BlobRepo struct {
	db *gorm.DB
}

// NewBlobRepo 创建物理文件仓储。
func NewBlobRepo(db *gorm.DB) *BlobRepo { return &BlobRepo{db: db} }

// WithTx 基于事务创建仓储副本。
func (r *BlobRepo) WithTx(tx *gorm.DB) *BlobRepo { return &BlobRepo{db: tx} }

// DB 返回底层连接。
func (r *BlobRepo) DB() *gorm.DB { return r.db }

// Create 新增物理文件记录。
func (r *BlobRepo) Create(b *model.Blob) error {
	return r.db.Create(b).Error
}

// FindByHash 按 MD5 查询。
func (r *BlobRepo) FindByHash(hash string) (*model.Blob, error) {
	var b model.Blob
	if err := r.db.Where("hash_md5 = ?", hash).First(&b).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

// FindByID 按主键查询。
func (r *BlobRepo) FindByID(id uint64) (*model.Blob, error) {
	var b model.Blob
	if err := r.db.First(&b, id).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

// IncRef 引用计数 +1。
func (r *BlobRepo) IncRef(id uint64) error {
	return r.db.Model(&model.Blob{}).Where("id = ?", id).
		UpdateColumn("ref_count", gorm.Expr("ref_count + 1")).Error
}

// IncRefN 引用计数 +n。
func (r *BlobRepo) IncRefN(id uint64, n int64) error {
	if n <= 0 {
		return nil
	}
	return r.db.Model(&model.Blob{}).Where("id = ?", id).
		UpdateColumn("ref_count", gorm.Expr("ref_count + ?", n)).Error
}

// DecRef 引用计数 -n（下限为 0）。
func (r *BlobRepo) DecRef(id uint64, n int64) error {
	if n <= 0 {
		return nil
	}
	return r.db.Model(&model.Blob{}).Where("id = ?", id).
		UpdateColumn("ref_count", gorm.Expr("CASE WHEN ref_count >= ? THEN ref_count - ? ELSE 0 END", n, n)).Error
}

// UpdateThumbPath 更新缩略图路径。
func (r *BlobRepo) UpdateThumbPath(id uint64, thumbPath string) error {
	return r.db.Model(&model.Blob{}).Where("id = ?", id).
		UpdateColumn("thumb_path", thumbPath).Error
}

// ListZeroRef 查询引用计数为 0 的物理文件。
func (r *BlobRepo) ListZeroRef(limit int) ([]model.Blob, error) {
	if limit <= 0 {
		limit = 100
	}
	var blobs []model.Blob
	err := r.db.Where("ref_count = 0").Order("id ASC").Limit(limit).Find(&blobs).Error
	return blobs, err
}

// DeleteIfZeroRef 在引用计数仍为 0 时删除记录（防止并发复活）。
func (r *BlobRepo) DeleteIfZeroRef(id uint64) (int64, error) {
	res := r.db.Where("id = ? AND ref_count = 0", id).Delete(&model.Blob{})
	return res.RowsAffected, res.Error
}

// SumSize 全站物理占用。
func (r *BlobRepo) SumSize() (int64, error) {
	var total int64
	row := r.db.Model(&model.Blob{}).Select("COALESCE(SUM(size),0)").Row()
	if err := row.Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

// Count 物理文件数量。
func (r *BlobRepo) Count() (int64, error) {
	var count int64
	err := r.db.Model(&model.Blob{}).Count(&count).Error
	return count, err
}

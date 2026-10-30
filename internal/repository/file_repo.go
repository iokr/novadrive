package repository

import (
	"strings"
	"time"

	"github.com/iokr/novadrive/internal/constants"
	"github.com/iokr/novadrive/internal/model"
	"gorm.io/gorm"
)

// ExtStat 按扩展名聚合的统计结果。
type ExtStat struct {
	Ext   string
	Count int64
	Size  int64
}

// BlobRefAgg Blob 引用聚合结果。
type BlobRefAgg struct {
	BlobID uint64
	Count  int64
	Size   int64
}

// FileRepo 文件 / 目录仓储。
type FileRepo struct {
	db *gorm.DB
}

// NewFileRepo 创建文件仓储。
func NewFileRepo(db *gorm.DB) *FileRepo { return &FileRepo{db: db} }

// WithTx 基于事务创建仓储副本。
func (r *FileRepo) WithTx(tx *gorm.DB) *FileRepo { return &FileRepo{db: tx} }

// DB 返回底层连接。
func (r *FileRepo) DB() *gorm.DB { return r.db }

// sortColumns 排序字段白名单，防止 SQL 注入。
var sortColumns = map[string]string{
	"name":           "name",
	"size":           "size",
	"updated_at":     "updated_at",
	"created_at":     "created_at",
	"last_access_at": "last_access_at",
}

// normalizeSort 归一化排序参数。
func normalizeSort(sort, order string) (string, string) {
	col, ok := sortColumns[sort]
	if !ok {
		col = "name"
	}
	if strings.EqualFold(order, "desc") {
		return col, "DESC"
	}
	return col, "ASC"
}

// Create 新建文件 / 目录记录。
func (r *FileRepo) Create(f *model.File) error {
	return r.db.Create(f).Error
}

// FindByID 按主键查询（不区分状态）。
func (r *FileRepo) FindByID(id uint64) (*model.File, error) {
	var f model.File
	if err := r.db.First(&f, id).Error; err != nil {
		return nil, err
	}
	return &f, nil
}

// FindByIDAndUser 按主键 + 归属查询。
func (r *FileRepo) FindByIDAndUser(id, userID uint64) (*model.File, error) {
	var f model.File
	if err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&f).Error; err != nil {
		return nil, err
	}
	return &f, nil
}

// FindByName 按父目录 + 名称查询正常状态记录。
func (r *FileRepo) FindByName(userID, parentID uint64, name string) (*model.File, error) {
	var f model.File
	err := r.db.Where("user_id = ? AND parent_id = ? AND name = ? AND status = ?", userID, parentID, name, constants.FileStatusNormal).
		First(&f).Error
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// ExistsName 判断同父目录下是否存在同名条目。
func (r *FileRepo) ExistsName(userID, parentID uint64, name string, excludeID uint64) (bool, error) {
	q := r.db.Model(&model.File{}).
		Where("user_id = ? AND parent_id = ? AND name = ? AND status = ?", userID, parentID, name, constants.FileStatusNormal)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// ListByParent 查询指定目录下的条目（目录优先）。
func (r *FileRepo) ListByParent(userID, parentID uint64, sort, order string) ([]model.File, error) {
	col, dir := normalizeSort(sort, order)
	var files []model.File
	err := r.db.Where("user_id = ? AND parent_id = ? AND status = ?", userID, parentID, constants.FileStatusNormal).
		Order("is_dir DESC").
		Order(clause(col, dir)).
		Find(&files).Error
	return files, err
}

// clause 拼装排序片段（列名来自白名单）。
func clause(col, dir string) string {
	return col + " " + dir
}

// Search 全局按关键字检索文件名（仅正常状态）。
func (r *FileRepo) Search(userID uint64, keyword, sort, order string, limit int) ([]model.File, error) {
	col, dir := normalizeSort(sort, order)
	if limit <= 0 {
		limit = 500
	}
	like := "%" + strings.TrimSpace(keyword) + "%"
	var files []model.File
	err := r.db.Where("user_id = ? AND status = ? AND name LIKE ?", userID, constants.FileStatusNormal, like).
		Order("is_dir DESC").
		Order(clause(col, dir)).
		Limit(limit).
		Find(&files).Error
	return files, err
}

// ListAllByParent 查询某目录下全部子项（含回收站，用于递归遍历）。
func (r *FileRepo) ListAllByParent(userID, parentID uint64) ([]model.File, error) {
	var files []model.File
	err := r.db.Where("user_id = ? AND parent_id = ?", userID, parentID).Find(&files).Error
	return files, err
}

// FindLatestBySize 按目录 + 大小查找最新的正常文件（合并幂等兑底）。
func (r *FileRepo) FindLatestBySize(userID, parentID uint64, size int64) (*model.File, error) {
	var f model.File
	err := r.db.Where("user_id = ? AND parent_id = ? AND size = ? AND is_dir = 0 AND status = ?",
		userID, parentID, size, constants.FileStatusNormal).
		Order("id DESC").First(&f).Error
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// ListByParents 批量查询多个父目录下的子项（任意状态）。
func (r *FileRepo) ListByParents(userID uint64, parentIDs []uint64) ([]model.File, error) {
	if len(parentIDs) == 0 {
		return nil, nil
	}
	var files []model.File
	err := r.db.Where("user_id = ? AND parent_id IN ?", userID, parentIDs).Find(&files).Error
	return files, err
}

// ListByStatus 按状态查询（用于回收站列表）。
func (r *FileRepo) ListByStatus(userID uint64, status int8, sort, order string) ([]model.File, error) {
	col, dir := normalizeSort(sort, order)
	var files []model.File
	err := r.db.Where("user_id = ? AND status = ?", userID, status).
		Order(clause(col, dir)).
		Find(&files).Error
	return files, err
}

// Recent 最近访问列表。
func (r *FileRepo) Recent(userID uint64, limit int) ([]model.File, error) {
	if limit <= 0 {
		limit = constants.RecentDefaultLimit
	}
	var files []model.File
	err := r.db.Where("user_id = ? AND status = ? AND last_access_at IS NOT NULL", userID, constants.FileStatusNormal).
		Order("last_access_at DESC").
		Limit(limit).
		Find(&files).Error
	return files, err
}

// Favorites 收藏列表。
func (r *FileRepo) Favorites(userID uint64) ([]model.File, error) {
	var files []model.File
	err := r.db.Where("user_id = ? AND status = ? AND is_favorite = 1", userID, constants.FileStatusNormal).
		Order("updated_at DESC").
		Find(&files).Error
	return files, err
}

// StatsByExt 按扩展名聚合文件数量与大小。
func (r *FileRepo) StatsByExt(userID uint64) ([]ExtStat, error) {
	var rows []ExtStat
	err := r.db.Model(&model.File{}).
		Select("ext, COUNT(*) AS count, COALESCE(SUM(size),0) AS size").
		Where("user_id = ? AND status = ? AND is_dir = 0", userID, constants.FileStatusNormal).
		Group("ext").
		Scan(&rows).Error
	return rows, err
}

// Counts 统计文件数 / 目录数。
func (r *FileRepo) Counts(userID uint64) (files int64, dirs int64, err error) {
	var rows []struct {
		IsDir int8
		Count int64
	}
	err = r.db.Model(&model.File{}).
		Select("is_dir, COUNT(*) AS count").
		Where("user_id = ? AND status = ?", userID, constants.FileStatusNormal).
		Group("is_dir").
		Scan(&rows).Error
	if err != nil {
		return 0, 0, err
	}
	for _, row := range rows {
		if row.IsDir == constants.IsDir {
			dirs = row.Count
		} else {
			files = row.Count
		}
	}
	return files, dirs, nil
}

// UpdateFields 更新归属用户的条目字段。
func (r *FileRepo) UpdateFields(id, userID uint64, fields map[string]any) (int64, error) {
	if len(fields) == 0 {
		return 0, nil
	}
	res := r.db.Model(&model.File{}).Where("id = ? AND user_id = ?", id, userID).Updates(fields)
	return res.RowsAffected, res.Error
}

// UpdateFieldsByID 按主键更新（内部使用，调用方已完成归属校验）。
func (r *FileRepo) UpdateFieldsByID(id uint64, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	return r.db.Model(&model.File{}).Where("id = ?", id).Updates(fields).Error
}

// TouchAccess 刷新最后访问时间。
func (r *FileRepo) TouchAccess(id, userID uint64, at time.Time) error {
	return r.db.Model(&model.File{}).
		Where("id = ? AND user_id = ?", id, userID).
		UpdateColumn("last_access_at", at).Error
}

// SetStatusRecursive 递归更新子树的 status / deleted_at。
func (r *FileRepo) SetStatusRecursive(ids []uint64, status int8, deletedAt *time.Time) error {
	if len(ids) == 0 {
		return nil
	}
	fields := map[string]any{"status": status, "deleted_at": deletedAt}
	if err := r.db.Model(&model.File{}).Where("id IN ?", ids).Updates(fields).Error; err != nil {
		return err
	}
	if status == constants.FileStatusNormal {
		// 还原时清空回收站时间。
		return r.db.Model(&model.File{}).Where("id IN ?", ids).UpdateColumn("deleted_at", nil).Error
	}
	return nil
}

// DeleteByIDs 物理删除条目（不可恢复）。
func (r *FileRepo) DeleteByIDs(ids []uint64) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.Where("id IN ?", ids).Delete(&model.File{}).Error
}

// BlobRefs 聚合指定条目的 blob 引用（仅文件）。
func (r *FileRepo) BlobRefs(ids []uint64) ([]BlobRefAgg, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var rows []BlobRefAgg
	err := r.db.Model(&model.File{}).
		Select("blob_id, COUNT(*) AS count, COALESCE(SUM(size),0) AS size").
		Where("id IN ? AND is_dir = 0 AND blob_id > 0", ids).
		Group("blob_id").
		Scan(&rows).Error
	return rows, err
}

// SumSizeByIDs 汇总指定条目的大小。
func (r *FileRepo) SumSizeByIDs(ids []uint64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	var total int64
	row := r.db.Model(&model.File{}).
		Select("COALESCE(SUM(size),0)").
		Where("id IN ?", ids).
		Row()
	if err := row.Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

// GlobalCounts 统计全站文件数 / 目录数 / 逻辑占用。
func (r *FileRepo) GlobalCounts() (files int64, dirs int64, logicalSize int64, err error) {
	var rows []struct {
		IsDir int8
		Count int64
		Size  int64
	}
	err = r.db.Model(&model.File{}).
		Select("is_dir, COUNT(*) AS count, COALESCE(SUM(size),0) AS size").
		Group("is_dir").
		Scan(&rows).Error
	if err != nil {
		return 0, 0, 0, err
	}
	for _, row := range rows {
		if row.IsDir == constants.IsDir {
			dirs = row.Count
		} else {
			files = row.Count
		}
		logicalSize += row.Size
	}
	return files, dirs, logicalSize, nil
}

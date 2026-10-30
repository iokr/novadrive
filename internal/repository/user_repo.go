package repository

import (
	"strings"
	"time"

	"github.com/iokr/novadrive/internal/constants"
	"github.com/iokr/novadrive/internal/model"
	"gorm.io/gorm"
)

// UserRepo 用户仓储。
type UserRepo struct {
	db *gorm.DB
}

// NewUserRepo 创建用户仓储。
func NewUserRepo(db *gorm.DB) *UserRepo { return &UserRepo{db: db} }

// WithTx 基于事务创建仓储副本。
func (r *UserRepo) WithTx(tx *gorm.DB) *UserRepo { return &UserRepo{db: tx} }

// DB 返回底层连接。
func (r *UserRepo) DB() *gorm.DB { return r.db }

// Create 新建用户。
func (r *UserRepo) Create(u *model.User) error {
	return r.db.Create(u).Error
}

// FindByID 按主键查询。
func (r *UserRepo) FindByID(id uint64) (*model.User, error) {
	var u model.User
	if err := r.db.First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// FindByUsernameOrEmail 按用户名或邮箱查询。
func (r *UserRepo) FindByUsernameOrEmail(account string) (*model.User, error) {
	var u model.User
	err := r.db.Where("username = ? OR email = ?", account, account).First(&u).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// ExistsUsername 用户名是否已存在。
func (r *UserRepo) ExistsUsername(username string) (bool, error) {
	var count int64
	err := r.db.Model(&model.User{}).Where("username = ?", username).Count(&count).Error
	return count > 0, err
}

// ExistsEmail 邮箱是否已存在。
func (r *UserRepo) ExistsEmail(email string) (bool, error) {
	var count int64
	err := r.db.Model(&model.User{}).Where("email = ?", email).Count(&count).Error
	return count > 0, err
}

// UpdateFields 更新指定字段。
func (r *UserRepo) UpdateFields(id uint64, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	return r.db.Model(&model.User{}).Where("id = ?", id).Updates(fields).Error
}

// TouchLogin 记录最后登录时间。
func (r *UserRepo) TouchLogin(id uint64, at time.Time) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).UpdateColumn("last_login_at", at).Error
}

// AddStorageUsed 增加已用存储。
func (r *UserRepo) AddStorageUsed(id uint64, delta int64) error {
	if delta == 0 {
		return nil
	}
	return r.db.Model(&model.User{}).
		Where("id = ?", id).
		UpdateColumn("storage_used", gorm.Expr("storage_used + ?", delta)).Error
}

// SubStorageUsed 减少已用存储（下限为 0）。
func (r *UserRepo) SubStorageUsed(id uint64, delta int64) error {
	if delta <= 0 {
		return nil
	}
	return r.db.Model(&model.User{}).
		Where("id = ?", id).
		UpdateColumn("storage_used", gorm.Expr("CASE WHEN storage_used >= ? THEN storage_used - ? ELSE 0 END", delta, delta)).Error
}

// List 分页查询用户，keyword 匹配用户名 / 邮箱 / 昵称。
func (r *UserRepo) List(keyword string, offset, limit int) ([]model.User, int64, error) {
	q := r.db.Model(&model.User{})
	if kw := strings.TrimSpace(keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("username LIKE ? OR email LIKE ? OR nickname LIKE ?", like, like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var users []model.User
	err := q.Order("id ASC").Offset(offset).Limit(limit).Find(&users).Error
	return users, total, err
}

// Count 用户总数。
func (r *UserRepo) Count() (int64, error) {
	var count int64
	err := r.db.Model(&model.User{}).Count(&count).Error
	return count, err
}

// CountActive 正常状态用户数。
func (r *UserRepo) CountActive() (int64, error) {
	var count int64
	err := r.db.Model(&model.User{}).Where("status = ?", constants.UserStatusNormal).Count(&count).Error
	return count, err
}

// ExistsAdmin 是否存在管理员账号。
func (r *UserRepo) ExistsAdmin() (bool, error) {
	var count int64
	err := r.db.Model(&model.User{}).Where("role = ?", constants.RoleAdmin).Count(&count).Error
	return count > 0, err
}

// Package model 定义 GORM 数据模型，字段标签中的注释与 DDL 保持一致。
package model

import "time"

// User 用户表。
type User struct {
	ID           uint64     `gorm:"column:id;primaryKey;autoIncrement;comment:用户主键ID" json:"id"`
	Username     string     `gorm:"column:username;type:varchar(32);not null;uniqueIndex:uk_username;comment:用户名，登录账号，全局唯一" json:"username"`
	Email        string     `gorm:"column:email;type:varchar(128);not null;uniqueIndex:uk_email;comment:邮箱，全局唯一，可用于登录" json:"email"`
	PasswordHash string     `gorm:"column:password_hash;type:varchar(255);not null;comment:密码哈希（bcrypt）" json:"-"`
	Nickname     string     `gorm:"column:nickname;type:varchar(64);not null;default:'';comment:昵称，展示名称" json:"nickname"`
	Avatar       string     `gorm:"column:avatar;type:varchar(255);not null;default:'';comment:头像访问路径，空表示默认头像" json:"avatar"`
	Role         int8       `gorm:"column:role;type:tinyint;not null;default:0;comment:角色：0-普通用户，1-管理员" json:"role"`
	Status       int8       `gorm:"column:status;type:tinyint;not null;default:1;comment:账号状态：0-禁用，1-正常" json:"status"`
	StorageUsed  int64      `gorm:"column:storage_used;type:bigint unsigned;not null;default:0;comment:已用逻辑存储（字节），按 files.size 累加" json:"storage_used"`
	StorageQuota int64      `gorm:"column:storage_quota;type:bigint unsigned;not null;default:10737418240;comment:存储配额上限（字节），默认10GB" json:"storage_quota"`
	LastLoginAt  *time.Time `gorm:"column:last_login_at;comment:最后登录时间" json:"last_login_at"`
	CreatedAt    time.Time  `gorm:"column:created_at;comment:创建时间" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"column:updated_at;comment:更新时间" json:"updated_at"`
}

// TableName 指定表名。
func (User) TableName() string { return "users" }

// IsAdmin 是否为管理员。
func (u *User) IsAdmin() bool { return u.Role == 1 }

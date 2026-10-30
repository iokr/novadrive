package model

import "time"

// Share 分享表。
type Share struct {
	ID            uint64     `gorm:"column:id;primaryKey;autoIncrement;comment:分享主键ID" json:"id"`
	UserID        uint64     `gorm:"column:user_id;not null;index:idx_user_id;comment:创建分享的用户ID，关联 users.id" json:"user_id"`
	FileID        uint64     `gorm:"column:file_id;not null;index:idx_file_id;comment:被分享的文件或目录ID，关联 files.id" json:"file_id"`
	Token         string     `gorm:"column:token;type:varchar(32);not null;uniqueIndex:uk_token;comment:分享访问码，URL中唯一标识" json:"token"`
	Type          int8       `gorm:"column:type;type:tinyint;not null;default:0;comment:分享类型：0-文件分享，1-目录分享" json:"type"`
	Password      string     `gorm:"column:password;type:varchar(255);not null;default:'';comment:提取密码（bcrypt哈希），空表示无密码" json:"-"`
	ExpireAt      *time.Time `gorm:"column:expire_at;comment:过期时间，NULL表示永久有效" json:"expire_at"`
	MaxDownloads  int        `gorm:"column:max_downloads;not null;default:0;comment:最大下载次数，0表示不限制" json:"max_downloads"`
	DownloadCount int        `gorm:"column:download_count;not null;default:0;comment:已下载次数" json:"download_count"`
	ViewCount     int        `gorm:"column:view_count;not null;default:0;comment:访问次数" json:"view_count"`
	Status        int8       `gorm:"column:status;type:tinyint;not null;default:1;comment:状态：0-已取消，1-正常，2-已过期" json:"status"`
	CreatedAt     time.Time  `gorm:"column:created_at;comment:创建时间" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at;comment:更新时间" json:"updated_at"`
}

// TableName 指定表名。
func (Share) TableName() string { return "shares" }

// HasPassword 是否设置了提取码。
func (s *Share) HasPassword() bool { return s.Password != "" }

// IsExpired 分享是否已过期。
func (s *Share) IsExpired(now time.Time) bool {
	return s.ExpireAt != nil && now.After(*s.ExpireAt)
}

// ReachDownloadLimit 是否已达下载次数上限。
func (s *Share) ReachDownloadLimit() bool {
	return s.MaxDownloads > 0 && s.DownloadCount >= s.MaxDownloads
}

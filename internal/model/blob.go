package model

import "time"

// Blob 物理文件表（按 MD5 全局唯一，用于秒传与去重）。
type Blob struct {
	ID          uint64    `gorm:"column:id;primaryKey;autoIncrement;comment:物理文件主键ID" json:"id"`
	HashMD5     string    `gorm:"column:hash_md5;type:char(32);not null;uniqueIndex:uk_hash_md5;comment:文件MD5哈希，全局唯一，用于秒传与去重" json:"hash_md5"`
	Size        int64     `gorm:"column:size;type:bigint unsigned;not null;comment:物理文件大小（字节）" json:"size"`
	Ext         string    `gorm:"column:ext;type:varchar(32);not null;default:'';comment:主扩展名（小写，不含点）" json:"ext"`
	MimeType    string    `gorm:"column:mime_type;type:varchar(128);not null;default:'';comment:MIME类型" json:"mime_type"`
	StoragePath string    `gorm:"column:storage_path;type:varchar(512);not null;comment:物理存储相对路径，按 hash 前两位分片" json:"storage_path"`
	ThumbPath   string    `gorm:"column:thumb_path;type:varchar(512);not null;default:'';comment:缩略图相对路径，空表示无缩略图" json:"thumb_path"`
	RefCount    int64     `gorm:"column:ref_count;type:int unsigned;not null;default:1;index:idx_ref_count;comment:引用计数，为0时物理文件可被GC清理" json:"ref_count"`
	CreatedAt   time.Time `gorm:"column:created_at;comment:创建时间" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;comment:更新时间" json:"updated_at"`
}

// TableName 指定表名。
func (Blob) TableName() string { return "blobs" }

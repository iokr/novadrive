package model

import "time"

// File 文件与目录表（逻辑条目，物理内容通过 blob_id 引用 blobs）。
type File struct {
	ID           uint64     `gorm:"column:id;primaryKey;autoIncrement;comment:文件/目录主键ID" json:"id"`
	UserID       uint64     `gorm:"column:user_id;not null;index:idx_user_parent_status,priority:1;index:idx_user_name,priority:1;index:idx_user_favorite,priority:1;index:idx_user_last_access,priority:1;comment:所属用户ID，关联 users.id" json:"user_id"`
	ParentID     uint64     `gorm:"column:parent_id;not null;default:0;index:idx_user_parent_status,priority:2;comment:父目录ID，0表示根目录" json:"parent_id"`
	Name         string     `gorm:"column:name;type:varchar(255);not null;index:idx_user_name,priority:2;comment:文件或目录名称（含扩展名）" json:"name"`
	IsDir        int8       `gorm:"column:is_dir;type:tinyint;not null;default:0;comment:类型：0-文件，1-目录" json:"is_dir"`
	Ext          string     `gorm:"column:ext;type:varchar(32);not null;default:'';comment:扩展名（小写，不含点），目录为空" json:"ext"`
	Size         int64      `gorm:"column:size;type:bigint unsigned;not null;default:0;comment:文件大小（字节），目录为0" json:"size"`
	MimeType     string     `gorm:"column:mime_type;type:varchar(128);not null;default:'';comment:MIME类型，用于前端预览判断" json:"mime_type"`
	BlobID       uint64     `gorm:"column:blob_id;not null;default:0;index:idx_blob_id;comment:物理文件ID，关联 blobs.id，目录为0" json:"blob_id"`
	IsFavorite   int8       `gorm:"column:is_favorite;type:tinyint;not null;default:0;index:idx_user_favorite,priority:2;comment:是否收藏：0-否，1-是" json:"is_favorite"`
	LastAccessAt *time.Time `gorm:"column:last_access_at;index:idx_user_last_access,priority:2;comment:最后访问时间，用于最近访问排序" json:"last_access_at"`
	Status       int8       `gorm:"column:status;type:tinyint;not null;default:1;index:idx_user_parent_status,priority:3;comment:状态：0-回收站，1-正常" json:"status"`
	// RecycleAt 对应 deleted_at 列；刻意不命名为 DeletedAt，避免 GORM 软删除语义干扰回收站逻辑。
	RecycleAt *time.Time `gorm:"column:deleted_at;comment:进入回收站的时间，NULL表示未删除" json:"deleted_at"`
	CreatedAt time.Time  `gorm:"column:created_at;comment:创建时间" json:"created_at"`
	UpdatedAt time.Time  `gorm:"column:updated_at;comment:更新时间" json:"updated_at"`
}

// TableName 指定表名。
func (File) TableName() string { return "files" }

// IsDirectory 是否为目录。
func (f *File) IsDirectory() bool { return f.IsDir == 1 }

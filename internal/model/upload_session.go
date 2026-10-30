package model

import "time"

// UploadSession 分片上传会话表（分片本身不入库，以文件形式存放在 ChunkDir）。
type UploadSession struct {
	ID          uint64    `gorm:"column:id;primaryKey;autoIncrement;comment:上传会话主键ID" json:"id"`
	UploadID    string    `gorm:"column:upload_id;type:char(36);not null;uniqueIndex:uk_upload_id;comment:上传会话唯一标识（UUID），客户端持有" json:"upload_id"`
	UserID      uint64    `gorm:"column:user_id;not null;index:idx_user_status,priority:1;comment:所属用户ID，关联 users.id" json:"user_id"`
	ParentID    uint64    `gorm:"column:parent_id;not null;default:0;comment:目标父目录ID，0表示根目录" json:"parent_id"`
	FileName    string    `gorm:"column:file_name;type:varchar(255);not null;comment:上传文件名（含扩展名）" json:"file_name"`
	FileSize    int64     `gorm:"column:file_size;type:bigint unsigned;not null;comment:文件总大小（字节）" json:"file_size"`
	HashMD5     string    `gorm:"column:hash_md5;type:char(32);not null;default:'';comment:文件MD5，前端可选提供，用于秒传与合并校验" json:"hash_md5"`
	ChunkSize   int       `gorm:"column:chunk_size;type:int unsigned;not null;comment:分片大小（字节），由客户端在init时提交" json:"chunk_size"`
	TotalChunks int       `gorm:"column:total_chunks;type:int unsigned;not null;comment:分片总数" json:"total_chunks"`
	Status      int8      `gorm:"column:status;type:tinyint;not null;default:0;index:idx_user_status,priority:2;index:idx_status_expire,priority:1;comment:会话状态：0-上传中，1-已合并，2-已取消，3-已过期" json:"status"`
	ChunkDir    string    `gorm:"column:chunk_dir;type:varchar(512);not null;comment:分片临时目录相对路径" json:"chunk_dir"`
	ExpireAt    time.Time `gorm:"column:expire_at;not null;index:idx_status_expire,priority:2;comment:会话过期时间，超时由GC清理" json:"expire_at"`
	CreatedAt   time.Time `gorm:"column:created_at;comment:创建时间" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;comment:更新时间" json:"updated_at"`
}

// TableName 指定表名。
func (UploadSession) TableName() string { return "upload_sessions" }

// ExpectedChunkSize 计算指定分片的期望大小。
func (s *UploadSession) ExpectedChunkSize(index int) int64 {
	last := s.TotalChunks - 1
	if index < last {
		return int64(s.ChunkSize)
	}
	return s.FileSize - int64(s.ChunkSize)*int64(last)
}

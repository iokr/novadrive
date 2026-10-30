package dto

import "time"

// AdminUserItem 管理员视角的用户条目。
type AdminUserItem struct {
	ID           uint64     `json:"id"`
	Username     string     `json:"username"`
	Email        string     `json:"email"`
	Nickname     string     `json:"nickname"`
	Role         int8       `json:"role"`
	Status       int8       `json:"status"`
	StorageUsed  int64      `json:"storage_used"`
	StorageQuota int64      `json:"storage_quota"`
	LastLoginAt  *time.Time `json:"last_login_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

// AdminUserListResp 用户列表响应。
type AdminUserListResp struct {
	Total int64           `json:"total"`
	Page  int             `json:"page"`
	Size  int             `json:"size"`
	Items []AdminUserItem `json:"items"`
}

// SetStatusReq 启用 / 禁用请求。
type SetStatusReq struct {
	Status int8 `json:"status" binding:"oneof=0 1"`
}

// SetQuotaReq 配额调整请求。
type SetQuotaReq struct {
	StorageQuota int64 `json:"storage_quota" binding:"required,min=0"`
}

// AdminStatsResp 系统统计响应。
type AdminStatsResp struct {
	UserCount       int64 `json:"user_count"`
	ActiveUserCount int64 `json:"active_user_count"`
	FileCount       int64 `json:"file_count"`
	DirCount        int64 `json:"dir_count"`
	LogicalSize     int64 `json:"logical_size"`
	PhysicalSize    int64 `json:"physical_size"`
	BlobCount       int64 `json:"blob_count"`
	ShareCount      int64 `json:"share_count"`
	ActiveSessions  int64 `json:"active_sessions"`
}

// UploadSessionItem 上传会话监控条目。
type UploadSessionItem struct {
	ID          uint64    `json:"id"`
	UploadID    string    `json:"upload_id"`
	UserID      uint64    `json:"user_id"`
	UserName    string    `json:"username"`
	FileName    string    `json:"file_name"`
	FileSize    int64     `json:"file_size"`
	ChunkSize   int       `json:"chunk_size"`
	TotalChunks int       `json:"total_chunks"`
	Status      int8      `json:"status"`
	ExpireAt    time.Time `json:"expire_at"`
	CreatedAt   time.Time `json:"created_at"`
}

// UploadSessionListResp 上传会话列表响应。
type UploadSessionListResp struct {
	Total int64               `json:"total"`
	Items []UploadSessionItem `json:"items"`
}

package dto

import "time"

// FileItem 文件 / 目录列表项。
type FileItem struct {
	ID           uint64     `json:"id"`
	ParentID     uint64     `json:"parent_id"`
	Name         string     `json:"name"`
	IsDir        int8       `json:"is_dir"`
	Ext          string     `json:"ext"`
	Size         int64      `json:"size"`
	MimeType     string     `json:"mime_type"`
	IsFavorite   int8       `json:"is_favorite"`
	Status       int8       `json:"status"`
	HasThumb     bool       `json:"has_thumb"`
	LastAccessAt *time.Time `json:"last_access_at,omitempty"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// FileListResp 列表响应。
type FileListResp struct {
	ParentID uint64     `json:"parent_id"`
	Total    int        `json:"total"`
	Items    []FileItem `json:"items"`
}

// BreadcrumbItem 面包屑条目。
type BreadcrumbItem struct {
	ID   uint64 `json:"id"`
	Name string `json:"name"`
}

// TypeStat 单类型统计。
type TypeStat struct {
	Type  string `json:"type"`
	Count int64  `json:"count"`
	Size  int64  `json:"size"`
}

// FileStatsResp 空间统计响应。
type FileStatsResp struct {
	Used       int64      `json:"used"`
	Quota      int64      `json:"quota"`
	TotalSize  int64      `json:"total_size"`
	TotalCount int64      `json:"total_count"`
	DirCount   int64      `json:"dir_count"`
	ByType     []TypeStat `json:"by_type"`
}

// MkdirReq 新建目录请求。
type MkdirReq struct {
	ParentID uint64 `json:"parent_id"`
	Name     string `json:"name" binding:"required"`
}

// RenameReq 重命名请求。
type RenameReq struct {
	Name string `json:"name" binding:"required"`
}

// MoveReq 移动请求。
type MoveReq struct {
	TargetParentID uint64 `json:"target_parent_id"`
}

// CopyReq 复制请求。
type CopyReq struct {
	TargetParentID uint64 `json:"target_parent_id"`
	NewName        string `json:"new_name"`
}

// CopyResp 复制响应。
type CopyResp struct {
	NewID uint64 `json:"new_id"`
}

// FavoriteReq 收藏请求。
type FavoriteReq struct {
	IsFavorite int8 `json:"is_favorite"`
}

// CheckReq 秒传预检请求。
type CheckReq struct {
	HashMD5 string `json:"hash_md5" binding:"required"`
	Size    int64  `json:"size"`
}

// CheckResp 秒传预检响应。
type CheckResp struct {
	Hit    bool   `json:"hit"`
	BlobID uint64 `json:"blob_id,omitempty"`
}

// ManifestItem 文件夹上传清单项。
type ManifestItem struct {
	Index        int    `json:"index"`
	RelativePath string `json:"relative_path"`
}

// BatchSucceeded 批量上传成功项。
type BatchSucceeded struct {
	Name     string `json:"name"`
	ID       uint64 `json:"id"`
	ParentID uint64 `json:"parent_id"`
	Size     int64  `json:"size"`
}

// BatchFailed 批量上传失败项。
type BatchFailed struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
	Code   int    `json:"code"`
}

// BatchUploadResp 批量上传响应（部分成功可接受）。
type BatchUploadResp struct {
	Succeeded    []BatchSucceeded `json:"succeeded"`
	Failed       []BatchFailed    `json:"failed"`
	Total        int              `json:"total"`
	SuccessCount int              `json:"success_count"`
	FailCount    int              `json:"fail_count"`
}

// BatchDownloadReq 批量下载请求。
type BatchDownloadReq struct {
	IDs []uint64 `json:"ids" binding:"required,min=1"`
}

// UploadResp 小文件上传响应。
type UploadResp struct {
	ID       uint64 `json:"id"`
	Name     string `json:"name"`
	ParentID uint64 `json:"parent_id"`
	Size     int64  `json:"size"`
}

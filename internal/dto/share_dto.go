package dto

import "time"

// CreateShareReq 创建分享请求。
type CreateShareReq struct {
	FileID       uint64 `json:"file_id" binding:"required"`
	Password     string `json:"password"`
	ExpireDays   int    `json:"expire_days"`
	MaxDownloads int    `json:"max_downloads"`
}

// CreateShareResp 创建分享响应。
type CreateShareResp struct {
	ID          uint64     `json:"id"`
	Token       string     `json:"token"`
	URL         string     `json:"url"`
	HasPassword bool       `json:"has_password"`
	Password    string     `json:"password,omitempty"`
	ExpireAt    *time.Time `json:"expire_at,omitempty"`
}

// ShareItem 我的分享列表项。
type ShareItem struct {
	ID            uint64     `json:"id"`
	Token         string     `json:"token"`
	URL           string     `json:"url"`
	FileID        uint64     `json:"file_id"`
	FileName      string     `json:"file_name"`
	IsDir         int8       `json:"is_dir"`
	Type          int8       `json:"type"`
	HasPassword   bool       `json:"has_password"`
	ExpireAt      *time.Time `json:"expire_at"`
	MaxDownloads  int        `json:"max_downloads"`
	DownloadCount int        `json:"download_count"`
	ViewCount     int        `json:"view_count"`
	Status        int8       `json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
}

// PublicShareInfo 公开分享基本信息。
type PublicShareInfo struct {
	Token         string     `json:"token"`
	FileName      string     `json:"file_name"`
	IsDir         int8       `json:"is_dir"`
	FileSize      int64      `json:"file_size"`
	Type          int8       `json:"type"`
	HasPassword   bool       `json:"has_password"`
	ExpireAt      *time.Time `json:"expire_at"`
	ViewCount     int        `json:"view_count"`
	DownloadCount int        `json:"download_count"`
	MaxDownloads  int        `json:"max_downloads"`
	CreatedAt     time.Time  `json:"created_at"`
}

// VerifyResp 校验提取码响应。
type VerifyResp struct {
	AccessKey string `json:"access_key"`
}

// PublicFileList 分享内容浏览响应。
type PublicFileList struct {
	Share   PublicShareInfo `json:"share"`
	RootID  uint64          `json:"root_id"`
	Current uint64          `json:"current_id"`
	Total   int             `json:"total"`
	Items   []FileItem      `json:"items"`
}

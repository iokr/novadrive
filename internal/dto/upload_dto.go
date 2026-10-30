package dto

import "time"

// InitUploadReq 分片上传初始化请求。
type InitUploadReq struct {
	ParentID    uint64 `json:"parent_id"`
	FileName    string `json:"file_name" binding:"required"`
	FileSize    int64  `json:"file_size" binding:"min=0"`
	HashMD5     string `json:"hash_md5"`
	ChunkSize   int64  `json:"chunk_size" binding:"required,min=1"`
	TotalChunks int    `json:"total_chunks" binding:"required,min=1"`
}

// InitUploadResp 分片上传初始化响应。
type InitUploadResp struct {
	UploadID       string   `json:"upload_id,omitempty"`
	Instant        bool     `json:"instant"`
	File           FileItem `json:"file,omitempty"`
	UploadedChunks []int    `json:"uploaded_chunks,omitempty"`
	ChunkSize      int      `json:"chunk_size,omitempty"`
	TotalChunks    int      `json:"total_chunks,omitempty"`
}

// ChunkResp 分片上传响应。
type ChunkResp struct {
	Index    int  `json:"index"`
	Received bool `json:"received"`
}

// UploadStatusResp 分片上传进度响应。
type UploadStatusResp struct {
	UploadID       string    `json:"upload_id"`
	Status         int8      `json:"status"`
	FileName       string    `json:"file_name"`
	FileSize       int64     `json:"file_size"`
	UploadedChunks []int     `json:"uploaded_chunks"`
	TotalChunks    int       `json:"total_chunks"`
	UploadedBytes  int64     `json:"uploaded_bytes"`
	ExpireAt       time.Time `json:"expire_at"`
}

// CompleteReq 合并请求。
type CompleteReq struct {
	HashMD5 string `json:"hash_md5"`
}

// CompleteResp 合并响应。
type CompleteResp struct {
	File    FileItem `json:"file"`
	Instant bool     `json:"instant"`
}

// Package constants 汇总业务枚举与固定取值。
package constants

import "time"

const (
	// AppName 服务名 / 产品英文名。
	AppName = "NovaDrive"
	// AppNameCN 产品中文名。
	AppNameCN = "星云盘"
	// APIPrefix 接口基础路径。
	APIPrefix = "/api/v1"
)

// 用户角色。
const (
	RoleUser  int8 = 0
	RoleAdmin int8 = 1
)

// 用户状态。
const (
	UserStatusDisabled int8 = 0
	UserStatusNormal   int8 = 1
)

// 文件状态。
const (
	FileStatusRecycle int8 = 0
	FileStatusNormal  int8 = 1
)

// 文件类型。
const (
	IsFile int8 = 0
	IsDir  int8 = 1
)

// 上传会话状态。
const (
	SessionUploading int8 = 0
	SessionMerged    int8 = 1
	SessionCanceled  int8 = 2
	SessionExpired   int8 = 3
)

// 分片常量。
const (
	// ChunkSuffix 分片文件扩展名。
	ChunkSuffix = ".part"
	// MergeTempPattern 合并临时文件名模板。
	MergeTempPattern = "merge-*.tmp"
)

// 分享状态。
const (
	ShareStatusCanceled int8 = 0
	ShareStatusNormal   int8 = 1
	ShareStatusExpired  int8 = 2
)

// 分享类型。
const (
	ShareTypeFile int8 = 0
	ShareTypeDir  int8 = 1
)

// 根目录父节点。
const (
	RootParentID uint64 = 0
)

// DefaultStorageQuota 默认存储配额（10GB）。
const DefaultStorageQuota int64 = 10737418240

// DefaultBcryptCost 密码哈希开销。
const DefaultBcryptCost = 10

// MaxNameLength 文件名 / 目录名最大长度。
const MaxNameLength = 255

// MaxDirDepth 目录递归最大层级（防御异常数据）。
const MaxDirDepth = 64

// RecentDefaultLimit "最近访问"默认条数。
const RecentDefaultLimit = 20

// RecentMaxLimit "最近访问"最大条数。
const RecentMaxLimit = 100

// AccessTokenQueryKey 流媒体 / 预览接口的 token 查询参数名。
const AccessTokenQueryKey = "token"

// ShareKeyHeader 分享访问校验头。
const ShareKeyHeader = "X-Share-Key"

// StatsCategories 空间统计分类。
var StatsCategories = []string{"image", "video", "pdf", "audio", "text", "other"}

// FileAccessTouchInterval 访问时间刷新最小间隔（避免频繁写库）。
const FileAccessTouchInterval = time.Minute

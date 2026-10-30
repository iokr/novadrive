// Package config 负责加载 YAML 配置文件（唯一配置来源，不使用环境变量）。
package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config 顶层配置结构。
type Config struct {
	Server    Server    `mapstructure:"server"`
	Database  Database  `mapstructure:"database"`
	JWT       JWT       `mapstructure:"jwt"`
	Storage   Storage   `mapstructure:"storage"`
	Download  Download  `mapstructure:"download"`
	Thumbnail Thumbnail `mapstructure:"thumbnail"`
	Share     Share     `mapstructure:"share"`
	BlobGC    BlobGC    `mapstructure:"blob_gc"`
	Bootstrap Bootstrap `mapstructure:"bootstrap"`
	Log       Log       `mapstructure:"log"`
}

// Server HTTP 服务配置。
type Server struct {
	Name         string `mapstructure:"name"`
	Port         int    `mapstructure:"port"`
	Mode         string `mapstructure:"mode"`
	ReadTimeout  int    `mapstructure:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout"`
	BaseURL      string `mapstructure:"base_url"`
}

// Database MySQL 配置。
type Database struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	Name         string `mapstructure:"name"`
	Charset      string `mapstructure:"charset"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
}

// JWT 令牌配置。
type JWT struct {
	Secret      string `mapstructure:"secret"`
	ExpireHours int    `mapstructure:"expire_hours"`
}

// Storage 存储配置。
type Storage struct {
	Type    string     `mapstructure:"type"`
	Local   LocalStore `mapstructure:"local"`
	Upload  Upload     `mapstructure:"upload"`
	Chunk   Chunk      `mapstructure:"chunk"`
	Session Session    `mapstructure:"session"`
}

// LocalStore 本地存储根目录配置。
type LocalStore struct {
	Root       string `mapstructure:"root"`
	ThumbRoot  string `mapstructure:"thumb_root"`
	UploadRoot string `mapstructure:"upload_root"`
}

// Upload 上传限制配置。
type Upload struct {
	MaxFileSize   int64    `mapstructure:"max_file_size"`
	MaxBatchFiles int      `mapstructure:"max_batch_files"`
	MaxBatchSize  int64    `mapstructure:"max_batch_size"`
	AllowedExt    []string `mapstructure:"allowed_ext"`
}

// Chunk 分片上传配置。
type Chunk struct {
	Enabled          bool  `mapstructure:"enabled"`
	Threshold        int64 `mapstructure:"threshold"`
	DefaultChunkSize int64 `mapstructure:"default_chunk_size"`
	MaxChunkSize     int64 `mapstructure:"max_chunk_size"`
	MergeBufferSize  int   `mapstructure:"merge_buffer_size"`
}

// Session 上传会话配置。
type Session struct {
	ExpireHours       int `mapstructure:"expire_hours"`
	GCIntervalMinutes int `mapstructure:"gc_interval_minutes"`
}

// Download 下载 / zip 打包配置。
type Download struct {
	ZipBufferSize int `mapstructure:"zip_buffer_size"`
	ZipMaxEntries int `mapstructure:"zip_max_entries"`
}

// Thumbnail 缩略图配置。
type Thumbnail struct {
	Enabled   bool `mapstructure:"enabled"`
	MaxWidth  int  `mapstructure:"max_width"`
	MaxHeight int  `mapstructure:"max_height"`
	Quality   int  `mapstructure:"quality"`
}

// Share 分享配置。
type Share struct {
	TokenLength       int `mapstructure:"token_length"`
	DefaultExpireDays int `mapstructure:"default_expire_days"`
}

// BlobGC 物理文件 GC 配置。
type BlobGC struct {
	Enabled         bool `mapstructure:"enabled"`
	IntervalMinutes int  `mapstructure:"interval_minutes"`
}

// Bootstrap 初始管理员配置。
type Bootstrap struct {
	AdminUsername string `mapstructure:"admin_username"`
	AdminEmail    string `mapstructure:"admin_email"`
	AdminPassword string `mapstructure:"admin_password"`
}

// Log 日志配置。
type Log struct {
	Level string `mapstructure:"level"`
	Path  string `mapstructure:"path"`
}

// defaultAllowedExt 配置缺省时的上传白名单。
var defaultAllowedExt = []string{
	"pdf", "mp4", "webm", "mov", "jpg", "jpeg", "png", "gif", "webp", "bmp", "svg",
	"mp3", "wav", "txt", "md", "json", "zip", "docx", "xlsx", "pptx",
}

// Load 从给定路径加载并校验配置。
func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件 %s 失败: %w", path, err)
	}
	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}
	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) applyDefaults() {
	if c.Server.Name == "" {
		c.Server.Name = "novadrive"
	}
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Server.Mode == "" {
		c.Server.Mode = "release"
	}
	if c.Server.ReadTimeout <= 0 {
		c.Server.ReadTimeout = 60
	}
	if c.Server.WriteTimeout <= 0 {
		c.Server.WriteTimeout = 600
	}
	if c.Server.BaseURL == "" {
		c.Server.BaseURL = fmt.Sprintf("http://localhost:%d", c.Server.Port)
	}
	if c.Database.Port == 0 {
		c.Database.Port = 3306
	}
	if c.Database.Charset == "" {
		c.Database.Charset = "utf8mb4"
	}
	if c.Database.MaxIdleConns <= 0 {
		c.Database.MaxIdleConns = 10
	}
	if c.Database.MaxOpenConns <= 0 {
		c.Database.MaxOpenConns = 100
	}
	if c.JWT.ExpireHours <= 0 {
		c.JWT.ExpireHours = 72
	}
	if c.Storage.Type == "" {
		c.Storage.Type = "local"
	}
	if c.Storage.Local.Root == "" {
		c.Storage.Local.Root = "./data/blobs"
	}
	if c.Storage.Local.ThumbRoot == "" {
		c.Storage.Local.ThumbRoot = "./data/thumbs"
	}
	if c.Storage.Local.UploadRoot == "" {
		c.Storage.Local.UploadRoot = "./data/uploads"
	}
	if c.Storage.Upload.MaxFileSize <= 0 {
		c.Storage.Upload.MaxFileSize = 2 << 30
	}
	if c.Storage.Upload.MaxBatchFiles <= 0 {
		c.Storage.Upload.MaxBatchFiles = 500
	}
	if c.Storage.Upload.MaxBatchSize <= 0 {
		c.Storage.Upload.MaxBatchSize = 4 << 30
	}
	if len(c.Storage.Upload.AllowedExt) == 0 {
		c.Storage.Upload.AllowedExt = append([]string(nil), defaultAllowedExt...)
	}
	if c.Storage.Chunk.DefaultChunkSize <= 0 {
		c.Storage.Chunk.DefaultChunkSize = 5 << 20
	}
	if c.Storage.Chunk.Threshold <= 0 {
		c.Storage.Chunk.Threshold = 10 << 20
	}
	if c.Storage.Chunk.MaxChunkSize <= 0 {
		c.Storage.Chunk.MaxChunkSize = 50 << 20
	}
	if c.Storage.Chunk.MergeBufferSize <= 0 {
		c.Storage.Chunk.MergeBufferSize = 4 << 20
	}
	if c.Storage.Session.ExpireHours <= 0 {
		c.Storage.Session.ExpireHours = 24
	}
	if c.Storage.Session.GCIntervalMinutes <= 0 {
		c.Storage.Session.GCIntervalMinutes = 60
	}
	if c.Download.ZipBufferSize <= 0 {
		c.Download.ZipBufferSize = 32 << 10
	}
	if c.Download.ZipMaxEntries <= 0 {
		c.Download.ZipMaxEntries = 10000
	}
	if c.Thumbnail.MaxWidth <= 0 {
		c.Thumbnail.MaxWidth = 400
	}
	if c.Thumbnail.MaxHeight <= 0 {
		c.Thumbnail.MaxHeight = 400
	}
	if c.Thumbnail.Quality <= 0 || c.Thumbnail.Quality > 100 {
		c.Thumbnail.Quality = 80
	}
	if c.Share.TokenLength <= 0 {
		c.Share.TokenLength = 32
	}
	if c.BlobGC.IntervalMinutes <= 0 {
		c.BlobGC.IntervalMinutes = 30
	}
	if c.Bootstrap.AdminUsername == "" {
		c.Bootstrap.AdminUsername = "admin"
	}
	if c.Bootstrap.AdminEmail == "" {
		c.Bootstrap.AdminEmail = "admin@local"
	}
	if c.Log.Level == "" {
		c.Log.Level = "info"
	}
	if c.Log.Path == "" {
		c.Log.Path = "./logs"
	}
}

func (c *Config) validate() error {
	if c.Database.Host == "" || c.Database.Name == "" || c.Database.User == "" {
		return errors.New("数据库配置不完整：host / name / user 为必填项")
	}
	if c.JWT.Secret == "" {
		return errors.New("jwt.secret 不能为空")
	}
	if c.Storage.Type != "local" {
		return fmt.Errorf("暂不支持的存储类型: %s", c.Storage.Type)
	}
	return nil
}

// DSN 生成 MySQL 连接串。
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=true&loc=Local&timeout=10s",
		c.Database.User, c.Database.Password, c.Database.Host, c.Database.Port,
		c.Database.Name, c.Database.Charset,
	)
}

// Addr 监听地址。
func (c *Config) Addr() string {
	return fmt.Sprintf(":%d", c.Server.Port)
}

// SessionTTL 上传会话有效期。
func (c *Config) SessionTTL() time.Duration {
	return time.Duration(c.Storage.Session.ExpireHours) * time.Hour
}

// SessionGCInterval 上传会话 GC 周期。
func (c *Config) SessionGCInterval() time.Duration {
	return time.Duration(c.Storage.Session.GCIntervalMinutes) * time.Minute
}

// BlobGCInterval 物理文件 GC 周期。
func (c *Config) BlobGCInterval() time.Duration {
	return time.Duration(c.BlobGC.IntervalMinutes) * time.Minute
}

// TokenTTL JWT 有效期。
func (c *Config) TokenTTL() time.Duration {
	return time.Duration(c.JWT.ExpireHours) * time.Hour
}

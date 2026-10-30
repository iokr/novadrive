package config

import (
	"path/filepath"
	"testing"
)

// TestLoadExampleConfig 校验仓库自带的示例配置可解析并通过校验。
func TestLoadExampleConfig(t *testing.T) {
	path := filepath.Join("..", "..", "config", "config.yaml")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("加载示例配置失败: %v", err)
	}
	if cfg.Server.Port != 8080 {
		t.Fatalf("默认端口期望 8080，实际 %d", cfg.Server.Port)
	}
	if cfg.Storage.Type != "local" {
		t.Fatalf("存储类型期望 local，实际 %s", cfg.Storage.Type)
	}
	if len(cfg.Storage.Upload.AllowedExt) == 0 {
		t.Fatal("上传扩展名白名单不应为空")
	}
	if cfg.TokenTTL() <= 0 {
		t.Fatal("JWT 有效期应为正数")
	}
	if cfg.BlobGCInterval() <= 0 || cfg.SessionGCInterval() <= 0 {
		t.Fatal("GC 周期应为正数")
	}
}

// TestLoadMissingFile 校验缺失配置文件时返回错误。
func TestLoadMissingFile(t *testing.T) {
	if _, err := Load(filepath.Join("testdata", "not-exist.yaml")); err == nil {
		t.Fatal("配置文件缺失时应当返回错误")
	}
}

// TestApplyDefaultsAndValidate 校验缺省值回填与关键校验。
func TestApplyDefaultsAndValidate(t *testing.T) {
	cfg := &Config{}
	cfg.applyDefaults()
	if err := cfg.validate(); err == nil {
		t.Fatal("缺少 jwt.secret 与数据库账号时应当校验失败")
	}
	cfg.JWT.Secret = "test-secret"
	cfg.Database.Host = "127.0.0.1"
	cfg.Database.User = "root"
	cfg.Database.Name = "novadrive"
	if err := cfg.validate(); err != nil {
		t.Fatalf("补齐必填项后应当校验通过: %v", err)
	}
	if cfg.Storage.Chunk.DefaultChunkSize <= 0 || cfg.Thumbnail.MaxWidth <= 0 {
		t.Fatal("缺省值回填不完整")
	}
}

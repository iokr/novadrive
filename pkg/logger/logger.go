// Package logger 提供基于 zap 的日志初始化（控制台 + 文件）。
package logger

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/iokr/novadrive/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Init 按配置创建 logger：日志同时输出到标准输出与 logs 目录下的文件。
func Init(cfg *config.Config) (*zap.Logger, error) {
	level := zapcore.InfoLevel
	if err := level.UnmarshalText([]byte(cfg.Log.Level)); err != nil {
		level = zapcore.InfoLevel
	}

	encCfg := zap.NewProductionEncoderConfig()
	encCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encCfg.EncodeDuration = zapcore.MillisDurationEncoder
	consoleEncoder := zapcore.NewConsoleEncoder(encCfg)
	fileEncoder := zapcore.NewJSONEncoder(encCfg)

	consoleSyncer := zapcore.AddSync(os.Stdout)

	if err := os.MkdirAll(cfg.Log.Path, 0o755); err != nil {
		return nil, fmt.Errorf("创建日志目录失败: %w", err)
	}
	filePath := filepath.Join(cfg.Log.Path, cfg.Server.Name+".log")
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("打开日志文件失败: %w", err)
	}
	fileSyncer := zapcore.AddSync(file)

	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, consoleSyncer, level),
		zapcore.NewCore(fileEncoder, fileSyncer, level),
	)
	return zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel)), nil
}

// Nop 返回无输出 logger（测试或降级场景）。
func Nop() *zap.Logger { return zap.NewNop() }

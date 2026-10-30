// Package repository 负责数据持久化（GORM）。
package repository

import (
	"errors"
	"fmt"
	"time"

	"github.com/iokr/novadrive/internal/config"
	"github.com/iokr/novadrive/internal/model"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// gormWriter 把 GORM 日志转接到 zap。
type gormWriter struct{ log *zap.Logger }

// Printf 实现 gormlogger.Writer。
func (w gormWriter) Printf(format string, args ...any) {
	w.log.Sugar().Warnf(format, args...)
}

// NewDB 建立数据库连接并配置连接池。
func NewDB(cfg *config.Config, log *zap.Logger) (*gorm.DB, error) {
	level := gormlogger.Warn
	if cfg.Server.Mode == "debug" {
		level = gormlogger.Info
	}
	db, err := gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{
		Logger: gormlogger.New(gormWriter{log: log}, gormlogger.Config{
			SlowThreshold:             500 * time.Millisecond,
			LogLevel:                  level,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		}),
		NamingStrategy:         schema.NamingStrategy{SingularTable: true},
		SkipDefaultTransaction: true,
	})
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库连接池失败: %w", err)
	}
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Hour)
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("数据库不可用: %w", err)
	}
	return db, nil
}

// AutoMigrate 自动建表 / 同步表结构（与 migrations/0001_init_schema.sql 保持一致）。
func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&model.User{},
		&model.Blob{},
		&model.File{},
		&model.UploadSession{},
		&model.Share{},
	); err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}
	return nil
}

// Close 关闭数据库连接。
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// IsNotFound 判断是否为记录不存在错误。
func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

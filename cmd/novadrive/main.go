// Package main 是 NovaDrive（星云盘）服务入口：单二进制同时提供 API 与 SPA。
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/iokr/novadrive/internal/config"
	"github.com/iokr/novadrive/internal/handler"
	"github.com/iokr/novadrive/internal/jobs"
	"github.com/iokr/novadrive/internal/repository"
	"github.com/iokr/novadrive/internal/router"
	"github.com/iokr/novadrive/internal/service"
	"github.com/iokr/novadrive/internal/storage"
	"github.com/iokr/novadrive/pkg/jwt"
	"github.com/iokr/novadrive/pkg/logger"
	"go.uber.org/zap"
)

// shutdownTimeout 优雅关闭的等待上限。
const shutdownTimeout = 15 * time.Second

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "启动失败: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	configPath := flag.String("c", "config/config.yaml", "配置文件路径")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	log, err := logger.Init(cfg)
	if err != nil {
		return err
	}
	defer func() { _ = log.Sync() }()
	log.Info("正在启动 NovaDrive",
		zap.String("addr", cfg.Addr()),
		zap.String("config", *configPath),
		zap.String("mode", cfg.Server.Mode))

	db, err := repository.NewDB(cfg, log)
	if err != nil {
		return err
	}
	defer func() { _ = repository.Close(db) }()

	if err := repository.AutoMigrate(db); err != nil {
		return err
	}

	local, err := storage.NewLocal(cfg)
	if err != nil {
		return err
	}

	userRepo := repository.NewUserRepo(db)
	blobRepo := repository.NewBlobRepo(db)
	fileRepo := repository.NewFileRepo(db)
	uploadRepo := repository.NewUploadRepo(db)
	shareRepo := repository.NewShareRepo(db)

	jwtMgr := jwt.New(cfg.JWT.Secret, cfg.TokenTTL())

	userSvc := service.NewUserService(userRepo, fileRepo)
	thumbSvc := service.NewThumbService(cfg, blobRepo, local, log)
	blobSvc := service.NewBlobService(blobRepo, fileRepo, userRepo, local, thumbSvc, log)
	fileSvc := service.NewFileService(cfg, log, local, fileRepo, blobRepo, userRepo, userSvc, blobSvc)
	uploadSvc := service.NewUploadService(cfg, log, local, uploadRepo, fileRepo, blobSvc, userSvc)
	shareSvc := service.NewShareService(cfg, log, shareRepo, fileRepo, blobSvc, local, cfg.JWT.Secret)
	authSvc := service.NewAuthService(cfg, userRepo, jwtMgr, log)
	adminSvc := service.NewAdminService(cfg, log, userRepo, fileRepo, blobRepo, uploadRepo, shareRepo)

	if err := adminSvc.Bootstrap(); err != nil {
		return fmt.Errorf("初始化管理员账号失败: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	jobs.NewUploadGC(cfg, log, uploadRepo, shareRepo, local).Start(ctx)
	jobs.NewBlobGC(cfg, log, blobRepo, local).Start(ctx)

	engine := router.New(router.Deps{
		Cfg:      cfg,
		Log:      log,
		JWT:      jwtMgr,
		UserRepo: userRepo,
		Auth:     handler.NewAuthHandler(authSvc, log),
		User:     handler.NewUserHandler(userSvc, log),
		File:     handler.NewFileHandler(cfg, log, fileSvc, uploadSvc, thumbSvc, blobSvc),
		Share:    handler.NewShareHandler(cfg, log, shareSvc, fileSvc, blobSvc),
		Admin:    handler.NewAdminHandler(adminSvc, log),
	})

	srv := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           engine,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout:      time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		log.Info("HTTP 服务已就绪", zap.String("addr", cfg.Addr()))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()

	select {
	case err := <-serveErr:
		return fmt.Errorf("HTTP 服务异常退出: %w", err)
	case <-ctx.Done():
		log.Info("收到退出信号，开始优雅关闭")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("优雅关闭失败: %w", err)
	}
	log.Info("服务已退出")
	return nil
}

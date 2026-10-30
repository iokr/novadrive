// Package router 负责装配中间件与全部 HTTP 路由。
package router

import (
	"github.com/gin-gonic/gin"
	"github.com/iokr/novadrive/internal/config"
	"github.com/iokr/novadrive/internal/constants"
	"github.com/iokr/novadrive/internal/handler"
	"github.com/iokr/novadrive/internal/middleware"
	"github.com/iokr/novadrive/internal/repository"
	"github.com/iokr/novadrive/pkg/jwt"
	"github.com/iokr/novadrive/pkg/resp"
	"go.uber.org/zap"
)

// multipartMemory multipart 表单内存缓冲上限，超过后落临时文件。
const multipartMemory = 8 << 20

// Deps 路由依赖集合。
type Deps struct {
	Cfg      *config.Config
	Log      *zap.Logger
	JWT      *jwt.Manager
	UserRepo *repository.UserRepo
	Auth     *handler.AuthHandler
	User     *handler.UserHandler
	File     *handler.FileHandler
	Share    *handler.ShareHandler
	Admin    *handler.AdminHandler
}

// New 构建 Gin 引擎：Recover → Logger → CORS → (Auth) → (Admin) → Handler。
func New(d Deps) *gin.Engine {
	gin.SetMode(ginMode(d.Cfg.Server.Mode))
	engine := gin.New()
	engine.MaxMultipartMemory = multipartMemory
	engine.Use(
		middleware.Recover(d.Log),
		middleware.Logger(d.Log),
		middleware.CORS(),
	)

	api := engine.Group(constants.APIPrefix)
	api.GET("/health", func(c *gin.Context) {
		resp.OK(c, gin.H{"status": "ok", "name": constants.AppName})
	})

	// 公开接口：注册 / 登录。
	auth := api.Group("/auth")
	auth.POST("/register", d.Auth.Register)
	auth.POST("/login", d.Auth.Login)

	// 公开接口：分享访问。
	public := api.Group("/shares/public")
	public.GET("/:token", d.Share.PublicInfo)
	public.POST("/:token/verify", d.Share.Verify)
	public.GET("/:token/files", d.Share.PublicFiles)
	public.GET("/:token/preview/:file_id", d.Share.Preview)
	public.GET("/:token/download", d.Share.Download)

	// 以下路由均需登录。
	authorized := api.Group("", middleware.Auth(d.JWT, d.UserRepo))

	authorized.GET("/auth/profile", d.Auth.Profile)
	authorized.PUT("/auth/profile", d.Auth.UpdateProfile)
	authorized.PUT("/auth/password", d.Auth.ChangePassword)

	authorized.GET("/user/storage", d.User.Storage)
	authorized.GET("/user/overview", d.User.Overview)

	files := authorized.Group("/files")
	files.GET("", d.File.List)
	files.GET("/breadcrumb", d.File.Breadcrumb)
	files.GET("/recent", d.File.Recent)
	files.GET("/favorites", d.File.Favorites)
	files.GET("/stats", d.File.Stats)
	files.GET("/recycle", d.File.RecycleList)
	files.DELETE("/recycle", d.File.EmptyRecycle)
	files.POST("/mkdir", d.File.Mkdir)
	files.POST("/upload", d.File.Upload)
	files.POST("/upload/batch", d.File.UploadBatch)
	files.POST("/check", d.File.Check)
	files.POST("/upload/init", d.File.InitUpload)
	files.PUT("/upload/:upload_id/chunk", d.File.UploadChunk)
	files.GET("/upload/:upload_id/status", d.File.UploadStatus)
	files.POST("/upload/:upload_id/complete", d.File.CompleteUpload)
	files.DELETE("/upload/:upload_id", d.File.CancelUpload)
	files.POST("/download/batch", d.File.BatchDownload)
	files.GET("/:id", d.File.Detail)
	files.GET("/:id/thumb", d.File.Thumb)
	files.GET("/:id/raw", d.File.Raw)
	files.GET("/:id/download", d.File.Download)
	files.PUT("/:id/rename", d.File.Rename)
	files.PUT("/:id/move", d.File.Move)
	files.PUT("/:id/favorite", d.File.SetFavorite)
	files.POST("/:id/copy", d.File.Copy)
	files.POST("/:id/restore", d.File.Restore)
	files.DELETE("/:id", d.File.Delete)
	files.DELETE("/:id/force", d.File.ForceDelete)

	shares := authorized.Group("/shares")
	shares.POST("", d.Share.Create)
	shares.GET("", d.Share.List)
	shares.DELETE("/:id", d.Share.Cancel)

	admin := authorized.Group("/admin", middleware.Admin())
	admin.GET("/users", d.Admin.Users)
	admin.PUT("/users/:id/status", d.Admin.SetStatus)
	admin.PUT("/users/:id/quota", d.Admin.SetQuota)
	admin.GET("/stats", d.Admin.Stats)
	admin.GET("/upload-sessions", d.Admin.UploadSessions)

	// 未匹配路由交由 SPA 处理（静态资源 / history 回退 / 接口 404）。
	engine.NoRoute(spaHandler())
	return engine
}

// ginMode 将配置中的运行模式映射为 Gin 模式。
func ginMode(mode string) string {
	switch mode {
	case gin.DebugMode, gin.TestMode, gin.ReleaseMode:
		return mode
	default:
		return gin.ReleaseMode
	}
}

package service

import (
	"strings"

	"github.com/iokr/novadrive/internal/config"
	"github.com/iokr/novadrive/internal/constants"
	"github.com/iokr/novadrive/internal/dto"
	"github.com/iokr/novadrive/internal/model"
	"github.com/iokr/novadrive/internal/repository"
	"github.com/iokr/novadrive/pkg/errs"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// AdminService 管理员后台业务。
type AdminService struct {
	cfg        *config.Config
	log        *zap.Logger
	userRepo   *repository.UserRepo
	fileRepo   *repository.FileRepo
	blobRepo   *repository.BlobRepo
	uploadRepo *repository.UploadRepo
	shareRepo  *repository.ShareRepo
}

// NewAdminService 创建管理员服务。
func NewAdminService(
	cfg *config.Config,
	log *zap.Logger,
	userRepo *repository.UserRepo,
	fileRepo *repository.FileRepo,
	blobRepo *repository.BlobRepo,
	uploadRepo *repository.UploadRepo,
	shareRepo *repository.ShareRepo,
) *AdminService {
	return &AdminService{
		cfg:        cfg,
		log:        log,
		userRepo:   userRepo,
		fileRepo:   fileRepo,
		blobRepo:   blobRepo,
		uploadRepo: uploadRepo,
		shareRepo:  shareRepo,
	}
}

// Bootstrap 启动时确保存在管理员账号。
func (s *AdminService) Bootstrap() error {
	exists, err := s.userRepo.ExistsAdmin()
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	username := strings.TrimSpace(s.cfg.Bootstrap.AdminUsername)
	email := strings.ToLower(strings.TrimSpace(s.cfg.Bootstrap.AdminEmail))
	password := s.cfg.Bootstrap.AdminPassword
	if username == "" || password == "" {
		s.log.Warn("未配置 bootstrap 管理员账号，跳过初始化")
		return nil
	}
	if taken, err := s.userRepo.ExistsUsername(username); err != nil {
		return err
	} else if taken {
		// 已存在同名普通用户，提升为管理员。
		user, err := s.userRepo.FindByUsernameOrEmail(username)
		if err != nil {
			return err
		}
		return s.userRepo.UpdateFields(user.ID, map[string]any{"role": constants.RoleAdmin})
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), constants.DefaultBcryptCost)
	if err != nil {
		return err
	}
	admin := &model.User{
		Username:     username,
		Email:        email,
		PasswordHash: string(hashed),
		Nickname:     "管理员",
		Role:         constants.RoleAdmin,
		Status:       constants.UserStatusNormal,
		StorageQuota: constants.DefaultStorageQuota,
	}
	if err := s.userRepo.Create(admin); err != nil {
		return err
	}
	s.log.Warn("已创建初始管理员账号，请首次登录后立即修改密码",
		zap.String("username", username), zap.String("email", email))
	return nil
}

// ListUsers 用户列表（分页 + 关键字）。
func (s *AdminService) ListUsers(page, size int, keyword string) (*dto.AdminUserListResp, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	if size > 200 {
		size = 200
	}
	users, total, err := s.userRepo.List(keyword, (page-1)*size, size)
	if err != nil {
		return nil, err
	}
	items := make([]dto.AdminUserItem, 0, len(users))
	for i := range users {
		u := users[i]
		items = append(items, dto.AdminUserItem{
			ID:           u.ID,
			Username:     u.Username,
			Email:        u.Email,
			Nickname:     u.Nickname,
			Role:         u.Role,
			Status:       u.Status,
			StorageUsed:  u.StorageUsed,
			StorageQuota: u.StorageQuota,
			LastLoginAt:  u.LastLoginAt,
			CreatedAt:    u.CreatedAt,
		})
	}
	return &dto.AdminUserListResp{Total: total, Page: page, Size: size, Items: items}, nil
}

// SetStatus 启用 / 禁用用户。
func (s *AdminService) SetStatus(userID uint64, status int8) error {
	if status != constants.UserStatusDisabled && status != constants.UserStatusNormal {
		return errs.ErrBadParam
	}
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		if repository.IsNotFound(err) {
			return errs.ErrBadParam.WithMessage("用户不存在")
		}
		return err
	}
	if user.Role == constants.RoleAdmin && status == constants.UserStatusDisabled {
		return errs.ErrBadParam.WithMessage("不能禁用管理员账号")
	}
	return s.userRepo.UpdateFields(userID, map[string]any{"status": status})
}

// SetQuota 调整用户配额。
func (s *AdminService) SetQuota(userID uint64, quota int64) error {
	if quota < 0 {
		return errs.ErrBadParam.WithMessage("配额不能为负数")
	}
	if _, err := s.userRepo.FindByID(userID); err != nil {
		if repository.IsNotFound(err) {
			return errs.ErrBadParam.WithMessage("用户不存在")
		}
		return err
	}
	return s.userRepo.UpdateFields(userID, map[string]any{"storage_quota": quota})
}

// Stats 系统统计。
func (s *AdminService) Stats() (*dto.AdminStatsResp, error) {
	userCount, err := s.userRepo.Count()
	if err != nil {
		return nil, err
	}
	activeUsers, err := s.userRepo.CountActive()
	if err != nil {
		return nil, err
	}
	fileCount, dirCount, logicalSize, err := s.fileRepo.GlobalCounts()
	if err != nil {
		return nil, err
	}
	physicalSize, err := s.blobRepo.SumSize()
	if err != nil {
		return nil, err
	}
	blobCount, err := s.blobRepo.Count()
	if err != nil {
		return nil, err
	}
	shareCount, err := s.shareRepo.Count()
	if err != nil {
		return nil, err
	}
	activeSessions, err := s.uploadRepo.CountByStatus(constants.SessionUploading)
	if err != nil {
		return nil, err
	}
	return &dto.AdminStatsResp{
		UserCount:       userCount,
		ActiveUserCount: activeUsers,
		FileCount:       fileCount,
		DirCount:        dirCount,
		LogicalSize:     logicalSize,
		PhysicalSize:    physicalSize,
		BlobCount:       blobCount,
		ShareCount:      shareCount,
		ActiveSessions:  activeSessions,
	}, nil
}

// UploadSessions 活跃上传会话监控。
func (s *AdminService) UploadSessions(status *int8, limit int) (*dto.UploadSessionListResp, error) {
	sessions, err := s.uploadRepo.ListByStatus(status, limit)
	if err != nil {
		return nil, err
	}
	userNames := make(map[uint64]string, len(sessions))
	items := make([]dto.UploadSessionItem, 0, len(sessions))
	for i := range sessions {
		sess := sessions[i]
		name, ok := userNames[sess.UserID]
		if !ok {
			if u, err := s.userRepo.FindByID(sess.UserID); err == nil {
				name = u.Username
			}
			userNames[sess.UserID] = name
		}
		items = append(items, dto.UploadSessionItem{
			ID:          sess.ID,
			UploadID:    sess.UploadID,
			UserID:      sess.UserID,
			UserName:    name,
			FileName:    sess.FileName,
			FileSize:    sess.FileSize,
			ChunkSize:   sess.ChunkSize,
			TotalChunks: sess.TotalChunks,
			Status:      sess.Status,
			ExpireAt:    sess.ExpireAt,
			CreatedAt:   sess.CreatedAt,
		})
	}
	return &dto.UploadSessionListResp{Total: int64(len(items)), Items: items}, nil
}

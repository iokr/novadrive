package service

import (
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/iokr/novadrive/internal/config"
	"github.com/iokr/novadrive/internal/constants"
	"github.com/iokr/novadrive/internal/dto"
	"github.com/iokr/novadrive/internal/model"
	"github.com/iokr/novadrive/internal/repository"
	"github.com/iokr/novadrive/pkg/errs"
	"github.com/iokr/novadrive/pkg/jwt"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// usernamePattern 用户名合法字符（字母 / 数字 / 下划线）。
var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_]{3,32}$`)

// AuthService 注册 / 登录 / 个人中心业务。
type AuthService struct {
	cfg      *config.Config
	userRepo *repository.UserRepo
	jwtMgr   *jwt.Manager
	log      *zap.Logger
}

// NewAuthService 创建认证服务。
func NewAuthService(cfg *config.Config, userRepo *repository.UserRepo, jwtMgr *jwt.Manager, log *zap.Logger) *AuthService {
	return &AuthService{cfg: cfg, userRepo: userRepo, jwtMgr: jwtMgr, log: log}
}

// Register 注册新用户。
func (s *AuthService) Register(req *dto.RegisterReq) (*dto.RegisterResp, error) {
	username := strings.TrimSpace(req.Username)
	if !usernamePattern.MatchString(username) {
		return nil, errs.ErrBadParam.WithMessage("用户名需为 3-32 位字母、数字或下划线")
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if _, err := mail.ParseAddress(email); err != nil {
		return nil, errs.ErrBadParam.WithMessage("邮箱格式不正确")
	}
	if len(req.Password) < 6 || len(req.Password) > 64 {
		return nil, errs.ErrBadParam.WithMessage("密码长度需为 6-64 位")
	}

	exists, err := s.userRepo.ExistsUsername(username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errs.ErrUserExists.WithMessage("用户名已存在")
	}
	exists, err = s.userRepo.ExistsEmail(email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errs.ErrUserExists.WithMessage("邮箱已被注册")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), constants.DefaultBcryptCost)
	if err != nil {
		return nil, err
	}
	user := &model.User{
		Username:     username,
		Email:        email,
		PasswordHash: string(hashed),
		Nickname:     username,
		Role:         constants.RoleUser,
		Status:       constants.UserStatusNormal,
		StorageQuota: constants.DefaultStorageQuota,
	}
	if err := s.userRepo.Create(user); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			return nil, errs.ErrUserExists
		}
		return nil, err
	}
	return &dto.RegisterResp{ID: user.ID, Username: user.Username}, nil
}

// Login 登录并签发 JWT。
func (s *AuthService) Login(req *dto.LoginReq) (*dto.LoginResp, error) {
	identifier := req.Identifier()
	if identifier == "" || req.Password == "" {
		return nil, errs.ErrBadParam.WithMessage("请输入用户名/邮箱与密码")
	}
	user, err := s.userRepo.FindByUsernameOrEmail(identifier)
	if err != nil {
		if repository.IsNotFound(err) {
			return nil, errs.ErrCredential
		}
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errs.ErrCredential
	}
	if user.Status != constants.UserStatusNormal {
		return nil, errs.ErrUserDisabled
	}
	token, err := s.jwtMgr.Generate(user.ID, user.Role)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if err := s.userRepo.TouchLogin(user.ID, now); err != nil {
		s.log.Warn("更新登录时间失败", zap.Uint64("user_id", user.ID), zap.Error(err))
	} else {
		user.LastLoginAt = &now
	}
	return &dto.LoginResp{
		Token:     token,
		ExpiresIn: int64(s.jwtMgr.TTL().Seconds()),
		User:      dto.NewUserInfo(user),
	}, nil
}

// Profile 当前用户信息。
func (s *AuthService) Profile(userID uint64) (*dto.UserInfo, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		if repository.IsNotFound(err) {
			return nil, errs.ErrUnauthorized
		}
		return nil, err
	}
	info := dto.NewUserInfo(user)
	return &info, nil
}

// UpdateProfile 修改昵称 / 头像。
func (s *AuthService) UpdateProfile(userID uint64, req *dto.UpdateProfileReq) (*dto.UserInfo, error) {
	fields := make(map[string]any)
	if req.Nickname != nil {
		nickname := strings.TrimSpace(*req.Nickname)
		if len(nickname) > 64 {
			return nil, errs.ErrBadParam.WithMessage("昵称过长")
		}
		fields["nickname"] = nickname
	}
	if req.Avatar != nil {
		avatar := strings.TrimSpace(*req.Avatar)
		if len(avatar) > 255 {
			return nil, errs.ErrBadParam.WithMessage("头像地址过长")
		}
		fields["avatar"] = avatar
	}
	if len(fields) == 0 {
		return s.Profile(userID)
	}
	if err := s.userRepo.UpdateFields(userID, fields); err != nil {
		return nil, err
	}
	return s.Profile(userID)
}

// ChangePassword 修改密码。
func (s *AuthService) ChangePassword(userID uint64, req *dto.ChangePasswordReq) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		if repository.IsNotFound(err) {
			return errs.ErrUnauthorized
		}
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
		return errs.ErrPasswordIncorrect
	}
	if req.OldPassword == req.NewPassword {
		return errs.ErrBadParam.WithMessage("新密码不能与原密码相同")
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), constants.DefaultBcryptCost)
	if err != nil {
		return err
	}
	return s.userRepo.UpdateFields(userID, map[string]any{"password_hash": string(hashed)})
}

// VerifyPassword 校验密码（供管理员等内部场景复用）。
func (s *AuthService) VerifyPassword(hashed, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain)) == nil
}

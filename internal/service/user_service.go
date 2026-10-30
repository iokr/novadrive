package service

import (
	"github.com/iokr/novadrive/internal/constants"
	"github.com/iokr/novadrive/internal/dto"
	"github.com/iokr/novadrive/internal/model"
	"github.com/iokr/novadrive/internal/repository"
	"github.com/iokr/novadrive/pkg/errs"
	"github.com/iokr/novadrive/pkg/fileutil"
)

// UserService 用户相关业务。
type UserService struct {
	userRepo *repository.UserRepo
	fileRepo *repository.FileRepo
}

// NewUserService 创建用户服务。
func NewUserService(userRepo *repository.UserRepo, fileRepo *repository.FileRepo) *UserService {
	return &UserService{userRepo: userRepo, fileRepo: fileRepo}
}

// Get 查询用户，禁用账号视为无权限。
func (s *UserService) Get(userID uint64) (*model.User, error) {
	u, err := s.userRepo.FindByID(userID)
	if err != nil {
		if repository.IsNotFound(err) {
			return nil, errs.ErrUnauthorized
		}
		return nil, err
	}
	if u.Status != constants.UserStatusNormal {
		return nil, errs.ErrUserDisabled
	}
	return u, nil
}

// CheckQuota 校验剩余配额是否足够写入 size 字节。
func (s *UserService) CheckQuota(userID uint64, size int64) error {
	u, err := s.userRepo.FindByID(userID)
	if err != nil {
		if repository.IsNotFound(err) {
			return errs.ErrUnauthorized
		}
		return err
	}
	if size > 0 && u.StorageUsed+size > u.StorageQuota {
		return errs.ErrQuotaExceeded
	}
	return nil
}

// AddStorageUsed 累加已用存储。
func (s *UserService) AddStorageUsed(userID uint64, size int64) error {
	return s.userRepo.AddStorageUsed(userID, size)
}

// SubStorageUsed 扣减已用存储。
func (s *UserService) SubStorageUsed(userID uint64, size int64) error {
	return s.userRepo.SubStorageUsed(userID, size)
}

// Overview 空间统计：已用 / 配额 / 文件数 / 按类型分布。
func (s *UserService) Overview(userID uint64) (*dto.FileStatsResp, error) {
	u, err := s.userRepo.FindByID(userID)
	if err != nil {
		if repository.IsNotFound(err) {
			return nil, errs.ErrUnauthorized
		}
		return nil, err
	}
	rows, err := s.fileRepo.StatsByExt(userID)
	if err != nil {
		return nil, err
	}
	fileCount, dirCount, err := s.fileRepo.Counts(userID)
	if err != nil {
		return nil, err
	}
	agg := make(map[string]*dto.TypeStat, len(constants.StatsCategories))
	for _, c := range constants.StatsCategories {
		agg[c] = &dto.TypeStat{Type: c}
	}
	var totalSize int64
	for _, row := range rows {
		category := fileutil.Category(row.Ext)
		stat, ok := agg[category]
		if !ok {
			stat = &dto.TypeStat{Type: category}
			agg[category] = stat
		}
		stat.Count += row.Count
		stat.Size += row.Size
		totalSize += row.Size
	}
	byType := make([]dto.TypeStat, 0, len(constants.StatsCategories))
	for _, c := range constants.StatsCategories {
		stat := agg[c]
		if stat == nil {
			stat = &dto.TypeStat{Type: c}
		}
		byType = append(byType, *stat)
	}
	return &dto.FileStatsResp{
		Used:       u.StorageUsed,
		Quota:      u.StorageQuota,
		TotalSize:  totalSize,
		TotalCount: fileCount,
		DirCount:   dirCount,
		ByType:     byType,
	}, nil
}

// SpaceInfo 侧边栏空间条所需的精简信息。
type SpaceInfo struct {
	Used      int64 `json:"used"`
	Quota     int64 `json:"quota"`
	FileCount int64 `json:"file_count"`
}

// Space 返回已用 / 配额 / 文件数。
func (s *UserService) Space(userID uint64) (*SpaceInfo, error) {
	u, err := s.userRepo.FindByID(userID)
	if err != nil {
		if repository.IsNotFound(err) {
			return nil, errs.ErrUnauthorized
		}
		return nil, err
	}
	fileCount, _, err := s.fileRepo.Counts(userID)
	if err != nil {
		return nil, err
	}
	return &SpaceInfo{Used: u.StorageUsed, Quota: u.StorageQuota, FileCount: fileCount}, nil
}

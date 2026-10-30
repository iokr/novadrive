package service

import (
	"strings"
	"time"

	"github.com/iokr/novadrive/internal/config"
	"github.com/iokr/novadrive/internal/constants"
	"github.com/iokr/novadrive/internal/dto"
	"github.com/iokr/novadrive/internal/model"
	"github.com/iokr/novadrive/internal/repository"
	"github.com/iokr/novadrive/internal/storage"
	"github.com/iokr/novadrive/pkg/errs"
	"github.com/iokr/novadrive/pkg/random"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// ShareService 分享业务（创建 / 公开访问 / 密码校验 / 下载计数）。
type ShareService struct {
	cfg       *config.Config
	log       *zap.Logger
	shareRepo *repository.ShareRepo
	fileRepo  *repository.FileRepo
	blobSvc   *BlobService
	storage   storage.Storage
	secret    string
}

// NewShareService 创建分享服务。
func NewShareService(
	cfg *config.Config,
	log *zap.Logger,
	shareRepo *repository.ShareRepo,
	fileRepo *repository.FileRepo,
	blobSvc *BlobService,
	st storage.Storage,
	secret string,
) *ShareService {
	return &ShareService{
		cfg:       cfg,
		log:       log,
		shareRepo: shareRepo,
		fileRepo:  fileRepo,
		blobSvc:   blobSvc,
		storage:   st,
		secret:    secret,
	}
}

// AccessKey 计算分享访问签名（用于校验提取码通过后的后续请求）。
func (s *ShareService) AccessKey(token string) string {
	return random.Sign(s.secret, "share:"+token)
}

// shareURL 生成分享访问地址。
func (s *ShareService) shareURL(token string) string {
	return strings.TrimRight(s.cfg.Server.BaseURL, "/") + "/s/" + token
}

// genToken 生成指定长度的分享码。
func (s *ShareService) genToken() (string, error) {
	length := s.cfg.Share.TokenLength
	if length < 16 {
		length = 16
	}
	bytesNeeded := (length + 1) / 2
	token, err := random.Token(bytesNeeded)
	if err != nil {
		return "", err
	}
	return token[:length], nil
}

// findShare 查询有效分享（自动标记过期）。
func (s *ShareService) findShare(token string) (*model.Share, error) {
	share, err := s.shareRepo.FindByToken(token)
	if err != nil {
		if repository.IsNotFound(err) {
			return nil, errs.ErrShareInvalid
		}
		return nil, err
	}
	if share.Status == constants.ShareStatusCanceled {
		return nil, errs.ErrShareInvalid.WithMessage("分享已被取消")
	}
	if share.IsExpired(time.Now()) {
		if share.Status == constants.ShareStatusNormal {
			_ = s.shareRepo.UpdateFields(share.ID, map[string]any{"status": constants.ShareStatusExpired})
		}
		return nil, errs.ErrShareInvalid.WithMessage("分享已过期")
	}
	if share.Status == constants.ShareStatusExpired {
		return nil, errs.ErrShareInvalid.WithMessage("分享已过期")
	}
	return share, nil
}

// shareFile 加载被分享的条目（要求正常状态）。
func (s *ShareService) shareFile(share *model.Share) (*model.File, error) {
	file, err := s.fileRepo.FindByIDAndUser(share.FileID, share.UserID)
	if err != nil {
		if repository.IsNotFound(err) {
			return nil, errs.ErrShareInvalid.WithMessage("分享内容已被删除")
		}
		return nil, err
	}
	if file.Status != constants.FileStatusNormal {
		return nil, errs.ErrShareInvalid.WithMessage("分享内容已被删除")
	}
	return file, nil
}

// Create 创建分享。
func (s *ShareService) Create(userID uint64, req *dto.CreateShareReq) (*dto.CreateShareResp, error) {
	file, err := s.fileRepo.FindByIDAndUser(req.FileID, userID)
	if err != nil {
		if repository.IsNotFound(err) {
			return nil, errs.ErrFileNotFound
		}
		return nil, err
	}
	if file.Status != constants.FileStatusNormal {
		return nil, errs.ErrFileNotFound
	}

	var (
		hashedPassword string
		plainPassword  string
	)
	if pwd := strings.TrimSpace(req.Password); pwd != "" {
		if len(pwd) < 4 || len(pwd) > 32 {
			return nil, errs.ErrBadParam.WithMessage("提取码长度需为 4-32 位")
		}
		hashed, err := bcrypt.GenerateFromPassword([]byte(pwd), constants.DefaultBcryptCost)
		if err != nil {
			return nil, err
		}
		hashedPassword = string(hashed)
		plainPassword = pwd
	}

	var expireAt *time.Time
	days := req.ExpireDays
	if days <= 0 {
		days = s.cfg.Share.DefaultExpireDays
	}
	if days > 0 {
		t := time.Now().AddDate(0, 0, days)
		expireAt = &t
	}
	if req.MaxDownloads < 0 {
		return nil, errs.ErrBadParam.WithMessage("下载次数上限不能为负数")
	}

	shareType := constants.ShareTypeFile
	if file.IsDir == constants.IsDir {
		shareType = constants.ShareTypeDir
	}

	var token string
	for attempt := 0; attempt < 5; attempt++ {
		token, err = s.genToken()
		if err != nil {
			return nil, err
		}
		if _, findErr := s.shareRepo.FindByToken(token); repository.IsNotFound(findErr) {
			break
		}
	}

	share := &model.Share{
		UserID:       userID,
		FileID:       file.ID,
		Token:        token,
		Type:         shareType,
		Password:     hashedPassword,
		ExpireAt:     expireAt,
		MaxDownloads: req.MaxDownloads,
		Status:       constants.ShareStatusNormal,
	}
	if err := s.shareRepo.Create(share); err != nil {
		return nil, err
	}
	return &dto.CreateShareResp{
		ID:          share.ID,
		Token:       share.Token,
		URL:         s.shareURL(share.Token),
		HasPassword: share.HasPassword(),
		Password:    plainPassword,
		ExpireAt:    share.ExpireAt,
	}, nil
}

// List 我的分享列表。
func (s *ShareService) List(userID uint64) ([]dto.ShareItem, error) {
	shares, err := s.shareRepo.ListByUser(userID)
	if err != nil {
		return nil, err
	}
	items := make([]dto.ShareItem, 0, len(shares))
	now := time.Now()
	for i := range shares {
		share := shares[i]
		status := share.Status
		if status == constants.ShareStatusNormal && share.IsExpired(now) {
			status = constants.ShareStatusExpired
			_ = s.shareRepo.UpdateFields(share.ID, map[string]any{"status": constants.ShareStatusExpired})
		}
		fileName := ""
		var isDir int8
		if file, err := s.fileRepo.FindByIDAndUser(share.FileID, userID); err == nil {
			fileName = file.Name
			isDir = file.IsDir
		}
		items = append(items, dto.ShareItem{
			ID:            share.ID,
			Token:         share.Token,
			URL:           s.shareURL(share.Token),
			FileID:        share.FileID,
			FileName:      fileName,
			IsDir:         isDir,
			Type:          share.Type,
			HasPassword:   share.HasPassword(),
			ExpireAt:      share.ExpireAt,
			MaxDownloads:  share.MaxDownloads,
			DownloadCount: share.DownloadCount,
			ViewCount:     share.ViewCount,
			Status:        status,
			CreatedAt:     share.CreatedAt,
		})
	}
	return items, nil
}

// Cancel 取消分享。
func (s *ShareService) Cancel(userID, shareID uint64) error {
	share, err := s.shareRepo.FindByID(shareID)
	if err != nil {
		if repository.IsNotFound(err) {
			return errs.ErrShareInvalid
		}
		return err
	}
	if share.UserID != userID {
		return errs.ErrShareInvalid
	}
	return s.shareRepo.UpdateFields(shareID, map[string]any{"status": constants.ShareStatusCanceled})
}

// publicInfo 构造公开分享信息。
func (s *ShareService) publicInfo(share *model.Share, file *model.File) dto.PublicShareInfo {
	return dto.PublicShareInfo{
		Token:         share.Token,
		FileName:      file.Name,
		IsDir:         file.IsDir,
		FileSize:      file.Size,
		Type:          share.Type,
		HasPassword:   share.HasPassword(),
		ExpireAt:      share.ExpireAt,
		ViewCount:     share.ViewCount,
		DownloadCount: share.DownloadCount,
		MaxDownloads:  share.MaxDownloads,
		CreatedAt:     share.CreatedAt,
	}
}

// PublicInfo 公开分享基本信息。
func (s *ShareService) PublicInfo(token string) (*dto.PublicShareInfo, error) {
	share, err := s.findShare(token)
	if err != nil {
		return nil, err
	}
	file, err := s.shareFile(share)
	if err != nil {
		return nil, err
	}
	if err := s.shareRepo.IncView(share.ID); err != nil {
		s.log.Warn("分享访问计数失败", zap.Uint64("share_id", share.ID), zap.Error(err))
	}
	info := s.publicInfo(share, file)
	return &info, nil
}

// Verify 校验提取码并返回 access_key。
func (s *ShareService) Verify(token, password string) (*dto.VerifyResp, error) {
	share, err := s.findShare(token)
	if err != nil {
		return nil, err
	}
	if _, err := s.shareFile(share); err != nil {
		return nil, err
	}
	if share.HasPassword() {
		if strings.TrimSpace(password) == "" {
			return nil, errs.ErrSharePassword
		}
		if err := bcrypt.CompareHashAndPassword([]byte(share.Password), []byte(password)); err != nil {
			return nil, errs.ErrSharePassword
		}
	}
	return &dto.VerifyResp{AccessKey: s.AccessKey(share.Token)}, nil
}

// checkAccess 校验访问签名。
func (s *ShareService) checkAccess(share *model.Share, accessKey string) error {
	if !share.HasPassword() {
		return nil
	}
	if accessKey == "" || !random.EqualSign(accessKey, s.AccessKey(share.Token)) {
		return errs.ErrSharePassword
	}
	return nil
}

// withinRoot 判断 nodeID 是否位于 rootID 子树内。
func (s *ShareService) withinRoot(root *model.File, nodeID uint64) (bool, error) {
	if nodeID == root.ID {
		return true, nil
	}
	current := nodeID
	for depth := 0; depth < constants.MaxDirDepth && current != constants.RootParentID; depth++ {
		node, err := s.fileRepo.FindByIDAndUser(current, root.UserID)
		if err != nil {
			if repository.IsNotFound(err) {
				return false, errs.ErrShareInvalid
			}
			return false, err
		}
		if node.Status != constants.FileStatusNormal {
			return false, errs.ErrShareInvalid.WithMessage("分享内容已被删除")
		}
		if node.ParentID == root.ID {
			return true, nil
		}
		current = node.ParentID
	}
	return false, nil
}

// PublicFiles 分享内容浏览（目录分享逐层浏览）。
func (s *ShareService) PublicFiles(token, accessKey string, parentID uint64) (*dto.PublicFileList, error) {
	share, err := s.findShare(token)
	if err != nil {
		return nil, err
	}
	if err := s.checkAccess(share, accessKey); err != nil {
		return nil, err
	}
	root, err := s.shareFile(share)
	if err != nil {
		return nil, err
	}
	info := s.publicInfo(share, root)

	if root.IsDir != constants.IsDir {
		return &dto.PublicFileList{
			Share:   info,
			RootID:  root.ID,
			Current: root.ID,
			Total:   1,
			Items:   []dto.FileItem{fileItem(root)},
		}, nil
	}
	if parentID == constants.RootParentID {
		parentID = root.ID
	}
	ok, err := s.withinRoot(root, parentID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errs.ErrForbidden.WithMessage("无权访问该目录")
	}
	children, err := s.fileRepo.ListByParent(root.UserID, parentID, "name", "asc")
	if err != nil {
		return nil, err
	}
	return &dto.PublicFileList{
		Share:   info,
		RootID:  root.ID,
		Current: parentID,
		Total:   len(children),
		Items:   fileItems(children),
	}, nil
}

// PublicTarget 分享内预览 / 下载目标（目录返回 nil blob，由调用方打包 zip）。
func (s *ShareService) PublicTarget(token, accessKey string, fileID uint64) (*dto.PublicShareInfo, *model.File, *model.Blob, error) {
	share, err := s.findShare(token)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := s.checkAccess(share, accessKey); err != nil {
		return nil, nil, nil, err
	}
	root, err := s.shareFile(share)
	if err != nil {
		return nil, nil, nil, err
	}
	if fileID == constants.RootParentID {
		fileID = root.ID
	}
	ok, err := s.withinRoot(root, fileID)
	if err != nil {
		return nil, nil, nil, err
	}
	if !ok {
		return nil, nil, nil, errs.ErrForbidden.WithMessage("无权访问该分享内容")
	}
	target, err := s.fileRepo.FindByIDAndUser(fileID, root.UserID)
	if err != nil {
		if repository.IsNotFound(err) {
			return nil, nil, nil, errs.ErrFileNotFound
		}
		return nil, nil, nil, err
	}
	if target.Status != constants.FileStatusNormal {
		return nil, nil, nil, errs.ErrShareInvalid.WithMessage("分享内容已被删除")
	}
	info := s.publicInfo(share, root)
	if target.IsDir == constants.IsDir {
		return &info, target, nil, nil
	}
	blob, err := s.blobSvc.FindByID(target.BlobID)
	if err != nil {
		return nil, nil, nil, err
	}
	return &info, target, blob, nil
}

// RecordDownload 记录下载次数并返回目标（超限返回 4001）。
func (s *ShareService) RecordDownload(token, accessKey string, fileID uint64) (*dto.PublicShareInfo, *model.File, *model.Blob, error) {
	info, file, blob, err := s.PublicTarget(token, accessKey, fileID)
	if err != nil {
		return nil, nil, nil, err
	}
	share, err := s.findShare(token)
	if err != nil {
		return nil, nil, nil, err
	}
	if share.ReachDownloadLimit() {
		return nil, nil, nil, errs.ErrShareInvalid.WithMessage("分享下载次数已达上限")
	}
	ok, err := s.shareRepo.IncDownload(share.ID)
	if err != nil {
		return nil, nil, nil, err
	}
	if !ok {
		return nil, nil, nil, errs.ErrShareInvalid.WithMessage("分享下载次数已达上限")
	}
	return info, file, blob, nil
}

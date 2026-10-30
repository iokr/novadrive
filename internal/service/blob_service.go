package service

import (
	"io"

	"github.com/iokr/novadrive/internal/constants"
	"github.com/iokr/novadrive/internal/model"
	"github.com/iokr/novadrive/internal/repository"
	"github.com/iokr/novadrive/internal/storage"
	"github.com/iokr/novadrive/pkg/errs"
	"github.com/iokr/novadrive/pkg/fileutil"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// BlobService 物理文件（Blob）与逻辑文件落库的统一入口。
type BlobService struct {
	blobRepo *repository.BlobRepo
	fileRepo *repository.FileRepo
	userRepo *repository.UserRepo
	storage  storage.Storage
	thumb    *ThumbService
	log      *zap.Logger
}

// NewBlobService 创建 Blob 服务。
func NewBlobService(
	blobRepo *repository.BlobRepo,
	fileRepo *repository.FileRepo,
	userRepo *repository.UserRepo,
	st storage.Storage,
	thumb *ThumbService,
	log *zap.Logger,
) *BlobService {
	return &BlobService{
		blobRepo: blobRepo,
		fileRepo: fileRepo,
		userRepo: userRepo,
		storage:  st,
		thumb:    thumb,
		log:      log,
	}
}

// PersistTemp 统一入库入口：临时文件 → (命中则引用+1 / 未命中则落盘入库) → files + 配额。
func (s *BlobService) PersistTemp(userID, parentID uint64, name, tmpPath, md5sum string, size int64) (*model.File, error) {
	if err := s.checkQuota(userID, size); err != nil {
		_ = s.storage.DeleteTemp(tmpPath)
		return nil, err
	}
	ext := fileutil.Ext(name)
	mimeType := fileutil.MimeByExt(ext)
	var (
		file    *model.File
		newBlob *model.Blob
	)
	err := s.fileRepo.DB().Transaction(func(tx *gorm.DB) error {
		blob, created, err := s.resolveBlob(tx, tmpPath, md5sum, ext, mimeType, size)
		if err != nil {
			return err
		}
		finalName, err := uniqueName(s.fileRepo.WithTx(tx), userID, parentID, name)
		if err != nil {
			return err
		}
		f := &model.File{
			UserID:   userID,
			ParentID: parentID,
			Name:     finalName,
			IsDir:    constants.IsFile,
			Ext:      ext,
			Size:     size,
			MimeType: mimeType,
			BlobID:   blob.ID,
			Status:   constants.FileStatusNormal,
		}
		if err := s.fileRepo.WithTx(tx).Create(f); err != nil {
			return err
		}
		if err := s.userRepo.WithTx(tx).AddStorageUsed(userID, size); err != nil {
			return err
		}
		file = f
		if created {
			newBlob = blob
		}
		return nil
	})
	if err != nil {
		_ = s.storage.DeleteTemp(tmpPath)
		return nil, err
	}
	if newBlob != nil {
		s.thumb.GenerateAsync(newBlob)
	}
	return file, nil
}

// resolveBlob 在事务内解析物理文件：命中则引用计数 +1，未命中则移动文件并插入记录。
func (s *BlobService) resolveBlob(tx *gorm.DB, tmpPath, md5sum, ext, mimeType string, size int64) (*model.Blob, bool, error) {
	blobRepo := s.blobRepo.WithTx(tx)
	var existing model.Blob
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("hash_md5 = ?", md5sum).First(&existing).Error
	switch {
	case err == nil:
		if err := blobRepo.IncRef(existing.ID); err != nil {
			return nil, false, err
		}
		existing.RefCount++
		return &existing, false, nil
	case repository.IsNotFound(err):
		relPath, moveErr := s.storage.MoveToBlob(tmpPath, md5sum, ext)
		if moveErr != nil {
			return nil, false, moveErr
		}
		blob := &model.Blob{
			HashMD5:     md5sum,
			Size:        size,
			Ext:         ext,
			MimeType:    mimeType,
			StoragePath: relPath,
			RefCount:    1,
		}
		if createErr := blobRepo.Create(blob); createErr != nil {
			return nil, false, createErr
		}
		return blob, true, nil
	default:
		return nil, false, err
	}
}

// AttachExisting 秒传：复用已存在的物理文件，仅创建逻辑文件记录。
func (s *BlobService) AttachExisting(userID, parentID uint64, name string, blob *model.Blob) (*model.File, error) {
	if err := s.checkQuota(userID, blob.Size); err != nil {
		return nil, err
	}
	ext := fileutil.Ext(name)
	mimeType := blob.MimeType
	if mimeType == "" {
		mimeType = fileutil.MimeByExt(ext)
	}
	var file *model.File
	err := s.fileRepo.DB().Transaction(func(tx *gorm.DB) error {
		finalName, err := uniqueName(s.fileRepo.WithTx(tx), userID, parentID, name)
		if err != nil {
			return err
		}
		if err := s.blobRepo.WithTx(tx).IncRef(blob.ID); err != nil {
			return err
		}
		f := &model.File{
			UserID:   userID,
			ParentID: parentID,
			Name:     finalName,
			IsDir:    constants.IsFile,
			Ext:      ext,
			Size:     blob.Size,
			MimeType: mimeType,
			BlobID:   blob.ID,
			Status:   constants.FileStatusNormal,
		}
		if err := s.fileRepo.WithTx(tx).Create(f); err != nil {
			return err
		}
		if err := s.userRepo.WithTx(tx).AddStorageUsed(userID, blob.Size); err != nil {
			return err
		}
		file = f
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.thumb.GenerateAsync(blob)
	return file, nil
}

// checkQuota 校验剩余配额。
func (s *BlobService) checkQuota(userID uint64, size int64) error {
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

// FindByHash 按 MD5 查找物理文件（秒传预检）。
func (s *BlobService) FindByHash(md5sum string) (*model.Blob, error) {
	blob, err := s.blobRepo.FindByHash(md5sum)
	if err != nil {
		if repository.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return blob, nil
}

// FindByID 按主键查找物理文件。
func (s *BlobService) FindByID(id uint64) (*model.Blob, error) {
	blob, err := s.blobRepo.FindByID(id)
	if err != nil {
		if repository.IsNotFound(err) {
			return nil, errs.ErrFileNotFound
		}
		return nil, err
	}
	return blob, nil
}

// OpenBlob 打开物理文件用于读取（支持 Range）。
func (s *BlobService) OpenBlob(blob *model.Blob) (io.ReadSeeker, error) {
	return s.storage.Open(blob.StoragePath)
}

package service

import (
	"io"
	"math"
	"sync"
	"time"

	"github.com/iokr/novadrive/internal/config"
	"github.com/iokr/novadrive/internal/constants"
	"github.com/iokr/novadrive/internal/dto"
	"github.com/iokr/novadrive/internal/model"
	"github.com/iokr/novadrive/internal/repository"
	"github.com/iokr/novadrive/internal/storage"
	"github.com/iokr/novadrive/pkg/errs"
	"github.com/iokr/novadrive/pkg/fileutil"
	"github.com/iokr/novadrive/pkg/random"
	"go.uber.org/zap"
)

// UploadService 分片上传：会话创建 / 分片接收 / 合并 / 断点续传。
type UploadService struct {
	cfg        *config.Config
	log        *zap.Logger
	storage    storage.Storage
	uploadRepo *repository.UploadRepo
	fileRepo   *repository.FileRepo
	blobSvc    *BlobService
	userSvc    *UserService
	locks      sync.Map
}

// NewUploadService 创建分片上传服务。
func NewUploadService(
	cfg *config.Config,
	log *zap.Logger,
	st storage.Storage,
	uploadRepo *repository.UploadRepo,
	fileRepo *repository.FileRepo,
	blobSvc *BlobService,
	userSvc *UserService,
) *UploadService {
	return &UploadService{
		cfg:        cfg,
		log:        log,
		storage:    st,
		uploadRepo: uploadRepo,
		fileRepo:   fileRepo,
		blobSvc:    blobSvc,
		userSvc:    userSvc,
	}
}

// lockFor 获取指定会话的进程内互斥锁。
func (s *UploadService) lockFor(uploadID string) *sync.Mutex {
	actual, _ := s.locks.LoadOrStore(uploadID, &sync.Mutex{})
	return actual.(*sync.Mutex)
}

// session 查询会话并校验归属 / 过期。
func (s *UploadService) session(userID uint64, uploadID string) (*model.UploadSession, error) {
	sess, err := s.uploadRepo.FindByUploadID(uploadID)
	if err != nil {
		if repository.IsNotFound(err) {
			return nil, errs.ErrSessionNotFound
		}
		return nil, err
	}
	if sess.UserID != userID {
		return nil, errs.ErrSessionNotFound
	}
	if sess.Status == constants.SessionUploading && time.Now().After(sess.ExpireAt) {
		_ = s.uploadRepo.UpdateStatus(sess.ID, constants.SessionExpired)
		_ = s.storage.DeleteChunkDir(sess.UploadID)
		return nil, errs.ErrSessionNotFound.WithMessage("上传会话已过期")
	}
	return sess, nil
}

// checkParent 校验目标父目录。
func (s *UploadService) checkParent(userID, parentID uint64) error {
	if parentID == constants.RootParentID {
		return nil
	}
	parent, err := s.fileRepo.FindByIDAndUser(parentID, userID)
	if err != nil {
		if repository.IsNotFound(err) {
			return errs.ErrInvalidTarget
		}
		return err
	}
	if parent.IsDir != constants.IsDir || parent.Status != constants.FileStatusNormal {
		return errs.ErrInvalidTarget
	}
	return nil
}

// Init 初始化分片上传会话（含秒传检测、配额校验与会话复用）。
func (s *UploadService) Init(userID uint64, req *dto.InitUploadReq) (*dto.InitUploadResp, error) {
	if !s.cfg.Storage.Chunk.Enabled {
		return nil, errs.ErrBadParam.WithMessage("分片上传未启用")
	}
	fileName, err := fileutil.SanitizeName(req.FileName)
	if err != nil {
		return nil, errs.ErrInvalidName.WithMessage("%s", err.Error())
	}
	ext := fileutil.Ext(fileName)
	if !fileutil.IsAllowedExt(ext, s.cfg.Storage.Upload.AllowedExt) {
		return nil, errs.ErrExtNotAllowed.WithMessage("扩展名 %s 不允许上传", ext)
	}
	if req.FileSize < 0 || req.FileSize > s.cfg.Storage.Upload.MaxFileSize {
		return nil, errs.ErrFileTooLarge
	}
	chunkSize := req.ChunkSize
	if chunkSize <= 0 || chunkSize > s.cfg.Storage.Chunk.MaxChunkSize {
		return nil, errs.ErrChunkInvalid.WithMessage("分片大小需在 1 ~ %d 之间", s.cfg.Storage.Chunk.MaxChunkSize)
	}
	expectedChunks := int(math.Ceil(float64(req.FileSize) / float64(chunkSize)))
	if expectedChunks == 0 {
		expectedChunks = 1
	}
	if req.TotalChunks != expectedChunks {
		return nil, errs.ErrChunkInvalid.WithMessage("分片总数应为 %d", expectedChunks)
	}
	if err := s.checkParent(userID, req.ParentID); err != nil {
		return nil, err
	}

	hashMD5 := normalizeMD5(req.HashMD5)
	if hashMD5 != "" {
		blob, err := s.blobSvc.FindByHash(hashMD5)
		if err != nil {
			return nil, err
		}
		if blob != nil {
			file, err := s.blobSvc.AttachExisting(userID, req.ParentID, fileName, blob)
			if err != nil {
				return nil, err
			}
			return &dto.InitUploadResp{Instant: true, File: fileItem(file)}, nil
		}
	}

	if err := s.userSvc.CheckQuota(userID, req.FileSize); err != nil {
		return nil, err
	}

	old, err := s.uploadRepo.FindResumable(userID, req.ParentID, fileName, req.FileSize, hashMD5)
	if err != nil && !repository.IsNotFound(err) {
		return nil, err
	}
	if old != nil {
		indices, listErr := s.storage.ListChunks(old.UploadID)
		if listErr != nil {
			s.log.Warn("扫描已上传分片失败", zap.String("upload_id", old.UploadID), zap.Error(listErr))
			indices = nil
		}
		return &dto.InitUploadResp{
			UploadID:       old.UploadID,
			Instant:        false,
			UploadedChunks: indices,
			ChunkSize:      old.ChunkSize,
			TotalChunks:    old.TotalChunks,
		}, nil
	}

	uploadID := random.UUID()
	absDir, err := s.storage.ChunkDir(uploadID)
	if err != nil {
		return nil, err
	}
	session := &model.UploadSession{
		UploadID:    uploadID,
		UserID:      userID,
		ParentID:    req.ParentID,
		FileName:    fileName,
		FileSize:    req.FileSize,
		HashMD5:     hashMD5,
		ChunkSize:   int(chunkSize),
		TotalChunks: expectedChunks,
		Status:      constants.SessionUploading,
		ChunkDir:    absDir,
		ExpireAt:    time.Now().Add(s.cfg.SessionTTL()),
	}
	if err := s.uploadRepo.Create(session); err != nil {
		_ = s.storage.DeleteChunkDir(uploadID)
		return nil, err
	}
	return &dto.InitUploadResp{
		UploadID:    uploadID,
		Instant:     false,
		ChunkSize:   session.ChunkSize,
		TotalChunks: session.TotalChunks,
	}, nil
}

// UploadChunk 接收单个分片（幂等覆盖写）。
func (s *UploadService) UploadChunk(userID uint64, uploadID string, index int, body io.Reader, contentLength int64) (*dto.ChunkResp, error) {
	sess, err := s.session(userID, uploadID)
	if err != nil {
		return nil, err
	}
	if sess.Status != constants.SessionUploading {
		return nil, errs.ErrSessionNotFound.WithMessage("上传会话已结束")
	}
	if index < 0 || index >= sess.TotalChunks {
		return nil, errs.ErrChunkInvalid.WithMessage("分片索引需在 0 ~ %d 之间", sess.TotalChunks-1)
	}
	expected := sess.ExpectedChunkSize(index)
	if expected < 0 {
		return nil, errs.ErrChunkInvalid
	}
	if contentLength >= 0 && contentLength != expected {
		return nil, errs.ErrChunkInvalid.WithMessage("分片 %d 大小应为 %d 字节", index, expected)
	}
	size, err := s.storage.SaveChunk(uploadID, index, body)
	if err != nil {
		return nil, err
	}
	if size != expected {
		return nil, errs.ErrChunkInvalid.WithMessage("分片 %d 大小应为 %d 字节，实际 %d 字节", index, expected, size)
	}
	return &dto.ChunkResp{Index: index, Received: true}, nil
}

// Status 查询上传进度（断点续传）。
func (s *UploadService) Status(userID uint64, uploadID string) (*dto.UploadStatusResp, error) {
	sess, err := s.session(userID, uploadID)
	if err != nil {
		return nil, err
	}
	indices, err := s.storage.ListChunks(uploadID)
	if err != nil {
		return nil, err
	}
	var uploadedBytes int64
	for _, idx := range indices {
		if idx >= 0 && idx < sess.TotalChunks {
			uploadedBytes += sess.ExpectedChunkSize(idx)
		}
	}
	return &dto.UploadStatusResp{
		UploadID:       sess.UploadID,
		Status:         sess.Status,
		FileName:       sess.FileName,
		FileSize:       sess.FileSize,
		UploadedChunks: indices,
		TotalChunks:    sess.TotalChunks,
		UploadedBytes:  uploadedBytes,
		ExpireAt:       sess.ExpireAt,
	}, nil
}

// Complete 校验分片完整性 → 合并 → 统一入库。
func (s *UploadService) Complete(userID uint64, uploadID, hashMD5 string) (*dto.CompleteResp, error) {
	mu := s.lockFor(uploadID)
	mu.Lock()
	defer mu.Unlock()

	sess, err := s.session(userID, uploadID)
	if err != nil {
		return nil, err
	}
	if sess.Status == constants.SessionMerged {
		file, err := s.findMergedFile(sess)
		if err != nil {
			return nil, err
		}
		return &dto.CompleteResp{File: fileItem(file)}, nil
	}
	if sess.Status != constants.SessionUploading {
		return nil, errs.ErrSessionNotFound.WithMessage("上传会话已结束")
	}

	indices, err := s.storage.ListChunks(uploadID)
	if err != nil {
		return nil, err
	}
	present := make(map[int]bool, len(indices))
	for _, idx := range indices {
		present[idx] = true
	}
	missing := make([]int, 0)
	for i := 0; i < sess.TotalChunks; i++ {
		if !present[i] {
			missing = append(missing, i)
			if len(missing) >= 20 {
				break
			}
		}
	}
	if len(missing) > 0 {
		return nil, errs.ErrChunksIncomplete.WithMessage("缺少分片: %v", missing)
	}

	tmpFile, err := s.storage.CreateTempFile()
	if err != nil {
		return nil, err
	}
	tmpPath := tmpFile.Name()
	mergedMD5, totalSize, mergeErr := s.storage.MergeChunks(uploadID, sess.TotalChunks, tmpFile)
	closeErr := tmpFile.Close()
	if mergeErr != nil {
		_ = s.storage.DeleteTemp(tmpPath)
		return nil, mergeErr
	}
	if closeErr != nil {
		_ = s.storage.DeleteTemp(tmpPath)
		return nil, closeErr
	}
	if totalSize != sess.FileSize {
		_ = s.storage.DeleteTemp(tmpPath)
		return nil, errs.ErrChunksIncomplete.WithMessage("合并后大小 %d 与声明 %d 不一致", totalSize, sess.FileSize)
	}

	declared := normalizeMD5(hashMD5)
	if declared == "" {
		declared = normalizeMD5(sess.HashMD5)
	}
	if declared != "" && declared != mergedMD5 {
		_ = s.storage.DeleteTemp(tmpPath)
		return nil, errs.ErrMD5Mismatch
	}

	file, err := s.blobSvc.PersistTemp(userID, sess.ParentID, sess.FileName, tmpPath, mergedMD5, totalSize)
	if err != nil {
		return nil, err
	}
	if _, err := s.uploadRepo.MarkMerged(sess.ID); err != nil {
		s.log.Warn("标记会话已合并失败", zap.String("upload_id", uploadID), zap.Error(err))
	}
	if err := s.storage.DeleteChunkDir(uploadID); err != nil {
		s.log.Warn("清理分片目录失败", zap.String("upload_id", uploadID), zap.Error(err))
	}
	s.locks.Delete(uploadID)
	return &dto.CompleteResp{File: fileItem(file)}, nil
}

// findMergedFile 幂等返回已合并的入库文件。
func (s *UploadService) findMergedFile(sess *model.UploadSession) (*model.File, error) {
	if f, err := s.fileRepo.FindByName(sess.UserID, sess.ParentID, sess.FileName); err == nil {
		return f, nil
	}
	if f, err := s.fileRepo.FindLatestBySize(sess.UserID, sess.ParentID, sess.FileSize); err == nil {
		return f, nil
	}
	return nil, errs.ErrSessionNotFound.WithMessage("会话已合并，但未找到入库文件")
}

// Cancel 取消会话并清理分片。
func (s *UploadService) Cancel(userID uint64, uploadID string) error {
	sess, err := s.uploadRepo.FindByUploadID(uploadID)
	if err != nil {
		if repository.IsNotFound(err) {
			return errs.ErrSessionNotFound
		}
		return err
	}
	if sess.UserID != userID {
		return errs.ErrSessionNotFound
	}
	if sess.Status == constants.SessionMerged {
		return errs.ErrBadParam.WithMessage("上传已完成，无法取消")
	}
	if err := s.storage.DeleteChunkDir(uploadID); err != nil {
		s.log.Warn("清理分片目录失败", zap.String("upload_id", uploadID), zap.Error(err))
	}
	if err := s.uploadRepo.UpdateStatus(sess.ID, constants.SessionCanceled); err != nil {
		return err
	}
	s.locks.Delete(uploadID)
	return nil
}

// InitChunkSize 依据配置计算默认分片大小（供前端参考）。
func (s *UploadService) InitChunkSize() int64 {
	return s.cfg.Storage.Chunk.DefaultChunkSize
}

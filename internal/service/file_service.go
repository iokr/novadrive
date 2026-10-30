package service

import (
	"mime/multipart"
	"path"
	"strings"
	"time"

	"github.com/iokr/novadrive/internal/config"
	"github.com/iokr/novadrive/internal/constants"
	"github.com/iokr/novadrive/internal/dto"
	"github.com/iokr/novadrive/internal/model"
	"github.com/iokr/novadrive/internal/repository"
	"github.com/iokr/novadrive/internal/storage"
	"github.com/iokr/novadrive/pkg/errs"
	"github.com/iokr/novadrive/pkg/fileutil"
	"github.com/iokr/novadrive/pkg/hash"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// FileService 文件与目录业务。
type FileService struct {
	cfg      *config.Config
	log      *zap.Logger
	storage  storage.Storage
	fileRepo *repository.FileRepo
	blobRepo *repository.BlobRepo
	userRepo *repository.UserRepo
	userSvc  *UserService
	blobSvc  *BlobService
}

// NewFileService 创建文件服务。
func NewFileService(
	cfg *config.Config,
	log *zap.Logger,
	st storage.Storage,
	fileRepo *repository.FileRepo,
	blobRepo *repository.BlobRepo,
	userRepo *repository.UserRepo,
	userSvc *UserService,
	blobSvc *BlobService,
) *FileService {
	return &FileService{
		cfg:      cfg,
		log:      log,
		storage:  st,
		fileRepo: fileRepo,
		blobRepo: blobRepo,
		userRepo: userRepo,
		userSvc:  userSvc,
		blobSvc:  blobSvc,
	}
}

// getFile 查询用户拥有的正常状态条目。
func (s *FileService) getFile(userID, fileID uint64) (*model.File, error) {
	f, err := s.fileRepo.FindByIDAndUser(fileID, userID)
	if err != nil {
		if repository.IsNotFound(err) {
			return nil, errs.ErrFileNotFound
		}
		return nil, err
	}
	if f.Status != constants.FileStatusNormal {
		return nil, errs.ErrFileNotFound
	}
	return f, nil
}

// getDir 校验目录存在且为目录；parentID 为 0 时表示根目录。
func (s *FileService) getDir(userID, dirID uint64) (*model.File, error) {
	if dirID == constants.RootParentID {
		return nil, nil
	}
	f, err := s.getFile(userID, dirID)
	if err != nil {
		return nil, err
	}
	if f.IsDir != constants.IsDir {
		return nil, errs.ErrInvalidTarget
	}
	return f, nil
}

// List 目录浏览 / 关键字搜索。
func (s *FileService) List(userID, parentID uint64, keyword, sort, order string) (*dto.FileListResp, error) {
	if kw := strings.TrimSpace(keyword); kw != "" {
		files, err := s.fileRepo.Search(userID, kw, sort, order, 500)
		if err != nil {
			return nil, err
		}
		return &dto.FileListResp{ParentID: parentID, Total: len(files), Items: fileItems(files)}, nil
	}
	if _, err := s.getDir(userID, parentID); err != nil {
		return nil, err
	}
	files, err := s.fileRepo.ListByParent(userID, parentID, sort, order)
	if err != nil {
		return nil, err
	}
	return &dto.FileListResp{ParentID: parentID, Total: len(files), Items: fileItems(files)}, nil
}

// Breadcrumb 面包屑（含根节点 ID=0）。
func (s *FileService) Breadcrumb(userID, parentID uint64) ([]dto.BreadcrumbItem, error) {
	items := []dto.BreadcrumbItem{{ID: constants.RootParentID, Name: "我的文件"}}
	if parentID == constants.RootParentID {
		return items, nil
	}
	chain := make([]dto.BreadcrumbItem, 0, 8)
	current := parentID
	for depth := 0; depth < constants.MaxDirDepth && current != constants.RootParentID; depth++ {
		f, err := s.fileRepo.FindByIDAndUser(current, userID)
		if err != nil {
			if repository.IsNotFound(err) {
				return nil, errs.ErrFileNotFound
			}
			return nil, err
		}
		if f.Status != constants.FileStatusNormal {
			return nil, errs.ErrFileNotFound
		}
		chain = append(chain, dto.BreadcrumbItem{ID: f.ID, Name: f.Name})
		current = f.ParentID
	}
	for i := len(chain) - 1; i >= 0; i-- {
		items = append(items, chain[i])
	}
	return items, nil
}

// Recent 最近访问。
func (s *FileService) Recent(userID uint64, limit int) ([]dto.FileItem, error) {
	if limit <= 0 {
		limit = constants.RecentDefaultLimit
	}
	if limit > constants.RecentMaxLimit {
		limit = constants.RecentMaxLimit
	}
	files, err := s.fileRepo.Recent(userID, limit)
	if err != nil {
		return nil, err
	}
	return fileItems(files), nil
}

// Favorites 收藏列表。
func (s *FileService) Favorites(userID uint64) ([]dto.FileItem, error) {
	files, err := s.fileRepo.Favorites(userID)
	if err != nil {
		return nil, err
	}
	return fileItems(files), nil
}

// Stats 空间统计（按类型聚合）。
func (s *FileService) Stats(userID uint64) (*dto.FileStatsResp, error) {
	return s.userSvc.Overview(userID)
}

// Mkdir 新建目录。
func (s *FileService) Mkdir(userID, parentID uint64, name string) (*dto.FileItem, error) {
	if _, err := s.getDir(userID, parentID); err != nil {
		return nil, err
	}
	clean, err := fileutil.SanitizeName(name)
	if err != nil {
		return nil, errs.ErrInvalidName.WithMessage("%s", err.Error())
	}
	exists, err := s.fileRepo.ExistsName(userID, parentID, clean, 0)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errs.ErrNameConflict
	}
	dir := &model.File{
		UserID:   userID,
		ParentID: parentID,
		Name:     clean,
		IsDir:    constants.IsDir,
		Status:   constants.FileStatusNormal,
	}
	if err := s.fileRepo.Create(dir); err != nil {
		return nil, err
	}
	item := fileItem(dir)
	return &item, nil
}

// Upload 单文件上传（支持 relative_path 逐级创建目录）。
func (s *FileService) Upload(userID, parentID uint64, relativePath string, fh *multipart.FileHeader) (*dto.UploadResp, error) {
	if _, err := s.getDir(userID, parentID); err != nil {
		return nil, err
	}
	rel := relativePath
	if strings.TrimSpace(rel) == "" {
		rel = fh.Filename
	}
	cleanRel, err := fileutil.CleanRelativePath(rel)
	if err != nil {
		return nil, errs.ErrInvalidName.WithMessage("%s", err.Error())
	}
	if cleanRel == "" {
		return nil, errs.ErrInvalidName
	}
	dirPart := path.Dir(cleanRel)
	if dirPart == "." {
		dirPart = ""
	}
	dirID, err := s.ensureDir(userID, parentID, dirPart, map[string]uint64{"": parentID})
	if err != nil {
		return nil, err
	}
	fileName, err := fileutil.SanitizeName(path.Base(cleanRel))
	if err != nil {
		return nil, errs.ErrInvalidName.WithMessage("%s", err.Error())
	}
	file, err := s.saveUpload(userID, dirID, fileName, fh)
	if err != nil {
		return nil, err
	}
	return &dto.UploadResp{ID: file.ID, Name: file.Name, ParentID: file.ParentID, Size: file.Size}, nil
}

// UploadBatch 批量 / 文件夹上传（普通通道，部分成功可接受）。
func (s *FileService) UploadBatch(userID, parentID uint64, files []*multipart.FileHeader, manifest []dto.ManifestItem) (*dto.BatchUploadResp, error) {
	if len(files) == 0 {
		return nil, errs.ErrBadParam.WithMessage("未收到上传文件")
	}
	if len(files) > s.cfg.Storage.Upload.MaxBatchFiles {
		return nil, errs.ErrUploadLimit.WithMessage("单批文件数超过上限 %d", s.cfg.Storage.Upload.MaxBatchFiles)
	}
	var batchSize int64
	for _, fh := range files {
		batchSize += fh.Size
	}
	if batchSize > s.cfg.Storage.Upload.MaxBatchSize {
		return nil, errs.ErrUploadLimit.WithMessage("单批总大小超过上限")
	}
	if _, err := s.getDir(userID, parentID); err != nil {
		return nil, err
	}

	relByIndex := make(map[int]string, len(manifest))
	for _, m := range manifest {
		relByIndex[m.Index] = m.RelativePath
	}
	dirCache := map[string]uint64{"": parentID}
	result := &dto.BatchUploadResp{
		Succeeded: make([]dto.BatchSucceeded, 0, len(files)),
		Failed:    make([]dto.BatchFailed, 0),
		Total:     len(files),
	}
	for i, fh := range files {
		rel := relByIndex[i]
		if strings.TrimSpace(rel) == "" {
			rel = fh.Filename
		}
		file, err := s.uploadOne(userID, parentID, rel, fh, dirCache)
		if err != nil {
			code, msg := errs.From(err)
			result.Failed = append(result.Failed, dto.BatchFailed{Name: fh.Filename, Reason: msg, Code: code})
			continue
		}
		result.Succeeded = append(result.Succeeded, dto.BatchSucceeded{
			Name: file.Name, ID: file.ID, ParentID: file.ParentID, Size: file.Size,
		})
	}
	result.SuccessCount = len(result.Succeeded)
	result.FailCount = len(result.Failed)
	return result, nil
}

// uploadOne 处理批量上传中的单个文件。
func (s *FileService) uploadOne(userID, parentID uint64, rel string, fh *multipart.FileHeader, dirCache map[string]uint64) (*model.File, error) {
	cleanRel, err := fileutil.CleanRelativePath(rel)
	if err != nil || cleanRel == "" {
		return nil, errs.ErrInvalidName
	}
	fileName, err := fileutil.SanitizeName(path.Base(cleanRel))
	if err != nil {
		return nil, errs.ErrInvalidName.WithMessage("%s", err.Error())
	}
	dirPart := path.Dir(cleanRel)
	if dirPart == "." {
		dirPart = ""
	}
	dirID, err := s.ensureDir(userID, parentID, dirPart, dirCache)
	if err != nil {
		return nil, err
	}
	return s.saveUpload(userID, dirID, fileName, fh)
}

// saveUpload 落盘 + MD5 + 统一入库。
func (s *FileService) saveUpload(userID, parentID uint64, name string, fh *multipart.FileHeader) (*model.File, error) {
	ext := fileutil.Ext(name)
	if !fileutil.IsAllowedExt(ext, s.cfg.Storage.Upload.AllowedExt) {
		return nil, errs.ErrExtNotAllowed.WithMessage("扩展名 %s 不允许上传", ext)
	}
	if fh.Size > s.cfg.Storage.Upload.MaxFileSize {
		return nil, errs.ErrFileTooLarge
	}
	if err := s.userSvc.CheckQuota(userID, fh.Size); err != nil {
		return nil, err
	}
	src, err := fh.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()
	tmpPath, size, err := s.storage.SaveTemp(src)
	if err != nil {
		return nil, err
	}
	md5sum, written, err := hash.MD5File(tmpPath)
	if err != nil {
		_ = s.storage.DeleteTemp(tmpPath)
		return nil, err
	}
	if written != size {
		_ = s.storage.DeleteTemp(tmpPath)
		return nil, errs.ErrInternal.WithMessage("上传数据写入不完整")
	}
	return s.blobSvc.PersistTemp(userID, parentID, name, tmpPath, md5sum, size)
}

// ensureDir 逐级创建目录，cache 以相对路径为键复用结果。
func (s *FileService) ensureDir(userID, parentID uint64, relDir string, cache map[string]uint64) (uint64, error) {
	relDir = strings.Trim(relDir, "/")
	if relDir == "" {
		return parentID, nil
	}
	if id, ok := cache[relDir]; ok {
		return id, nil
	}
	current := parentID
	currentPath := ""
	for _, seg := range strings.Split(relDir, "/") {
		name, err := fileutil.SanitizeName(seg)
		if err != nil {
			return 0, errs.ErrInvalidName.WithMessage("%s", err.Error())
		}
		if currentPath == "" {
			currentPath = name
		} else {
			currentPath = currentPath + "/" + name
		}
		if id, ok := cache[currentPath]; ok {
			current = id
			continue
		}
		existing, err := s.fileRepo.FindByName(userID, current, name)
		switch {
		case err == nil:
			if existing.IsDir != constants.IsDir {
				return 0, errs.ErrNameConflict.WithMessage("同名文件已存在: %s", name)
			}
			current = existing.ID
		case repository.IsNotFound(err):
			dir := &model.File{
				UserID:   userID,
				ParentID: current,
				Name:     name,
				IsDir:    constants.IsDir,
				Status:   constants.FileStatusNormal,
			}
			if err := s.fileRepo.Create(dir); err != nil {
				return 0, err
			}
			current = dir.ID
		default:
			return 0, err
		}
		cache[currentPath] = current
	}
	return current, nil
}

// Check 秒传预检。
func (s *FileService) Check(userID uint64, hashMD5 string, size int64) (*dto.CheckResp, error) {
	md5sum := normalizeMD5(hashMD5)
	if md5sum == "" {
		return nil, errs.ErrBadParam.WithMessage("hash_md5 非法")
	}
	blob, err := s.blobSvc.FindByHash(md5sum)
	if err != nil {
		return nil, err
	}
	if blob == nil {
		return &dto.CheckResp{Hit: false}, nil
	}
	return &dto.CheckResp{Hit: true, BlobID: blob.ID}, nil
}

// Detail 文件详情（顺带刷新最近访问时间）。
func (s *FileService) Detail(userID, fileID uint64) (*dto.FileItem, error) {
	f, err := s.getFile(userID, fileID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if f.LastAccessAt == nil || now.Sub(*f.LastAccessAt) > constants.FileAccessTouchInterval {
		if err := s.fileRepo.TouchAccess(f.ID, userID, now); err != nil {
			s.log.Warn("刷新访问时间失败", zap.Uint64("file_id", f.ID), zap.Error(err))
		} else {
			f.LastAccessAt = &now
		}
	}
	item := fileItem(f)
	return &item, nil
}

// Rename 重命名文件或目录。
func (s *FileService) Rename(userID, fileID uint64, newName string) (*dto.FileItem, error) {
	f, err := s.getFile(userID, fileID)
	if err != nil {
		return nil, err
	}
	clean, err := fileutil.SanitizeName(newName)
	if err != nil {
		return nil, errs.ErrInvalidName.WithMessage("%s", err.Error())
	}
	if clean == f.Name {
		return nil, errs.ErrSameName
	}
	exists, err := s.fileRepo.ExistsName(userID, f.ParentID, clean, f.ID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errs.ErrNameConflict
	}
	fields := map[string]any{"name": clean}
	if f.IsDir == constants.IsFile {
		ext := fileutil.Ext(clean)
		fields["ext"] = ext
		fields["mime_type"] = fileutil.MimeByExt(ext)
	}
	if _, err := s.fileRepo.UpdateFields(f.ID, userID, fields); err != nil {
		return nil, err
	}
	if refreshed, err := s.fileRepo.FindByIDAndUser(f.ID, userID); err == nil {
		f = refreshed
	}
	item := fileItem(f)
	return &item, nil
}

// Move 移动文件 / 目录。
func (s *FileService) Move(userID, fileID, targetParentID uint64) error {
	f, err := s.getFile(userID, fileID)
	if err != nil {
		return err
	}
	if f.ParentID == targetParentID {
		return nil
	}
	if _, err := s.getDir(userID, targetParentID); err != nil {
		return err
	}
	if f.IsDir == constants.IsDir && targetParentID != constants.RootParentID {
		if err := s.assertNotDescendant(userID, f.ID, targetParentID); err != nil {
			return err
		}
	}
	exists, err := s.fileRepo.ExistsName(userID, targetParentID, f.Name, f.ID)
	if err != nil {
		return err
	}
	if exists {
		return errs.ErrNameConflict
	}
	_, err = s.fileRepo.UpdateFields(f.ID, userID, map[string]any{"parent_id": targetParentID})
	return err
}

// assertNotDescendant 校验 nodeID 不是 targetID 的祖先（防止目录成环）。
func (s *FileService) assertNotDescendant(userID, nodeID, targetID uint64) error {
	current := targetID
	for depth := 0; depth < constants.MaxDirDepth && current != constants.RootParentID; depth++ {
		if current == nodeID {
			return errs.ErrInvalidMove
		}
		parent, err := s.fileRepo.FindByIDAndUser(current, userID)
		if err != nil {
			if repository.IsNotFound(err) {
				return errs.ErrInvalidTarget
			}
			return err
		}
		current = parent.ParentID
	}
	return nil
}

// SetFavorite 收藏 / 取消收藏。
func (s *FileService) SetFavorite(userID, fileID uint64, favorite int8) error {
	f, err := s.getFile(userID, fileID)
	if err != nil {
		return err
	}
	if favorite != 0 && favorite != 1 {
		return errs.ErrBadParam
	}
	_, err = s.fileRepo.UpdateFields(f.ID, userID, map[string]any{"is_favorite": favorite})
	return err
}

// Copy 复制文件或目录（目录递归，物理文件引用计数 +1）。
func (s *FileService) Copy(userID, fileID, targetParentID uint64, newName string) (*dto.CopyResp, error) {
	src, err := s.getFile(userID, fileID)
	if err != nil {
		return nil, err
	}
	if _, err := s.getDir(userID, targetParentID); err != nil {
		return nil, err
	}
	if src.IsDir == constants.IsDir && targetParentID != constants.RootParentID {
		if err := s.assertNotDescendant(userID, src.ID, targetParentID); err != nil {
			return nil, err
		}
	}
	subtree, err := collectSubtree(s.fileRepo, userID, src)
	if err != nil {
		return nil, err
	}
	totalSize := subtreeSize(subtree)
	if err := s.userSvc.CheckQuota(userID, totalSize); err != nil {
		return nil, err
	}

	name := src.Name
	if strings.TrimSpace(newName) != "" {
		clean, err := fileutil.SanitizeName(newName)
		if err != nil {
			return nil, errs.ErrInvalidName.WithMessage("%s", err.Error())
		}
		exists, err := s.fileRepo.ExistsName(userID, targetParentID, clean, 0)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, errs.ErrNameConflict
		}
		name = clean
	} else if src.ParentID == targetParentID {
		// 同目录复制自动追加序号，避免与源同名。
		name, err = uniqueName(s.fileRepo, userID, targetParentID, src.Name)
		if err != nil {
			return nil, err
		}
	} else {
		exists, err := s.fileRepo.ExistsName(userID, targetParentID, src.Name, 0)
		if err != nil {
			return nil, err
		}
		if exists {
			name, err = uniqueName(s.fileRepo, userID, targetParentID, src.Name)
			if err != nil {
				return nil, err
			}
		}
	}

	refCounts := make(map[uint64]int64)
	var rootID uint64
	err = s.fileRepo.DB().Transaction(func(tx *gorm.DB) error {
		fileRepo := s.fileRepo.WithTx(tx)
		blobRepo := s.blobRepo.WithTx(tx)
		rootCopy := &model.File{
			UserID:     userID,
			ParentID:   targetParentID,
			Name:       name,
			IsDir:      src.IsDir,
			Ext:        src.Ext,
			Size:       src.Size,
			MimeType:   src.MimeType,
			BlobID:     src.BlobID,
			IsFavorite: 0,
			Status:     constants.FileStatusNormal,
		}
		if err := fileRepo.Create(rootCopy); err != nil {
			return err
		}
		rootID = rootCopy.ID
		if src.IsDir == constants.IsFile && src.BlobID > 0 {
			refCounts[src.BlobID]++
		}
		idMap := map[uint64]uint64{src.ID: rootCopy.ID}
		for i := range subtree {
			node := subtree[i]
			if node.ID == src.ID {
				continue
			}
			newParent, ok := idMap[node.ParentID]
			if !ok {
				continue
			}
			copyNode := &model.File{
				UserID:   userID,
				ParentID: newParent,
				Name:     node.Name,
				IsDir:    node.IsDir,
				Ext:      node.Ext,
				Size:     node.Size,
				MimeType: node.MimeType,
				BlobID:   node.BlobID,
				Status:   constants.FileStatusNormal,
			}
			if err := fileRepo.Create(copyNode); err != nil {
				return err
			}
			idMap[node.ID] = copyNode.ID
			if node.IsDir == constants.IsFile && node.BlobID > 0 {
				refCounts[node.BlobID]++
			}
		}
		for blobID, count := range refCounts {
			if err := blobRepo.IncRefN(blobID, count); err != nil {
				return err
			}
		}
		return s.userRepo.WithTx(tx).AddStorageUsed(userID, totalSize)
	})
	if err != nil {
		return nil, err
	}
	return &dto.CopyResp{NewID: rootID}, nil
}

// Delete 删除到回收站（保留物理文件引用）。
func (s *FileService) Delete(userID, fileID uint64) error {
	f, err := s.getFile(userID, fileID)
	if err != nil {
		return err
	}
	subtree, err := collectSubtree(s.fileRepo, userID, f)
	if err != nil {
		return err
	}
	now := time.Now()
	return s.fileRepo.SetStatusRecursive(subtreeIDs(subtree), constants.FileStatusRecycle, &now)
}

// RecycleList 回收站列表（仅展示顶层条目）。
func (s *FileService) RecycleList(userID uint64) ([]dto.FileItem, error) {
	files, err := s.fileRepo.ListByStatus(userID, constants.FileStatusRecycle, "updated_at", "desc")
	if err != nil {
		return nil, err
	}
	deleted := make(map[uint64]bool, len(files))
	for i := range files {
		deleted[files[i].ID] = true
	}
	top := make([]model.File, 0, len(files))
	for i := range files {
		if files[i].ParentID != constants.RootParentID && deleted[files[i].ParentID] {
			continue
		}
		top = append(top, files[i])
	}
	return fileItems(top), nil
}

// Restore 从回收站还原。
func (s *FileService) Restore(userID, fileID uint64) error {
	f, err := s.fileRepo.FindByIDAndUser(fileID, userID)
	if err != nil {
		if repository.IsNotFound(err) {
			return errs.ErrFileNotFound
		}
		return err
	}
	if f.Status != constants.FileStatusRecycle {
		return errs.ErrBadParam.WithMessage("该条目不在回收站中")
	}
	subtree, err := collectSubtree(s.fileRepo, userID, f)
	if err != nil {
		return err
	}
	ids := make([]uint64, 0, len(subtree))
	for i := range subtree {
		if subtree[i].Status == constants.FileStatusRecycle {
			ids = append(ids, subtree[i].ID)
		}
	}
	if err := s.fileRepo.SetStatusRecursive(ids, constants.FileStatusNormal, nil); err != nil {
		return err
	}

	targetParent := f.ParentID
	if targetParent != constants.RootParentID {
		parent, err := s.fileRepo.FindByIDAndUser(targetParent, userID)
		if err != nil || parent.Status != constants.FileStatusNormal {
			targetParent = constants.RootParentID
		}
	}
	name := f.Name
	if exists, err := s.fileRepo.ExistsName(userID, targetParent, name, f.ID); err != nil {
		return err
	} else if exists {
		name, err = uniqueName(s.fileRepo, userID, targetParent, f.Name)
		if err != nil {
			return err
		}
	}
	if targetParent != f.ParentID || name != f.Name {
		if _, err := s.fileRepo.UpdateFields(f.ID, userID, map[string]any{
			"parent_id": targetParent,
			"name":      name,
		}); err != nil {
			return err
		}
	}
	return nil
}

// ForceDelete 彻底删除（引用计数 -1，回收配额）。
func (s *FileService) ForceDelete(userID, fileID uint64) error {
	f, err := s.fileRepo.FindByIDAndUser(fileID, userID)
	if err != nil {
		if repository.IsNotFound(err) {
			return errs.ErrFileNotFound
		}
		return err
	}
	subtree, err := collectSubtree(s.fileRepo, userID, f)
	if err != nil {
		return err
	}
	return s.purge(userID, subtree)
}

// EmptyRecycle 清空回收站。
func (s *FileService) EmptyRecycle(userID uint64) error {
	files, err := s.fileRepo.ListByStatus(userID, constants.FileStatusRecycle, "updated_at", "desc")
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return nil
	}
	return s.purge(userID, files)
}

// purge 物理删除记录并同步引用计数 / 配额。
func (s *FileService) purge(userID uint64, files []model.File) error {
	ids := subtreeIDs(files)
	totalSize := subtreeSize(files)
	return s.fileRepo.DB().Transaction(func(tx *gorm.DB) error {
		fileRepo := s.fileRepo.WithTx(tx)
		blobRepo := s.blobRepo.WithTx(tx)
		refs, err := fileRepo.BlobRefs(ids)
		if err != nil {
			return err
		}
		for _, ref := range refs {
			if err := blobRepo.DecRef(ref.BlobID, ref.Count); err != nil {
				return err
			}
		}
		if err := fileRepo.DeleteByIDs(ids); err != nil {
			return err
		}
		return s.userRepo.WithTx(tx).SubStorageUsed(userID, totalSize)
	})
}

// DownloadTarget 下载目标信息。
type DownloadTarget struct {
	File *model.File
	Blob *model.Blob
}

// Download 解析单文件 / 目录下载目标（目录由调用方转为 zip 流）。
func (s *FileService) Download(userID, fileID uint64) (*DownloadTarget, error) {
	f, err := s.getFile(userID, fileID)
	if err != nil {
		return nil, err
	}
	if f.IsDir == constants.IsDir {
		return &DownloadTarget{File: f}, nil
	}
	blob, err := s.blobSvc.FindByID(f.BlobID)
	if err != nil {
		return nil, err
	}
	return &DownloadTarget{File: f, Blob: blob}, nil
}

// Raw 预览 / 缩略图所需的目标信息。
func (s *FileService) Raw(userID, fileID uint64) (*DownloadTarget, error) {
	return s.Download(userID, fileID)
}

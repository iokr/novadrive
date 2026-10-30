package service

import (
	"time"

	"github.com/iokr/novadrive/internal/constants"
	"github.com/iokr/novadrive/internal/model"
	"github.com/iokr/novadrive/pkg/errs"
	"go.uber.org/zap"
)

// ZipEntry zip 流中的单个条目。
type ZipEntry struct {
	// Name zip 内相对路径；目录必须以 "/" 结尾。
	Name    string
	Dir     bool
	Blob    *model.Blob
	ModTime time.Time
}

// CollectDirZip 收集单目录下载的 zip 条目（路径相对于该目录）。
func (s *FileService) CollectDirZip(userID, dirID uint64) ([]ZipEntry, string, error) {
	dir, err := s.getFile(userID, dirID)
	if err != nil {
		return nil, "", err
	}
	if dir.IsDir != constants.IsDir {
		return nil, "", errs.ErrBadParam.WithMessage("该条目不是目录")
	}
	entries := make([]ZipEntry, 0, 32)
	cache := make(map[uint64]*model.Blob)
	if err := s.walkZip(userID, dir, "", &entries, cache, false); err != nil {
		return nil, "", err
	}
	return entries, dir.Name + ".zip", nil
}

// CollectBatchZip 收集批量下载的 zip 条目（顶层以各自原名称建目录）。
func (s *FileService) CollectBatchZip(userID uint64, ids []uint64) ([]ZipEntry, string, error) {
	if len(ids) == 0 {
		return nil, "", errs.ErrBadParam.WithMessage("未选择下载项")
	}
	entries := make([]ZipEntry, 0, 64)
	cache := make(map[uint64]*model.Blob)
	filename := "novadrive-files.zip"
	for _, id := range ids {
		node, err := s.getFile(userID, id)
		if err != nil {
			return nil, "", err
		}
		if node.IsDir == constants.IsDir {
			if err := s.walkZip(userID, node, node.Name+"/", &entries, cache, true); err != nil {
				return nil, "", err
			}
			continue
		}
		blob, err := s.blobOf(node, cache)
		if err != nil {
			s.log.Warn("下载条目缺少物理文件", zap.Uint64("file_id", node.ID), zap.Error(err))
			continue
		}
		if err := s.appendEntry(&entries, ZipEntry{Name: node.Name, Blob: blob, ModTime: node.UpdatedAt}); err != nil {
			return nil, "", err
		}
		if len(ids) == 1 {
			filename = node.Name + ".zip"
		}
	}
	return entries, filename, nil
}

// walkZip 递归收集子树条目。
func (s *FileService) walkZip(
	userID uint64,
	node *model.File,
	base string,
	entries *[]ZipEntry,
	cache map[uint64]*model.Blob,
	addRoot bool,
) error {
	if node.IsDir != constants.IsDir {
		return nil
	}
	if addRoot {
		if err := s.appendEntry(entries, ZipEntry{Name: base, Dir: true, ModTime: node.UpdatedAt}); err != nil {
			return err
		}
	}
	children, err := s.fileRepo.ListByParent(userID, node.ID, "name", "asc")
	if err != nil {
		return err
	}
	if len(children) == 0 && !addRoot {
		// 空目录也要生成合法 zip：写入一个占位目录项。
		return s.appendEntry(entries, ZipEntry{Name: node.Name + "/", Dir: true, ModTime: node.UpdatedAt})
	}
	for i := range children {
		child := children[i]
		childBase := base + child.Name
		if child.IsDir == constants.IsDir {
			if err := s.walkZip(userID, &child, childBase+"/", entries, cache, true); err != nil {
				return err
			}
			continue
		}
		blob, err := s.blobOf(&child, cache)
		if err != nil {
			s.log.Warn("下载条目缺少物理文件", zap.Uint64("file_id", child.ID), zap.Error(err))
			continue
		}
		if err := s.appendEntry(entries, ZipEntry{Name: childBase, Blob: blob, ModTime: child.UpdatedAt}); err != nil {
			return err
		}
	}
	return nil
}

// appendEntry 追加条目并校验上限。
func (s *FileService) appendEntry(entries *[]ZipEntry, entry ZipEntry) error {
	if len(*entries) >= s.cfg.Download.ZipMaxEntries {
		return errs.ErrZipEntries
	}
	*entries = append(*entries, entry)
	return nil
}

// blobOf 查询并缓存物理文件记录。
func (s *FileService) blobOf(f *model.File, cache map[uint64]*model.Blob) (*model.Blob, error) {
	if f.BlobID == 0 {
		return nil, errs.ErrFileNotFound.WithMessage("文件缺少物理数据")
	}
	if blob, ok := cache[f.BlobID]; ok {
		return blob, nil
	}
	blob, err := s.blobSvc.FindByID(f.BlobID)
	if err != nil {
		return nil, err
	}
	cache[f.BlobID] = blob
	return blob, nil
}

// ThumbTarget 缩略图目标（目录返回 3001，未生成缩略图时由调用方回落原图）。
func (s *FileService) ThumbTarget(userID, fileID uint64) (*DownloadTarget, error) {
	f, err := s.getFile(userID, fileID)
	if err != nil {
		return nil, err
	}
	if f.IsDir == constants.IsDir {
		return nil, errs.ErrBadParam.WithMessage("目录不支持缩略图")
	}
	blob, err := s.blobSvc.FindByID(f.BlobID)
	if err != nil {
		return nil, err
	}
	return &DownloadTarget{File: f, Blob: blob}, nil
}

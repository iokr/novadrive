// Package service 实现业务逻辑。
package service

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/iokr/novadrive/internal/constants"
	"github.com/iokr/novadrive/internal/dto"
	"github.com/iokr/novadrive/internal/model"
	"github.com/iokr/novadrive/internal/repository"
	"github.com/iokr/novadrive/pkg/fileutil"
)

// fileItem 把模型转换为列表项 DTO。
func fileItem(f *model.File) dto.FileItem {
	return dto.FileItem{
		ID:           f.ID,
		ParentID:     f.ParentID,
		Name:         f.Name,
		IsDir:        f.IsDir,
		Ext:          f.Ext,
		Size:         f.Size,
		MimeType:     f.MimeType,
		IsFavorite:   f.IsFavorite,
		Status:       f.Status,
		HasThumb:     f.IsDir == constants.IsFile && fileutil.IsImageExt(f.Ext),
		LastAccessAt: f.LastAccessAt,
		DeletedAt:    f.RecycleAt,
		CreatedAt:    f.CreatedAt,
		UpdatedAt:    f.UpdatedAt,
	}
}

// fileItems 批量转换。
func fileItems(files []model.File) []dto.FileItem {
	items := make([]dto.FileItem, 0, len(files))
	for i := range files {
		items = append(items, fileItem(&files[i]))
	}
	return items
}

// uniqueName 在指定目录下生成不冲突的名称：name -> name (1) -> name (2) ...
func uniqueName(repo *repository.FileRepo, userID, parentID uint64, name string) (string, error) {
	exists, err := repo.ExistsName(userID, parentID, name, 0)
	if err != nil {
		return "", err
	}
	if !exists {
		return name, nil
	}
	base, ext := fileutil.SplitName(name)
	for i := 1; i <= 1000; i++ {
		suffix := fmt.Sprintf(" (%d)", i)
		stem := truncateName(base, len(suffix)) + suffix
		candidate := stem
		if ext != "" {
			candidate = stem + "." + ext
		}
		exists, err := repo.ExistsName(userID, parentID, candidate, 0)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("无法为 %s 生成唯一名称", name)
}

// truncateName 截断名称，保证追加 suffix 后不超过上限。
func truncateName(name string, suffixLen int) string {
	limit := constants.MaxNameLength - suffixLen
	if utf8.RuneCountInString(name) <= limit {
		return name
	}
	runes := []rune(name)
	return string(runes[:limit])
}

// collectSubtree 收集 root 及其全部后代的记录（按层级 BFS）。
func collectSubtree(repo *repository.FileRepo, userID uint64, root *model.File) ([]model.File, error) {
	all := []model.File{*root}
	if root.IsDir != constants.IsDir {
		return all, nil
	}
	visited := map[uint64]bool{root.ID: true}
	frontier := []uint64{root.ID}
	for depth := 0; depth < constants.MaxDirDepth && len(frontier) > 0; depth++ {
		children, err := repo.ListByParents(userID, frontier)
		if err != nil {
			return nil, err
		}
		next := make([]uint64, 0, len(children))
		for _, c := range children {
			if visited[c.ID] {
				continue
			}
			visited[c.ID] = true
			all = append(all, c)
			if c.IsDir == constants.IsDir {
				next = append(next, c.ID)
			}
		}
		frontier = next
	}
	return all, nil
}

// subtreeIDs 收集子树 ID。
func subtreeIDs(files []model.File) []uint64 {
	ids := make([]uint64, 0, len(files))
	for i := range files {
		ids = append(ids, files[i].ID)
	}
	return ids
}

// subtreeSize 汇总子树内文件大小。
func subtreeSize(files []model.File) int64 {
	var total int64
	for i := range files {
		if files[i].IsDir == constants.IsFile {
			total += files[i].Size
		}
	}
	return total
}

// normalizeMD5 归一化 MD5（小写并校验格式），非法值返回空串。
func normalizeMD5(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if len(s) != 32 {
		return ""
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return ""
		}
	}
	return s
}

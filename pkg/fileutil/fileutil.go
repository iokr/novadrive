// Package fileutil 提供文件名、扩展名、MIME 与安全检查工具。
package fileutil

import (
	"errors"
	"path"
	"strings"
	"unicode/utf8"

	"github.com/iokr/novadrive/internal/constants"
)

// mimeByExt 扩展名到 MIME 的映射。
var mimeByExt = map[string]string{
	// 图片
	"jpg": "image/jpeg", "jpeg": "image/jpeg", "png": "image/png",
	"gif": "image/gif", "webp": "image/webp", "bmp": "image/bmp",
	"svg": "image/svg+xml", "ico": "image/x-icon", "tiff": "image/tiff",
	// 视频
	"mp4": "video/mp4", "webm": "video/webm", "mov": "video/quicktime",
	"mkv": "video/x-matroska", "avi": "video/x-msvideo", "m4v": "video/x-m4v",
	// 音频
	"mp3": "audio/mpeg", "wav": "audio/wav", "flac": "audio/flac",
	"ogg": "audio/ogg", "m4a": "audio/mp4", "aac": "audio/aac",
	// 文档
	"pdf": "application/pdf", "txt": "text/plain", "md": "text/markdown",
	"json": "application/json", "log": "text/plain", "csv": "text/csv",
	"xml": "application/xml", "html": "text/html", "css": "text/css",
	"js": "text/javascript", "yml": "text/yaml", "yaml": "text/yaml",
	"doc":  "application/msword",
	"docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	"xls":  "application/vnd.ms-excel",
	"xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	"ppt":  "application/vnd.ms-powerpoint",
	"pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	// 压缩包
	"zip": "application/zip",
	"rar": "application/vnd.rar",
	"7z":  "application/x-7z-compressed",
	"gz":  "application/gzip",
	"tar": "application/x-tar",
	// 其他
	"exe": "application/octet-stream", "bin": "application/octet-stream",
}

// categoryByExt 扩展名到大类的映射，用于空间统计。
var categoryByExt = map[string]string{
	"jpg": "image", "jpeg": "image", "png": "image", "gif": "image",
	"webp": "image", "bmp": "image", "svg": "image", "ico": "image",
	"mp4": "video", "webm": "video", "mov": "video", "mkv": "video",
	"avi": "video", "m4v": "video",
	"mp3": "audio", "wav": "audio", "flac": "audio", "ogg": "audio",
	"m4a": "audio", "aac": "audio",
	"pdf": "pdf",
	"txt": "text", "md": "text", "json": "text", "log": "text",
	"csv": "text", "xml": "text", "yml": "text", "yaml": "text",
	"html": "text", "css": "text", "js": "text",
}

// Ext 返回小写扩展名（不含点）。
func Ext(name string) string {
	ext := path.Ext(name)
	if ext == "" {
		return ""
	}
	return strings.ToLower(strings.TrimPrefix(ext, "."))
}

// SplitName 拆分主名与扩展名（扩展名不含点）。
func SplitName(name string) (string, string) {
	ext := Ext(name)
	if ext == "" {
		return name, ""
	}
	return strings.TrimSuffix(name, "."+ext), ext
}

// JoinExt 拼接主名与扩展名。
func JoinExt(base, ext string) string {
	if ext == "" {
		return base
	}
	return base + "." + ext
}

// MimeByExt 根据扩展名推断 MIME。
func MimeByExt(ext string) string {
	if v, ok := mimeByExt[strings.ToLower(ext)]; ok {
		return v
	}
	return "application/octet-stream"
}

// Category 返回统计分类：image/video/audio/pdf/text/other。
func Category(ext string) string {
	if v, ok := categoryByExt[strings.ToLower(ext)]; ok {
		return v
	}
	return "other"
}

// IsImageExt 是否为可生成缩略图的图片扩展名。
func IsImageExt(ext string) bool {
	switch strings.ToLower(ext) {
	case "jpg", "jpeg", "png", "gif", "webp", "bmp":
		return true
	}
	return false
}

// IsImageMime 是否图片 MIME。
func IsImageMime(mime string) bool {
	return strings.HasPrefix(strings.ToLower(mime), "image/") && mime != "image/svg+xml"
}

// IsTextExt 是否为文本预览扩展名。
func IsTextExt(ext string) bool {
	if strings.HasPrefix(MimeByExt(ext), "text/") {
		return true
	}
	switch strings.ToLower(ext) {
	case "json", "log", "md", "yml", "yaml", "csv":
		return true
	}
	return false
}

// SanitizeName 校验并清洗文件名 / 目录名，拒绝路径穿越与非法字符。
func SanitizeName(name string) (string, error) {
	name = strings.TrimSpace(name)
	name = strings.Trim(name, " ")
	if name == "" || name == "." || name == ".." {
		return "", errors.New("名称不能为空")
	}
	if strings.ContainsAny(name, "/\\") {
		return "", errors.New("名称不能包含路径分隔符")
	}
	if strings.ContainsRune(name, 0) || strings.ContainsRune(name, '\r') || strings.ContainsRune(name, '\n') {
		return "", errors.New("名称包含非法字符")
	}
	if !utf8.ValidString(name) {
		return "", errors.New("名称包含非法字符")
	}
	if utf8.RuneCountInString(name) > constants.MaxNameLength {
		return "", errors.New("名称过长")
	}
	return name, nil
}

// CleanRelativePath 规范化客户端上传的相对路径，拒绝绝对路径与穿越片段。
func CleanRelativePath(p string) (string, error) {
	p = strings.ReplaceAll(strings.TrimSpace(p), "\\", "/")
	if p == "" {
		return "", nil
	}
	if strings.HasPrefix(p, "/") {
		return "", errors.New("相对路径不能以 / 开头")
	}
	cleaned := path.Clean(p)
	if cleaned == "." {
		return "", nil
	}
	for _, seg := range strings.Split(cleaned, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return "", errors.New("相对路径包含非法片段")
		}
		if utf8.RuneCountInString(seg) > constants.MaxNameLength {
			return "", errors.New("路径片段过长")
		}
	}
	return cleaned, nil
}

// IsAllowedExt 判断扩展名是否在上传白名单内（空扩展名允许）。
func IsAllowedExt(ext string, allowed []string) bool {
	if ext == "" {
		return true
	}
	for _, a := range allowed {
		if strings.EqualFold(strings.TrimPrefix(strings.TrimSpace(a), "."), ext) {
			return true
		}
	}
	return false
}

// FallbackName 生成 Content-Disposition 的 ASCII 兜底文件名。
func FallbackName(name string) string {
	if name == "" {
		return "download"
	}
	var b strings.Builder
	for _, r := range name {
		if r < 32 || r > 126 || r == '"' || r == '\\' {
			b.WriteByte('_')
			continue
		}
		b.WriteRune(r)
	}
	out := b.String()
	if out == "" {
		return "download"
	}
	return out
}

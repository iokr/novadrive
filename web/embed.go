// Package web 嵌入前端构建产物，实现单二进制部署。
package web

import "embed"

// Dist 前端构建产物目录（web/dist）。
//
// 构建前端后产物会随二进制一起发布；目录中缺少 index.html 时，
// SPA 兜底逻辑会返回提示页面。
//
//go:embed all:dist
var Dist embed.FS

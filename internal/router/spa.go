package router

import (
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/iokr/novadrive/internal/constants"
	"github.com/iokr/novadrive/pkg/resp"
	"github.com/iokr/novadrive/web"
)

// placeholderPage 前端产物缺失时的兜底页面。
const placeholderPage = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8" />
<meta name="viewport" content="width=device-width, initial-scale=1" />
<title>NovaDrive · 星云盘</title>
<style>
  body{margin:0;min-height:100vh;display:flex;align-items:center;justify-content:center;
       background:radial-gradient(circle at 30% 20%,#131A2E 0%,#0B1020 70%);
       color:#E5E7EB;font-family:system-ui,-apple-system,"Segoe UI",sans-serif}
  .card{padding:40px 48px;border-radius:16px;background:rgba(255,255,255,.06);
        border:1px solid rgba(255,255,255,.12);backdrop-filter:blur(16px);max-width:560px}
  h1{margin:0 0 12px;font-size:24px;background:linear-gradient(135deg,#6366F1,#22D3EE);
     -webkit-background-clip:text;background-clip:text;color:transparent}
  p{margin:8px 0;line-height:1.7;color:#9CA3AF;font-size:14px}
  code{background:rgba(99,102,241,.18);padding:2px 6px;border-radius:6px;color:#A5B4FC}
</style>
</head>
<body>
  <div class="card">
    <h1>NovaDrive · 星云盘</h1>
    <p>后端服务已启动，但未检测到前端构建产物。</p>
    <p>请在前端目录执行 <code>npm run build</code>，将产物复制到 <code>web/dist</code> 后重新编译。</p>
    <p>接口基础路径：<code>/api/v1</code>，健康检查：<code>/api/v1/health</code></p>
  </div>
</body>
</html>`

// assetContentTypes 显式声明构建产物的 MIME 类型。
//
// 不同操作系统的 mime 表并不一致（例如 Windows 注册表可能缺少 .mjs），
// 而浏览器对 ES Module / Worker 的 MIME 校验较严格，这里统一兜底。
var assetContentTypes = map[string]string{
	".js":    "text/javascript; charset=utf-8",
	".mjs":   "text/javascript; charset=utf-8",
	".css":   "text/css; charset=utf-8",
	".json":  "application/json; charset=utf-8",
	".map":   "application/json; charset=utf-8",
	".svg":   "image/svg+xml",
	".png":   "image/png",
	".jpg":   "image/jpeg",
	".jpeg":  "image/jpeg",
	".gif":   "image/gif",
	".webp":  "image/webp",
	".ico":   "image/x-icon",
	".woff":  "font/woff",
	".woff2": "font/woff2",
	".ttf":   "font/ttf",
	".wasm":  "application/wasm",
}

// assetContentType 返回指定文件后缀应使用的 Content-Type，未收录时返回空串。
func assetContentType(name string) string {
	return assetContentTypes[strings.ToLower(path.Ext(name))]
}

// spaHandler 提供嵌入的 SPA 静态资源与 history 路由回退。
// 未匹配到接口的 /api 前缀请求返回 JSON 404，其余路径返回静态文件或 index.html。
func spaHandler() gin.HandlerFunc {
	dist, err := fs.Sub(web.Dist, "dist")
	if err != nil {
		return func(c *gin.Context) {
			resp.Raw(c, http.StatusNotFound, resp.CodeFileNotFound, "页面不存在")
		}
	}
	fileServer := http.FileServer(http.FS(dist))
	return func(c *gin.Context) {
		requestPath := c.Request.URL.Path
		if strings.HasPrefix(requestPath, constants.APIPrefix) || strings.HasPrefix(requestPath, "/api/") {
			resp.Raw(c, http.StatusNotFound, resp.CodeBadParam, "接口不存在")
			return
		}
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			resp.Raw(c, http.StatusMethodNotAllowed, resp.CodeBadParam, "请求方法不允许")
			return
		}
		if name := cleanAssetPath(requestPath); name != "" {
			if info, err := fs.Stat(dist, name); err == nil && !info.IsDir() {
				if contentType := assetContentType(name); contentType != "" {
					c.Header("Content-Type", contentType)
				}
				// 带 hash 的构建产物可长期缓存，其余资源走协商缓存。
				if strings.HasPrefix(name, "assets/") {
					c.Header("Cache-Control", "public, max-age=31536000, immutable")
				} else {
					c.Header("Cache-Control", "public, max-age=3600")
				}
				fileServer.ServeHTTP(c.Writer, c.Request)
				return
			}
		}
		serveIndex(c, dist)
	}
}

// cleanAssetPath 规范化请求路径为可查询的嵌入式相对路径。
func cleanAssetPath(requestPath string) string {
	cleaned := path.Clean("/" + requestPath)
	cleaned = strings.TrimPrefix(cleaned, "/")
	if cleaned == "." {
		return ""
	}
	return cleaned
}

// serveIndex 输出 index.html；产物缺失时返回提示页面。
func serveIndex(c *gin.Context, dist fs.FS) {
	c.Header("Cache-Control", "no-cache")
	data, err := fs.ReadFile(dist, "index.html")
	if err != nil {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(placeholderPage))
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", data)
}

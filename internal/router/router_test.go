package router

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/iokr/novadrive/internal/config"
	"github.com/iokr/novadrive/web"
)

// TestRegisterRoutes 校验全部路由可成功注册（静态段与 :id 参数段共存不冲突）。
func TestRegisterRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// 服务依赖在测试中不参与请求处理，仅验证注册过程不 panic。
	engine := New(Deps{Cfg: &config.Config{}})

	registered := make(map[string]bool, 64)
	for _, r := range engine.Routes() {
		registered[r.Method+" "+r.Path] = true
	}

	required := []string{
		"GET /api/v1/health",
		"POST /api/v1/auth/register",
		"POST /api/v1/auth/login",
		"GET /api/v1/auth/profile",
		"PUT /api/v1/auth/password",
		"GET /api/v1/user/storage",
		"GET /api/v1/files",
		"GET /api/v1/files/breadcrumb",
		"GET /api/v1/files/recent",
		"GET /api/v1/files/favorites",
		"GET /api/v1/files/stats",
		"GET /api/v1/files/recycle",
		"DELETE /api/v1/files/recycle",
		"POST /api/v1/files/mkdir",
		"POST /api/v1/files/upload",
		"POST /api/v1/files/upload/batch",
		"POST /api/v1/files/check",
		"POST /api/v1/files/upload/init",
		"PUT /api/v1/files/upload/:upload_id/chunk",
		"GET /api/v1/files/upload/:upload_id/status",
		"POST /api/v1/files/upload/:upload_id/complete",
		"DELETE /api/v1/files/upload/:upload_id",
		"POST /api/v1/files/download/batch",
		"GET /api/v1/files/:id",
		"GET /api/v1/files/:id/thumb",
		"GET /api/v1/files/:id/raw",
		"GET /api/v1/files/:id/download",
		"PUT /api/v1/files/:id/rename",
		"PUT /api/v1/files/:id/move",
		"PUT /api/v1/files/:id/favorite",
		"POST /api/v1/files/:id/copy",
		"POST /api/v1/files/:id/restore",
		"DELETE /api/v1/files/:id",
		"DELETE /api/v1/files/:id/force",
		"POST /api/v1/shares",
		"GET /api/v1/shares",
		"DELETE /api/v1/shares/:id",
		"GET /api/v1/shares/public/:token",
		"POST /api/v1/shares/public/:token/verify",
		"GET /api/v1/shares/public/:token/files",
		"GET /api/v1/shares/public/:token/preview/:file_id",
		"GET /api/v1/shares/public/:token/download",
		"GET /api/v1/admin/users",
		"PUT /api/v1/admin/users/:id/status",
		"PUT /api/v1/admin/users/:id/quota",
		"GET /api/v1/admin/stats",
		"GET /api/v1/admin/upload-sessions",
	}
	for _, route := range required {
		if !registered[route] {
			t.Errorf("路由缺失: %s", route)
		}
	}
}

// TestSPAFallback 校验 history 路由回退、接口 404 与方法限制。
func TestSPAFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := New(Deps{Cfg: &config.Config{}})

	// 前端路由回退：未匹配路径返回 index.html（产物缺失时为提示页）。
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/dashboard/files", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("前端路由回退期望 200，实际 %d", w.Code)
	}
	if body := w.Body.String(); !strings.Contains(body, "NovaDrive") {
		t.Fatalf("回退页面内容异常: %s", body)
	}

	// 未匹配的接口返回 JSON 404，不落到 SPA。
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/not-exist", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("未匹配接口期望 404，实际 %d", w.Code)
	}
	if body := w.Body.String(); !strings.Contains(body, "\"code\"") {
		t.Fatalf("接口 404 应返回统一 JSON 结构: %s", body)
	}

	// 非 GET/HEAD 的未匹配路径返回 405。
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/dashboard", nil))
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("非 GET 回退期望 405，实际 %d", w.Code)
	}
}

// TestSPAServesBuiltAssets 在 web/dist 已构建时，校验首页与带 hash 的
// 静态资源（含 .mjs worker）能随二进制正确对外提供。
func TestSPAServesBuiltAssets(t *testing.T) {
	dist, err := fs.Sub(web.Dist, "dist")
	if err != nil {
		t.Skip("未嵌入前端产物")
	}
	if _, err := fs.Stat(dist, "index.html"); err != nil {
		t.Skip("web/dist/index.html 不存在，跳过构建产物校验")
	}
	gin.SetMode(gin.TestMode)
	engine := New(Deps{Cfg: &config.Config{}})

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("首页期望 200，实际 %d", w.Code)
	}
	if body := w.Body.String(); !strings.Contains(body, `id="app"`) {
		t.Fatalf("首页应返回构建后的 index.html，实际: %s", body)
	}

	// pdf.js 的 module worker 是唯一 .mjs 产物，其不能被拒绝加载。
	var worker string
	_ = fs.WalkDir(dist, "assets", func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr == nil && !d.IsDir() && strings.HasSuffix(p, ".mjs") {
			worker = p
			return fs.SkipAll
		}
		return nil
	})
	if worker == "" {
		t.Skip("未找到 .mjs 构建产物")
	}

	w = httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/"+worker, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("请求 %s 期望 200，实际 %d", worker, w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/javascript") {
		t.Fatalf("%s 的 Content-Type = %q，期望 text/javascript", worker, ct)
	}
	if cc := w.Header().Get("Cache-Control"); !strings.Contains(cc, "immutable") {
		t.Fatalf("带 hash 的产物应长期缓存，实际 Cache-Control = %q", cc)
	}
}

// TestAssetContentType 校验构建产物的 MIME 兜底，避免不同系统 mime 表差异
// 导致 ES Module / Worker 被浏览器拒绝加载。
func TestAssetContentType(t *testing.T) {
	cases := map[string]string{
		"assets/index-abc123.js":    "text/javascript; charset=utf-8",
		"assets/pdf.worker-abc.mjs": "text/javascript; charset=utf-8",
		"assets/index-abc123.css":   "text/css; charset=utf-8",
		"favicon.svg":               "image/svg+xml",
		"assets/font-abc.woff2":     "font/woff2",
		"assets/unknown.ext":        "",
	}
	for name, want := range cases {
		if got := assetContentType(name); got != want {
			t.Errorf("assetContentType(%q) = %q, 期望 %q", name, got, want)
		}
	}
}

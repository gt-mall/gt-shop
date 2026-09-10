package routers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "mall/admin_core/docs"
)

func TestSwaggerMiniappJSON(t *testing.T) {
	router := testRouter()

	// 8. 不需要 Authorization 请求
	req := httptest.NewRequest(http.MethodGet, "/swagger/miniapp/swagger.json", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// 1. GET /swagger/miniapp/swagger.json 返回 200
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /swagger/miniapp/swagger.json status = %d, want %d", rec.Code, http.StatusOK)
	}

	// 2. Content-Type 是 application/json
	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Fatalf("Content-Type = %q, want application/json", contentType)
	}

	// 3. JSON 能正常解析
	var doc struct {
		Swagger  string                    `json:"swagger"`
		BasePath string                    `json:"basePath"`
		Paths    map[string]map[string]any `json:"paths"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatalf("failed to parse swagger JSON: %v", err)
	}

	// 4. swagger == "2.0"
	if doc.Swagger != "2.0" {
		t.Errorf("doc.Swagger = %q, want '2.0'", doc.Swagger)
	}

	// 5. basePath == "/api/miniapp"
	if doc.BasePath != "/api/miniapp" {
		t.Errorf("doc.BasePath = %q, want '/api/miniapp'", doc.BasePath)
	}

	// 6. 包含 miniapp API
	expectedMiniappPaths := []string{
		"/auth/wechat-login",
		"/auth/register",
		"/auth/login",
		"/member/me",
		"/regions",
		"/products",
		"/products/{id}",
	}
	for _, p := range expectedMiniappPaths {
		if _, ok := doc.Paths[p]; !ok {
			t.Errorf("expected miniapp path %q not found in swagger paths: %v", p, doc.Paths)
		}
	}

	// 7. 不包含后台 system/product/member 管理端 等 Admin API
	adminPathPrefixes := []string{
		"/api/system",
		"/api/product",
		"/api/member",
		"/system",
		"/product/",
	}
	for p := range doc.Paths {
		for _, prefix := range adminPathPrefixes {
			if strings.HasPrefix(p, prefix) {
				t.Errorf("admin path %q should not be present in miniapp swagger", p)
			}
		}
	}
}

func TestSwaggerAdminUIAndDoc(t *testing.T) {
	router := testRouter()

	// 9. 后台 Swagger 路由仍然能访问
	// 9.1 后台 Swagger index.html
	req := httptest.NewRequest(http.MethodGet, "/swagger/admin/index.html", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /swagger/admin/index.html status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "swagger-ui") {
		t.Errorf("expected swagger-ui in body, got: %s", rec.Body.String()[:min(len(rec.Body.String()), 200)])
	}

	// 9.2 后台 Swagger doc.json
	reqDoc := httptest.NewRequest(http.MethodGet, "/swagger/admin/doc.json", nil)
	recDoc := httptest.NewRecorder()
	router.ServeHTTP(recDoc, reqDoc)
	if recDoc.Code != http.StatusOK {
		t.Fatalf("GET /swagger/admin/doc.json status = %d, want %d", recDoc.Code, http.StatusOK)
	}
	var adminDoc struct {
		Swagger  string                    `json:"swagger"`
		BasePath string                    `json:"basePath"`
		Paths    map[string]map[string]any `json:"paths"`
	}
	if err := json.Unmarshal(recDoc.Body.Bytes(), &adminDoc); err != nil {
		t.Fatalf("failed to parse admin doc.json: %v", err)
	}
	if adminDoc.Swagger != "2.0" {
		t.Errorf("adminDoc.Swagger = %q, want '2.0'", adminDoc.Swagger)
	}
	if adminDoc.BasePath != "/" {
		t.Errorf("adminDoc.BasePath = %q, want '/'", adminDoc.BasePath)
	}
	if _, ok := adminDoc.Paths["/api/auth/login"]; !ok {
		t.Errorf("adminDoc.Paths missing /api/auth/login")
	}

	// 9.3 兼容重定向
	for _, redirectPath := range []string{"/swagger", "/swagger/index.html"} {
		reqRedir := httptest.NewRequest(http.MethodGet, redirectPath, nil)
		recRedir := httptest.NewRecorder()
		router.ServeHTTP(recRedir, reqRedir)
		if recRedir.Code != http.StatusMovedPermanently {
			t.Errorf("GET %s status = %d, want %d", redirectPath, recRedir.Code, http.StatusMovedPermanently)
		}
		loc := recRedir.Header().Get("Location")
		if loc != "/swagger/admin/index.html" {
			t.Errorf("GET %s Location = %q, want '/swagger/admin/index.html'", redirectPath, loc)
		}
	}
}

func TestNoGinWildcardRouteConflict(t *testing.T) {
	// 10. NewRouter 启动时不存在 Gin wildcard route conflict
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("NewRouter panicked with: %v", r)
		}
	}()

	_ = testRouter()
}

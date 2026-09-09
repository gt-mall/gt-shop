package routers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"mall/admin_core/bootstrap"
	"mall/admin_core/internal/auth"
	"mall/admin_core/pkg/response"
)

func testRouter() http.Handler {
	return NewRouter(bootstrap.Config{}, nil, auth.NewService(nil, nil, "", time.Hour))
}

func TestSystemRoutesRequireAuthentication(t *testing.T) {
	router := testRouter()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("POST", "/api/system/users/list", strings.NewReader(`{}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	var result response.Response[any]
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.Code != -1 {
		t.Fatalf("response code = %d, want -1", result.Code)
	}
}

func TestDemoRoutes(t *testing.T) {
	router := testRouter()

	for _, path := range []string{"/health", "/api/demo", "/api/plugins/hello", "/api/miniapp/hello"} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest("GET", path, nil)
		router.ServeHTTP(recorder, request)
		if recorder.Code != 200 {
			t.Fatalf("GET %s status = %d, want 200", path, recorder.Code)
		}
	}
}

func TestSystemUserRoutesUsePost(t *testing.T) {
	router := testRouter()

	tests := []struct {
		path string
		body string
	}{
		{path: "/api/system/users/create", body: `{"username":"admin","password":"password"}`},
		{path: "/api/system/users/list", body: `{}`},
		{path: "/api/system/users/detail", body: `{"id":1}`},
		{path: "/api/system/users/update", body: `{"id":1,"nickname":"Admin"}`},
		{path: "/api/system/users/delete", body: `{"id":1}`},
	}

	for _, item := range tests {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest("POST", item.path, strings.NewReader(item.body))
		request.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(recorder, request)
		if recorder.Code != 200 {
			t.Fatalf("POST %s status = %d, want 200", item.path, recorder.Code)
		}
	}
}

func TestSystemRoleRoutesUsePost(t *testing.T) {
	router := testRouter()

	tests := []struct {
		path string
		body string
	}{
		{path: "/api/system/roles/create", body: `{"name":"管理员"}`},
		{path: "/api/system/roles/list", body: `{}`},
		{path: "/api/system/roles/detail", body: `{"id":1}`},
		{path: "/api/system/roles/update", body: `{"id":1,"remark":"系统管理员"}`},
		{path: "/api/system/roles/delete", body: `{"id":1}`},
	}

	for _, item := range tests {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest("POST", item.path, strings.NewReader(item.body))
		request.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(recorder, request)
		if recorder.Code != 200 {
			t.Fatalf("POST %s status = %d, want 200", item.path, recorder.Code)
		}
	}
}

func TestProductRoutesUsePostAndRequireAuth(t *testing.T) {
	router := testRouter()

	paths := []string{
		"/api/product/categories/list",
		"/api/product/categories/options",
		"/api/product/categories/create",
		"/api/product/categories/update",
		"/api/product/categories/delete",
		"/api/product/products/list",
		"/api/product/products/detail",
		"/api/product/products/create",
		"/api/product/products/update",
		"/api/product/products/status",
		"/api/product/products/delete",
	}

	for _, p := range paths {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest("POST", p, strings.NewReader(`{}`))
		request.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(recorder, request)

		if recorder.Code != 200 {
			t.Fatalf("POST %s HTTP status = %d, want 200", p, recorder.Code)
		}
		var result response.Response[any]
		if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
			t.Fatalf("decode response for %s: %v", p, err)
		}
		if result.Code != -1 {
			t.Fatalf("POST %s response code = %d, want -1 (unauthorized)", p, result.Code)
		}
	}
}

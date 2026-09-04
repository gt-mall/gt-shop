package routers

import (
	"net/http/httptest"
	"strings"
	"testing"

	"mall/admin_core/bootstrap"
)

func TestDemoRoutes(t *testing.T) {
	router := NewRouter(bootstrap.Config{}, nil)

	for _, path := range []string{"/health", "/api/demo", "/api/plugins/hello"} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest("GET", path, nil)
		router.ServeHTTP(recorder, request)
		if recorder.Code != 200 {
			t.Fatalf("GET %s status = %d, want 200", path, recorder.Code)
		}
	}
}

func TestSystemUserRoutesUsePost(t *testing.T) {
	router := NewRouter(bootstrap.Config{}, nil)

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
	router := NewRouter(bootstrap.Config{}, nil)

	tests := []struct {
		path string
		body string
	}{
		{path: "/api/system/roles/create", body: `{"name":"管理员","code":"admin"}`},
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

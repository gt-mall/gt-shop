package miniapp_test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"mall/admin_core/bootstrap"
	"mall/admin_core/internal/auth"
	miniappcommon "mall/admin_core/internal/miniapp/common"
	"mall/admin_core/routers"
)

func TestMiniappHelloService(t *testing.T) {
	service := miniappcommon.NewService()
	res := service.Hello()
	if res.Text != "Hello World" {
		t.Fatalf("want 'Hello World', got %q", res.Text)
	}
}

func TestMiniappHelloEndpoint(t *testing.T) {
	router := routers.NewRouter(bootstrap.Config{}, nil, auth.NewService(nil, nil, "", time.Hour))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/api/miniapp/hello", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != 200 {
		t.Fatalf("GET /api/miniapp/hello status = %d, want 200", recorder.Code)
	}
	var result struct {
		Code    int                         `json:"code"`
		Message string                      `json:"message"`
		Total   int                         `json:"total"`
		Data    miniappcommon.HelloResponse `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if result.Code != 1 || result.Data.Text != "Hello World" {
		t.Fatalf("unexpected payload: %+v", result)
	}
}

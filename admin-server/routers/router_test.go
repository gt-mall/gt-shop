package routers

import (
	"net/http/httptest"
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

package system_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mall/admin_core/bootstrap"
	"mall/admin_core/internal/auth"
	"mall/admin_core/routers"
)

func TestSystemVersionEndpoint(t *testing.T) {
	router := routers.NewRouter(bootstrap.Config{}, nil, auth.NewService(nil, nil, "", time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/api/system/version", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Version   string `json:"version"`
			BuildTime string `json:"build_time"`
			GitCommit string `json:"git_commit"`
		} `json:"data"`
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}

	if resp.Code != 1 {
		t.Errorf("expected code 1, got %d", resp.Code)
	}
	if resp.Data.Version == "" {
		t.Errorf("expected non-empty version")
	}
	if resp.Data.BuildTime == "" {
		t.Errorf("expected non-empty build_time")
	}
	if resp.Data.GitCommit == "" {
		t.Errorf("expected non-empty git_commit")
	}
}

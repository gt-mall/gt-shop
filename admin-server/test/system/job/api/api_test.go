package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/casbin/casbin/v2"
	casbinmodel "github.com/casbin/casbin/v2/model"
	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"mall/admin_core/bootstrap"
	"mall/admin_core/internal/auth"
	"mall/admin_core/internal/model"
	"mall/admin_core/internal/system/job"
	"mall/admin_core/pkg/response"
	"mall/admin_core/routers"
)

const testCasbinModel = `[request_definition]
r = sub, perm
[policy_definition]
p = sub, perm
[role_definition]
g = _, _
[policy_effect]
e = some(where (p.eft == allow))
[matchers]
m = g(r.sub, p.sub) && r.perm == p.perm
`

func TestJobAPIAuthenticationPermissionsAndNameValidation(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:job-api?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.SystemUser{}, &model.SystemRole{}, &model.SystemJobStatus{}); err != nil {
		t.Fatal(err)
	}

	casbinModel, err := casbinmodel.NewModelFromString(testCasbinModel)
	if err != nil {
		t.Fatal(err)
	}
	enforcer, err := casbin.NewSyncedEnforcer(casbinModel)
	if err != nil {
		t.Fatal(err)
	}
	for _, permission := range []string{auth.PermissionSystemJobList, auth.PermissionSystemJobRetry} {
		if !auth.IsPermissionCode(permission) {
			t.Fatalf("permission %s is not registered", permission)
		}
	}
	listToken := createSession(t, db, enforcer, "job_list", auth.PermissionSystemJobList)
	retryToken := createSession(t, db, enforcer, "job_retry", auth.PermissionSystemJobRetry)
	noPermissionToken := createSession(t, db, enforcer, "job_none")

	manager := job.NewManager(db)
	if err := manager.Register(job.Job{Name: "api_job", DisplayName: "API 任务", Run: func(context.Context) (job.RunResult, error) {
		return job.RunResult{}, nil
	}}); err != nil {
		t.Fatal(err)
	}
	router := routers.NewRouterWithJobs(bootstrap.Config{}, db, auth.NewService(db, enforcer, "test-secret", time.Hour), manager)
	beforeStart := request(router, http.MethodGet, "/api/system/jobs", listToken)
	if beforeStart.Code != 0 || !strings.Contains(beforeStart.Message, "任务管理器未运行") {
		t.Fatalf("unstarted manager must not show historical job status: %+v", beforeStart)
	}
	if err := manager.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(manager.Close)

	unauthorized := request(router, http.MethodGet, "/api/system/jobs", "")
	if unauthorized.Code != -1 {
		t.Fatalf("unauthenticated list code = %d, want -1", unauthorized.Code)
	}

	listed := request(router, http.MethodGet, "/api/system/jobs", listToken)
	if listed.Code != 1 || len(listed.Data) != 1 {
		t.Fatalf("authorized list response = %+v", listed)
	}

	denied := request(router, http.MethodGet, "/api/system/jobs", noPermissionToken)
	if denied.Code != 0 || denied.Message != "permission denied" {
		t.Fatalf("list permission response = %+v", denied)
	}

	retryDenied := request(router, http.MethodPost, "/api/system/jobs/not_registered/retry", noPermissionToken)
	if retryDenied.Code != 0 || retryDenied.Message != "permission denied" {
		t.Fatalf("retry permission response = %+v", retryDenied)
	}

	unknown := request(router, http.MethodPost, "/api/system/jobs/not_registered/retry", retryToken)
	if unknown.Code != 0 || unknown.Message != "未知任务" {
		t.Fatalf("unknown task response = %+v", unknown)
	}
}

func createSession(t *testing.T, db *gorm.DB, enforcer *casbin.SyncedEnforcer, suffix string, permissions ...string) string {
	t.Helper()
	role := model.SystemRole{Name: suffix, Code: suffix, Status: 1, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := db.Create(&role).Error; err != nil {
		t.Fatal(err)
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("test-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	user := model.SystemUser{Username: suffix, PasswordHash: string(passwordHash), Nickname: suffix, Status: 1, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := enforcer.AddRoleForUser(auth.UserSubject(user.ID), role.Code); err != nil {
		t.Fatal(err)
	}
	for _, permission := range permissions {
		if _, err := enforcer.AddPolicy(role.Code, permission); err != nil {
			t.Fatal(err)
		}
	}
	login, err := auth.NewService(db, enforcer, "test-secret", time.Hour).Login(context.Background(), auth.LoginRequest{Username: user.Username, Password: "test-password"})
	if err != nil {
		t.Fatal(err)
	}
	return login.Token
}

func request(router http.Handler, method, path, token string) response.Response[[]job.JobStatusResponse] {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(""))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	router.ServeHTTP(recorder, req)
	var result response.Response[[]job.JobStatusResponse]
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	return result
}

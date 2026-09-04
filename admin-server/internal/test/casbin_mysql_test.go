package test

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"mall/admin_core/bootstrap"
	"mall/admin_core/internal/auth"
)

func TestCasbinMySQLPersistence(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN is not set")
	}
	db, err := bootstrap.NewDB(context.Background(), bootstrap.Config{DatabaseDSN: dsn, DatabaseSet: true})
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get database handle: %v", err)
	}
	defer sqlDB.Close()

	enforcer, err := bootstrap.NewCasbinEnforcer(db)
	if err != nil {
		t.Fatalf("create first enforcer: %v", err)
	}
	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	subject := "user:test_" + suffix
	role := "role_test_" + suffix
	defer func() {
		_, _ = enforcer.DeleteUser(subject)
		_, _ = enforcer.DeleteRole(role)
	}()

	if _, err := enforcer.AddRoleForUser(subject, role); err != nil {
		t.Fatalf("persist role assignment: %v", err)
	}
	if _, err := enforcer.AddPermissionForUser(role, auth.PermissionSystemUserList); err != nil {
		t.Fatalf("persist role permission: %v", err)
	}

	reloaded, err := bootstrap.NewCasbinEnforcer(db)
	if err != nil {
		t.Fatalf("create reloaded enforcer: %v", err)
	}
	allowed, err := reloaded.Enforce(subject, auth.PermissionSystemUserList)
	if err != nil || !allowed {
		t.Fatalf("reloaded policy allowed = %v, err = %v", allowed, err)
	}
}

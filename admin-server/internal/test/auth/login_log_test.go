package auth_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm"

	"mall/admin_core/internal/auth"
	"mall/admin_core/internal/model"
	"mall/admin_core/pkg/logger"
)

func setupAuthLogTest(t *testing.T) (*gorm.DB, *auth.Service, string) {
	t.Helper()
	tmpDir := t.TempDir()
	logCfg := logger.Config{
		Level:      "debug",
		Path:       tmpDir,
		Console:    false,
		File:       true,
		MaxSize:    10,
		MaxAge:     1,
		MaxBackups: 1,
	}
	if err := logger.Init(logCfg); err != nil {
		t.Fatalf("logger init: %v", err)
	}
	t.Cleanup(func() {
		_ = logger.Close()
	})

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("sqlite open: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })

	if err := db.AutoMigrate(&model.SystemUser{}); err != nil {
		t.Fatalf("migrate SystemUser: %v", err)
	}

	passwordHash, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	normalUser := model.SystemUser{
		ID:           1,
		Username:     "admin_test",
		Nickname:     "Admin Tester",
		PasswordHash: string(passwordHash),
		Status:       1,
	}
	disabledUser := model.SystemUser{
		ID:           2,
		Username:     "disabled_user",
		Nickname:     "Disabled Tester",
		PasswordHash: string(passwordHash),
		Status:       0,
	}
	if err := db.Create(&normalUser).Error; err != nil {
		t.Fatalf("create normal user: %v", err)
	}
	if err := db.Create(&disabledUser).Error; err != nil {
		t.Fatalf("create disabled user: %v", err)
	}
	if err := db.Model(&disabledUser).Update("status", 0).Error; err != nil {
		t.Fatalf("set status 0: %v", err)
	}

	service := auth.NewService(db, nil, "test_jwt_secret_key_1234567890", 24*time.Hour)
	return db, service, tmpDir
}

func TestAdminLoginLogging(t *testing.T) {
	_, service, tmpDir := setupAuthLogTest(t)
	appLogFile := filepath.Join(tmpDir, "app.log")

	ctx := auth.WithClientInfo(context.Background(), "127.0.0.1", "TestAgent/1.0")

	// 1. Success login
	_, err := service.Login(ctx, auth.LoginRequest{
		Username: "admin_test",
		Password: "123456",
		Remember: true,
	})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	_ = logger.Sync()

	content, err := os.ReadFile(appLogFile)
	if err != nil {
		t.Fatalf("read app.log: %v", err)
	}
	logStr := string(content)
	if !strings.Contains(logStr, "admin login success") {
		t.Errorf("expected 'admin login success' in app.log, got:\n%s", logStr)
	}
	if !strings.Contains(logStr, "admin_test") {
		t.Errorf("expected username in log, got:\n%s", logStr)
	}
	if !strings.Contains(logStr, "127.0.0.1") {
		t.Errorf("expected IP in log, got:\n%s", logStr)
	}
	if strings.Contains(logStr, "123456") {
		t.Errorf("plain password must NEVER be logged!")
	}

	// 2. Non-existent user
	_, err = service.Login(ctx, auth.LoginRequest{
		Username: "no_such_user",
		Password: "password",
	})
	if err == nil {
		t.Fatal("expected error for non-existent user")
	}
	_ = logger.Sync()

	content, _ = os.ReadFile(appLogFile)
	logStr = string(content)
	if !strings.Contains(logStr, "admin login failed: user not found") {
		t.Errorf("expected 'admin login failed: user not found' in app.log, got:\n%s", logStr)
	}

	// 3. Invalid password
	_, err = service.Login(ctx, auth.LoginRequest{
		Username: "admin_test",
		Password: "wrongpassword",
	})
	if err == nil {
		t.Fatal("expected error for wrong password")
	}
	_ = logger.Sync()

	content, _ = os.ReadFile(appLogFile)
	logStr = string(content)
	if !strings.Contains(logStr, "admin login failed: invalid password") {
		t.Errorf("expected 'admin login failed: invalid password' in app.log, got:\n%s", logStr)
	}

	// 4. Disabled user
	_, err = service.Login(ctx, auth.LoginRequest{
		Username: "disabled_user",
		Password: "123456",
	})
	if err == nil {
		t.Fatal("expected error for disabled user")
	}
	_ = logger.Sync()

	content, _ = os.ReadFile(appLogFile)
	logStr = string(content)
	if !strings.Contains(logStr, "admin login failed: account disabled") {
		t.Errorf("expected 'admin login failed: account disabled' in app.log, got:\n%s", logStr)
	}
}

func TestAdminChangePasswordLogging(t *testing.T) {
	db, service, tmpDir := setupAuthLogTest(t)
	appLogFile := filepath.Join(tmpDir, "app.log")

	ctx := auth.WithClientInfo(context.Background(), "10.0.0.1", "Mozilla/5.0")
	var user model.SystemUser
	if err := db.First(&user, 1).Error; err != nil {
		t.Fatalf("find user: %v", err)
	}

	// 1. Wrong current password
	err := service.ChangePassword(ctx, user, auth.ChangePasswordRequest{
		CurrentPassword: "wrongpassword",
		NewPassword:     "654321",
	})
	if err == nil {
		t.Fatal("expected error for wrong current password")
	}
	_ = logger.Sync()

	content, _ := os.ReadFile(appLogFile)
	logStr := string(content)
	if !strings.Contains(logStr, "admin change password failed: incorrect current password") {
		t.Errorf("expected 'incorrect current password' log, got:\n%s", logStr)
	}

	// 2. Success change password
	err = service.ChangePassword(ctx, user, auth.ChangePasswordRequest{
		CurrentPassword: "123456",
		NewPassword:     "newpassword666",
	})
	if err != nil {
		t.Fatalf("change password failed: %v", err)
	}
	_ = logger.Sync()

	content, _ = os.ReadFile(appLogFile)
	logStr = string(content)
	if !strings.Contains(logStr, "admin change password success") {
		t.Errorf("expected 'admin change password success' in app.log, got:\n%s", logStr)
	}
	if strings.Contains(logStr, "newpassword666") {
		t.Errorf("new password must NEVER appear in logs!")
	}
}

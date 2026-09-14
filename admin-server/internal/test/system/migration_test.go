package system_test

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"mall/admin_core/bootstrap"
	"mall/admin_core/data"
	"mall/admin_core/internal/auth"
)

// TestEmbeddedMigrationsReading verifies that migrations are properly embedded in data.FS,
// can be read without disk dependencies, and are in lexical order.
func TestEmbeddedMigrationsReading(t *testing.T) {
	// 1. embedded FS 中可以读取 migrations
	entries, err := fs.ReadDir(data.FS, "migrations")
	if err != nil {
		t.Fatalf("fs.ReadDir(data.FS, 'migrations') failed: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected migrations directory to contain SQL files, got 0")
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".sql" {
			files = append(files, entry.Name())
		}
	}

	// 2. migration SQL 能按文件名排序
	isSorted := sort.StringsAreSorted(files)
	if !isSorted {
		t.Fatalf("migrations are not in lexical order: %v", files)
	}

	// 验证关键迁移文件存在并可读取内容
	expectedFiles := []string{
		"001_baseline.sql",
	}

	fileMap := make(map[string]bool)
	for _, f := range files {
		fileMap[f] = true
		content, err := fs.ReadFile(data.FS, path.Join("migrations", f))
		if err != nil {
			t.Fatalf("failed to read embedded migration file %s: %v", f, err)
		}
		if len(strings.TrimSpace(string(content))) == 0 {
			t.Fatalf("migration file %s is empty", f)
		}
	}

	for _, exp := range expectedFiles {
		if !fileMap[exp] {
			t.Errorf("expected migration file %s not found in embedded FS", exp)
		}
	}
	if len(files) != len(expectedFiles) {
		t.Errorf("expected %d migration file(s), got %d: %v", len(expectedFiles), len(files), files)
	}
}

// TestEmbeddedSeedsReading verifies that seed files are properly embedded in data.FS under seeds/.
func TestEmbeddedSeedsReading(t *testing.T) {
	entries, err := fs.ReadDir(data.FS, "seeds")
	if err != nil {
		t.Fatalf("fs.ReadDir(data.FS, 'seeds') failed: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected seeds directory to contain SQL files, got 0")
	}

	expectedSeeds := []string{
		"001_default_data.sql",
	}
	fileMap := make(map[string]bool)
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".sql" {
			fileMap[entry.Name()] = true
			content, err := fs.ReadFile(data.FS, path.Join("seeds", entry.Name()))
			if err != nil {
				t.Fatalf("failed to read embedded seed file %s: %v", entry.Name(), err)
			}
			if len(strings.TrimSpace(string(content))) == 0 {
				t.Fatalf("seed file %s is empty", entry.Name())
			}
		}
	}

	for _, exp := range expectedSeeds {
		if !fileMap[exp] {
			t.Errorf("expected seed file %s not found in embedded FS", exp)
		}
	}
}

// TestMigrationRunFromEmptyCwd verifies:
// - 不依赖 ./migrations 实体目录
// - 从不同工作目录 (例如空的临时目录) 启动也不影响 migration
// - 已执行 migration 不重复执行
// - 真实 embedded data.FS 迁移执行成功
// TestDatabaseDSNIsolation verifies:
// 1. TEST_DATABASE_DSN 为空时，集成测试 Skip，绝不回退到 DATABASE_DSN。
// 2. DATABASE_DSN 有值而 TEST_DATABASE_DSN 为空时，必须 Skip，绝不触碰正常开发库。
// 3. TEST_DATABASE_DSN == DATABASE_DSN 时，报错拒绝执行破坏性测试。
func TestDatabaseDSNIsolation(t *testing.T) {
	// 1. TEST_DATABASE_DSN is empty -> ErrTestDSNEmpty
	_, err := bootstrap.ValidateTestDSN("", "")
	if !errors.Is(err, bootstrap.ErrTestDSNEmpty) {
		t.Fatalf("expected ErrTestDSNEmpty when TEST_DATABASE_DSN is empty, got %v", err)
	}

	// 2. DATABASE_DSN is set, but TEST_DATABASE_DSN is empty -> must still return ErrTestDSNEmpty
	_, err = bootstrap.ValidateTestDSN("", "root:password@tcp(127.0.0.1:3306)/ai_shop")
	if !errors.Is(err, bootstrap.ErrTestDSNEmpty) {
		t.Fatalf("expected ErrTestDSNEmpty even when DATABASE_DSN is set, got %v", err)
	}

	// 3. TEST_DATABASE_DSN == DATABASE_DSN -> ErrTestDSNEqualsDev ("TEST_DATABASE_DSN must not equal DATABASE_DSN")
	devDSN := "root:password@tcp(127.0.0.1:3306)/ai_shop"
	_, err = bootstrap.ValidateTestDSN(devDSN, devDSN)
	if !errors.Is(err, bootstrap.ErrTestDSNEqualsDev) {
		t.Fatalf("expected ErrTestDSNEqualsDev when TEST_DATABASE_DSN == DATABASE_DSN, got %v", err)
	}
	if err.Error() != "TEST_DATABASE_DSN must not equal DATABASE_DSN" {
		t.Fatalf("expected error message 'TEST_DATABASE_DSN must not equal DATABASE_DSN', got %v", err)
	}

	// 4. Different query parameters pointing to same database
	dsnWithParam1 := "root:password@tcp(127.0.0.1:3306)/ai_shop?parseTime=true"
	dsnWithParam2 := "root:password@tcp(127.0.0.1:3306)/ai_shop?charset=utf8mb4"
	_, err = bootstrap.ValidateTestDSN(dsnWithParam1, dsnWithParam2)
	if !errors.Is(err, bootstrap.ErrTestDSNEqualsDev) {
		t.Fatalf("expected ErrTestDSNEqualsDev when pointing to the same host and database, got %v", err)
	}

	// 5. Valid test DSN pointing to different database
	testDSN := "root:password@tcp(127.0.0.1:3306)/ai_shop_test"
	valid, err := bootstrap.ValidateTestDSN(testDSN, devDSN)
	if err != nil || valid != testDSN {
		t.Fatalf("expected valid test DSN, got %q, err %v", valid, err)
	}
	if bootstrap.TestDBName(testDSN) != "ai_shop_test" {
		t.Fatalf("expected DBName 'ai_shop_test', got %q", bootstrap.TestDBName(testDSN))
	}
}

// TestCreateAdmin_Validation verifies input validation on CreateAdmin without requiring DB.
func TestCreateAdmin_Validation(t *testing.T) {
	ctx := context.Background()
	// Password empty
	_, err := bootstrap.CreateAdmin(ctx, nil, nil, "admin", "", "超级管理员")
	if !errors.Is(err, bootstrap.ErrPasswordRequired) {
		t.Fatalf("expected ErrPasswordRequired when password is empty, got %v", err)
	}

	// Username empty
	_, err = bootstrap.CreateAdmin(ctx, nil, nil, "", "password123", "超级管理员")
	if !errors.Is(err, bootstrap.ErrUsernameRequired) {
		t.Fatalf("expected ErrUsernameRequired when username is empty, got %v", err)
	}
}

// TestMigrationRunFromEmptyCwd verifies:
// - 不依赖 ./migrations 实体目录
// - 从不同工作目录 (例如空的临时目录) 启动也不影响 migration
// - 已执行 migration 不重复执行
// - 真实 embedded data.FS 迁移执行成功
func TestMigrationRunFromEmptyCwd(t *testing.T) {
	dsn, err := bootstrap.ValidateTestDSN(os.Getenv("TEST_DATABASE_DSN"), os.Getenv("DATABASE_DSN"))
	if errors.Is(err, bootstrap.ErrTestDSNEmpty) {
		t.Skip("TEST_DATABASE_DSN is not configured, skipping MySQL migration test")
	}
	if err != nil {
		t.Fatalf("test database configuration error: %v", err)
	}
	if dbName := bootstrap.TestDBName(dsn); dbName != "" {
		t.Logf("running test against database: %s", dbName)
	}

	ctx := context.Background()
	db, err := bootstrap.NewDB(ctx, bootstrap.Config{
		DatabaseDSN: dsn,
		DatabaseSet: true,
	})
	if err != nil {
		t.Fatalf("connect to database failed: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get database handle failed: %v", err)
	}
	defer sqlDB.Close()

	// 切换到完全没有任何 migrations 或 data 目录的空临时目录
	origCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir to tempDir: %v", err)
	}
	defer func() {
		_ = os.Chdir(origCwd)
	}()

	// 确认当前工作目录下没有 migrations 或 data 目录
	if _, err := os.Stat("migrations"); !os.IsNotExist(err) {
		t.Fatal("expected no migrations directory in tempDir")
	}
	if _, err := os.Stat("data"); !os.IsNotExist(err) {
		t.Fatal("expected no data directory in tempDir")
	}

	// 执行 migration，验证在空目录中也能正常通过 embedded data.FS 运行
	if err := bootstrap.RunMigrations(ctx, db); err != nil {
		t.Fatalf("RunMigrations failed from empty working directory: %v", err)
	}

	// 验证已执行 migration 不重复执行
	var beforeCount int64
	if err := db.Raw("SELECT COUNT(*) FROM schema_migrations").Scan(&beforeCount).Error; err != nil {
		t.Fatalf("count schema_migrations failed: %v", err)
	}
	if beforeCount == 0 {
		t.Fatal("expected schema_migrations to contain records")
	}

	// 再次调用 RunMigrations
	if err := bootstrap.RunMigrations(ctx, db); err != nil {
		t.Fatalf("second RunMigrations failed: %v", err)
	}

	var afterCount int64
	if err := db.Raw("SELECT COUNT(*) FROM schema_migrations").Scan(&afterCount).Error; err != nil {
		t.Fatalf("count schema_migrations after second run failed: %v", err)
	}
	if beforeCount != afterCount {
		t.Fatalf("migration count changed after second run: before=%d, after=%d", beforeCount, afterCount)
	}
}

// TestRunSeedsAndAlignBaseline verifies that:
// - RunSeeds can be executed safely and idempotently
// - AlignBaseline resets schema_migrations to 001_baseline.sql and applies seeds idempotently
func TestRunSeedsAndAlignBaseline(t *testing.T) {
	dsn, err := bootstrap.ValidateTestDSN(os.Getenv("TEST_DATABASE_DSN"), os.Getenv("DATABASE_DSN"))
	if errors.Is(err, bootstrap.ErrTestDSNEmpty) {
		t.Skip("TEST_DATABASE_DSN is not configured, skipping MySQL test")
	}
	if err != nil {
		t.Fatalf("test database configuration error: %v", err)
	}
	if dbName := bootstrap.TestDBName(dsn); dbName != "" {
		t.Logf("running test against database: %s", dbName)
	}

	ctx := context.Background()
	db, err := bootstrap.NewDB(ctx, bootstrap.Config{
		DatabaseDSN: dsn,
		DatabaseSet: true,
	})
	if err != nil {
		t.Fatalf("connect to database failed: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get database handle failed: %v", err)
	}
	defer sqlDB.Close()

	if err := bootstrap.RunSeeds(ctx, db); err != nil {
		t.Fatalf("RunSeeds failed: %v", err)
	}

	if err := bootstrap.AlignBaseline(ctx, db); err != nil {
		t.Fatalf("AlignBaseline failed: %v", err)
	}

	var migrationCount int64
	if err := db.Raw("SELECT COUNT(*) FROM schema_migrations WHERE version = '001_baseline.sql'").Scan(&migrationCount).Error; err != nil {
		t.Fatalf("count baseline migration failed: %v", err)
	}
	if migrationCount != 1 {
		t.Fatalf("expected 1 record for 001_baseline.sql, got %d", migrationCount)
	}
}

// TestAlignBaseline_RejectsInvalidSchema verifies that:
// 当数据库结构不符合 V1 Baseline 要求 (例如 after_sale_item.sku_text = VARCHAR(255))：
// - AlignBaseline 立即报错拒绝执行
// - 报错信息中包含具体字段差异
// - schema_migrations 不会被修改
func TestAlignBaseline_RejectsInvalidSchema(t *testing.T) {
	dsn, err := bootstrap.ValidateTestDSN(os.Getenv("TEST_DATABASE_DSN"), os.Getenv("DATABASE_DSN"))
	if errors.Is(err, bootstrap.ErrTestDSNEmpty) {
		t.Skip("TEST_DATABASE_DSN is not configured, skipping MySQL test")
	}
	if err != nil {
		t.Fatalf("test database configuration error: %v", err)
	}

	ctx := context.Background()
	db, err := bootstrap.NewDB(ctx, bootstrap.Config{
		DatabaseDSN: dsn,
		DatabaseSet: true,
	})
	if err != nil {
		t.Fatalf("connect to database failed: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get database handle failed: %v", err)
	}
	defer sqlDB.Close()

	// Ensure baseline is currently applied
	if err := bootstrap.RunInit(ctx, db); err != nil {
		t.Fatalf("RunInit failed: %v", err)
	}

	var beforeCount int64
	if err := db.Raw("SELECT COUNT(*) FROM schema_migrations").Scan(&beforeCount).Error; err != nil {
		t.Fatalf("count schema_migrations: %v", err)
	}

	// Alter after_sale_item.sku_text to VARCHAR(255) to simulate schema mismatch
	if err := db.Exec("ALTER TABLE after_sale_item MODIFY sku_text VARCHAR(255) NOT NULL DEFAULT ''").Error; err != nil {
		t.Fatalf("alter table for test: %v", err)
	}
	defer func() {
		_ = db.Exec("ALTER TABLE after_sale_item MODIFY sku_text VARCHAR(512) NOT NULL DEFAULT ''").Error
	}()

	// AlignBaseline must fail
	alignErr := bootstrap.AlignBaseline(ctx, db)
	if alignErr == nil {
		t.Fatal("expected AlignBaseline to fail on invalid schema (sku_text VARCHAR(255)), but it succeeded")
	}
	if !strings.Contains(alignErr.Error(), "after_sale_item.sku_text expected VARCHAR(512)") {
		t.Fatalf("expected error message to contain 'after_sale_item.sku_text expected VARCHAR(512)', got: %v", alignErr)
	}

	// schema_migrations must remain unchanged
	var afterCount int64
	if err := db.Raw("SELECT COUNT(*) FROM schema_migrations").Scan(&afterCount).Error; err != nil {
		t.Fatalf("count schema_migrations: %v", err)
	}
	if beforeCount != afterCount {
		t.Fatalf("schema_migrations was modified despite validation failure: before=%d, after=%d", beforeCount, afterCount)
	}
}

// TestAlignBaseline_NeverMarksFutureMigrations verifies that:
// 即使目录中存在未来迁移文件 (如 002_fake.sql)，align 也绝对不能标记 002_fake.sql。
func TestAlignBaseline_NeverMarksFutureMigrations(t *testing.T) {
	dsn, err := bootstrap.ValidateTestDSN(os.Getenv("TEST_DATABASE_DSN"), os.Getenv("DATABASE_DSN"))
	if errors.Is(err, bootstrap.ErrTestDSNEmpty) {
		t.Skip("TEST_DATABASE_DSN is not configured, skipping MySQL test")
	}
	if err != nil {
		t.Fatalf("test database configuration error: %v", err)
	}

	ctx := context.Background()
	db, err := bootstrap.NewDB(ctx, bootstrap.Config{
		DatabaseDSN: dsn,
		DatabaseSet: true,
	})
	if err != nil {
		t.Fatalf("connect to database failed: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get database handle failed: %v", err)
	}
	defer sqlDB.Close()

	// Ensure database has correct baseline schema
	_ = db.Exec("ALTER TABLE after_sale_item MODIFY sku_text VARCHAR(512) NOT NULL DEFAULT ''")
	if err := bootstrap.RunInit(ctx, db); err != nil {
		t.Fatalf("RunInit failed: %v", err)
	}

	baselineContent, err := fs.ReadFile(data.FS, "migrations/001_baseline.sql")
	if err != nil {
		t.Fatalf("read embedded 001_baseline.sql: %v", err)
	}
	seedContent, err := fs.ReadFile(data.FS, "seeds/001_default_data.sql")
	if err != nil {
		t.Fatalf("read embedded 001_default_data.sql: %v", err)
	}

	fakeFS := fstest.MapFS{
		"migrations/001_baseline.sql": &fstest.MapFile{Data: baselineContent},
		"migrations/002_fake.sql":     &fstest.MapFile{Data: []byte("-- fake future migration")},
		"seeds/001_default_data.sql":  &fstest.MapFile{Data: seedContent},
	}

	if err := bootstrap.AlignBaselineFS(ctx, db, fakeFS); err != nil {
		t.Fatalf("AlignBaselineFS failed: %v", err)
	}

	var fakeCount int64
	if err := db.Raw("SELECT COUNT(*) FROM schema_migrations WHERE version = '002_fake.sql'").Scan(&fakeCount).Error; err != nil {
		t.Fatalf("check 002_fake.sql in schema_migrations: %v", err)
	}
	if fakeCount != 0 {
		t.Fatalf("AlignBaselineFS marked 002_fake.sql as applied; expected 0 records, got %d", fakeCount)
	}

	var baselineCount int64
	if err := db.Raw("SELECT COUNT(*) FROM schema_migrations WHERE version = '001_baseline.sql'").Scan(&baselineCount).Error; err != nil {
		t.Fatalf("check 001_baseline.sql in schema_migrations: %v", err)
	}
	if baselineCount != 1 {
		t.Fatalf("expected 1 record for 001_baseline.sql, got %d", baselineCount)
	}
}

// TestCreateAdmin_Integration verifies that:
// - create-admin 成功创建超级管理员
// - super_admin 角色与 Casbin 权限正确绑定
// - 重复 username 返回错误
func TestCreateAdmin_Integration(t *testing.T) {
	dsn, err := bootstrap.ValidateTestDSN(os.Getenv("TEST_DATABASE_DSN"), os.Getenv("DATABASE_DSN"))
	if errors.Is(err, bootstrap.ErrTestDSNEmpty) {
		t.Skip("TEST_DATABASE_DSN is not configured, skipping MySQL test")
	}
	if err != nil {
		t.Fatalf("test database configuration error: %v", err)
	}

	ctx := context.Background()
	db, err := bootstrap.NewDB(ctx, bootstrap.Config{
		DatabaseDSN: dsn,
		DatabaseSet: true,
	})
	if err != nil {
		t.Fatalf("connect to database failed: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get database handle failed: %v", err)
	}
	defer sqlDB.Close()

	if err := bootstrap.RunInit(ctx, db); err != nil {
		t.Fatalf("RunInit failed: %v", err)
	}

	enforcer, err := bootstrap.NewCasbinEnforcer(db)
	if err != nil {
		t.Fatalf("NewCasbinEnforcer failed: %v", err)
	}
	jwtSecret := "test-secret-key-12345"
	authorizer := auth.NewService(db, enforcer, jwtSecret, 24*time.Hour)

	testUsername := fmt.Sprintf("test_admin_%d", time.Now().UnixNano())
	testPassword := "TestPass@123"

	// 1. Create super admin
	adminUser, err := bootstrap.CreateAdmin(ctx, db, authorizer, testUsername, testPassword, "测试管理员")
	if err != nil {
		t.Fatalf("CreateAdmin failed: %v", err)
	}
	if adminUser.ID == 0 || adminUser.Username != testUsername {
		t.Fatalf("unexpected admin user: %+v", adminUser)
	}

	// 2. Verify role assignment: user has super_admin
	roles, err := authorizer.UserRoles(ctx, adminUser.ID)
	if err != nil {
		t.Fatalf("UserRoles failed: %v", err)
	}
	hasSuperAdmin := false
	for _, r := range roles {
		if r.Code == auth.SuperAdminRoleCode {
			hasSuperAdmin = true
			break
		}
	}
	if !hasSuperAdmin {
		t.Fatalf("expected user to have super_admin role, got roles: %+v", roles)
	}

	// 3. Verify permissions: super_admin has all permissions
	allowed, err := authorizer.Can(ctx, adminUser.ID, "system:user:list")
	if err != nil || !allowed {
		t.Fatalf("expected super_admin to be permitted, got allowed=%v, err=%v", allowed, err)
	}

	// 4. Duplicate username must fail
	_, err = bootstrap.CreateAdmin(ctx, db, authorizer, testUsername, "AnotherPass123", "重复管理员")
	if err == nil {
		t.Fatalf("expected duplicate username to fail, but it succeeded")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected error to contain 'already exists', got %v", err)
	}
}

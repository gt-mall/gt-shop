package system_test

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"gorm.io/gorm"

	"mall/admin_core/bootstrap"
	"mall/admin_core/data"
	"mall/admin_core/internal/auth"
	"mall/admin_core/internal/model"
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
		"002_product_dict.sql",
		"003_product_review.sql",
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

// TestValidateAlignableMigrations verifies the align gate without a database:
// - migrations directory containing only 001_baseline.sql is alignable
// - any 002/003 post-baseline migration rejects align with an explicit file list
// - a missing baseline migration is rejected as well
func TestValidateAlignableMigrations(t *testing.T) {
	baselineContent, err := fs.ReadFile(data.FS, "migrations/001_baseline.sql")
	if err != nil {
		t.Fatalf("read embedded 001_baseline.sql: %v", err)
	}

	baselineOnly := fstest.MapFS{
		"migrations/001_baseline.sql": &fstest.MapFile{Data: baselineContent},
	}
	if err := bootstrap.ValidateAlignableMigrations(baselineOnly); err != nil {
		t.Fatalf("expected baseline-only migrations to be alignable, got %v", err)
	}

	withPostBaseline := fstest.MapFS{
		"migrations/001_baseline.sql": &fstest.MapFile{Data: baselineContent},
		"migrations/002_review.sql":   &fstest.MapFile{Data: []byte("-- review")},
		"migrations/003_coupon.sql":   &fstest.MapFile{Data: []byte("-- coupon")},
	}
	err = bootstrap.ValidateAlignableMigrations(withPostBaseline)
	if !errors.Is(err, bootstrap.ErrPostBaselineMigrationsExist) {
		t.Fatalf("expected ErrPostBaselineMigrationsExist, got %v", err)
	}
	for _, expected := range []string{
		"align is only allowed before post-baseline migrations exist",
		"002_review.sql",
		"003_coupon.sql",
	} {
		if !strings.Contains(err.Error(), expected) {
			t.Fatalf("expected error to contain %q, got: %v", expected, err)
		}
	}

	missingBaseline := fstest.MapFS{
		"migrations/002_review.sql": &fstest.MapFile{Data: []byte("-- review")},
	}
	if err := bootstrap.ValidateAlignableMigrations(missingBaseline); err == nil || errors.Is(err, bootstrap.ErrPostBaselineMigrationsExist) {
		t.Fatalf("expected missing baseline migration error, got %v", err)
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

// TestAlignBaseline_RejectsPostBaselineMigrations verifies that once the migrations
// directory contains any post-baseline migration (e.g. 002_fake.sql):
// - align is rejected even when the database schema is completely correct
// - schema_migrations records are left completely unchanged (no DELETE/reset)
// - seeds are not executed
func TestAlignBaseline_RejectsPostBaselineMigrations(t *testing.T) {
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

	// Simulate a post-baseline migration record that must never be deleted by align
	if err := db.Exec("INSERT INTO schema_migrations(version) VALUES ('002_existing.sql')").Error; err != nil {
		t.Fatalf("seed future migration record: %v", err)
	}
	defer func() {
		_ = db.Exec("DELETE FROM schema_migrations WHERE version = '002_existing.sql'").Error
	}()

	before := readMigrationVersions(t, db)

	// Marker table: only created if seeds run, proving align did not reach the seed step
	if err := db.Exec("DROP TABLE IF EXISTS align_seed_marker").Error; err != nil {
		t.Fatalf("drop seed marker table: %v", err)
	}
	defer func() {
		_ = db.Exec("DROP TABLE IF EXISTS align_seed_marker").Error
	}()

	baselineContent, err := fs.ReadFile(data.FS, "migrations/001_baseline.sql")
	if err != nil {
		t.Fatalf("read embedded 001_baseline.sql: %v", err)
	}

	fakeFS := fstest.MapFS{
		"migrations/001_baseline.sql": &fstest.MapFile{Data: baselineContent},
		"migrations/002_fake.sql":     &fstest.MapFile{Data: []byte("-- fake future migration")},
		"seeds/001_default_data.sql":  &fstest.MapFile{Data: []byte("CREATE TABLE align_seed_marker (id INT PRIMARY KEY);")},
	}

	alignErr := bootstrap.AlignBaselineFS(ctx, db, fakeFS)
	if alignErr == nil {
		t.Fatal("expected AlignBaselineFS to reject when 002_fake.sql exists, but it succeeded")
	}
	if !errors.Is(alignErr, bootstrap.ErrPostBaselineMigrationsExist) {
		t.Fatalf("expected ErrPostBaselineMigrationsExist, got: %v", alignErr)
	}
	if !strings.Contains(alignErr.Error(), "align is only allowed before post-baseline migrations exist") {
		t.Fatalf("unexpected align error message: %v", alignErr)
	}
	if !strings.Contains(alignErr.Error(), "002_fake.sql") {
		t.Fatalf("expected align error to name the offending migration file, got: %v", alignErr)
	}

	after := readMigrationVersions(t, db)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("schema_migrations must not change when align is rejected: before=%v, after=%v", before, after)
	}
	if !containsString(after, "002_existing.sql") {
		t.Fatalf("align deleted a post-baseline migration record: %v", after)
	}

	var markerCount int64
	if err := db.Raw("SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'align_seed_marker'").Scan(&markerCount).Error; err != nil {
		t.Fatalf("check seed marker table: %v", err)
	}
	if markerCount != 0 {
		t.Fatal("seeds were executed even though align was rejected")
	}
}

// TestAlignBaseline_AllowsBaselineOnlyFS verifies align still works when the migrations
// directory contains only 001_baseline.sql: it verifies the schema, resets
// schema_migrations to the baseline record and applies seeds.
func TestAlignBaseline_AllowsBaselineOnlyFS(t *testing.T) {
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

	_ = db.Exec("ALTER TABLE after_sale_item MODIFY sku_text VARCHAR(512) NOT NULL DEFAULT ''")
	if err := bootstrap.RunInit(ctx, db); err != nil {
		t.Fatalf("RunInit failed: %v", err)
	}

	// A stale record that align is allowed to reset while no code-level post-baseline
	// migration exists yet.
	if err := db.Exec("INSERT INTO schema_migrations(version) VALUES ('002_stale_test.sql')").Error; err != nil {
		t.Fatalf("seed stale migration record: %v", err)
	}
	defer func() {
		_ = db.Exec("DELETE FROM schema_migrations WHERE version = '002_stale_test.sql'").Error
	}()

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
		"seeds/001_default_data.sql":  &fstest.MapFile{Data: seedContent},
	}

	if err := bootstrap.AlignBaselineFS(ctx, db, fakeFS); err != nil {
		t.Fatalf("AlignBaselineFS failed for baseline-only migrations: %v", err)
	}

	versions := readMigrationVersions(t, db)
	if !reflect.DeepEqual(versions, []string{bootstrap.BaselineVersion}) {
		t.Fatalf("expected schema_migrations to contain only %s, got %v", bootstrap.BaselineVersion, versions)
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

	// 4. Duplicate username must fail without touching the existing user
	var passwordHashBefore string
	if err := db.Raw("SELECT password_hash FROM system_user WHERE id = ?", adminUser.ID).Scan(&passwordHashBefore).Error; err != nil {
		t.Fatalf("read password hash before duplicate attempt: %v", err)
	}

	_, err = bootstrap.CreateAdmin(ctx, db, authorizer, testUsername, "AnotherPass123", "重复管理员")
	if err == nil {
		t.Fatalf("expected duplicate username to fail, but it succeeded")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected error to contain 'already exists', got %v", err)
	}

	var passwordHashAfter string
	if err := db.Raw("SELECT password_hash FROM system_user WHERE id = ?", adminUser.ID).Scan(&passwordHashAfter).Error; err != nil {
		t.Fatalf("read password hash after duplicate attempt: %v", err)
	}
	if passwordHashAfter != passwordHashBefore {
		t.Fatal("duplicate create-admin must not modify the existing user's password")
	}

	rolesAfterDuplicate, err := authorizer.UserRoles(ctx, adminUser.ID)
	if err != nil {
		t.Fatalf("UserRoles after duplicate attempt failed: %v", err)
	}
	hasSuperAdminAfterDuplicate := false
	for _, r := range rolesAfterDuplicate {
		if r.Code == auth.SuperAdminRoleCode {
			hasSuperAdminAfterDuplicate = true
			break
		}
	}
	if !hasSuperAdminAfterDuplicate {
		t.Fatalf("duplicate create-admin must not modify the existing user's roles, got: %+v", rolesAfterDuplicate)
	}
}

// TestCreateAdmin_RollsBackOnRoleAssignmentFailure verifies that when the Casbin role
// assignment fails, create-admin does not leave a role-less system_user behind:
// - CreateAdmin returns the original assign error
// - the just-created system_user is deleted, so the username is free again
// - no Casbin user:<id> role mapping remains
// - re-running create-admin with the same username succeeds afterwards
func TestCreateAdmin_RollsBackOnRoleAssignmentFailure(t *testing.T) {
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
	authorizer := auth.NewService(db, enforcer, "test-secret-key-rollback", 24*time.Hour)

	username := fmt.Sprintf("rollback_admin_%d", time.Now().UnixNano())

	// Capture the primary key of the freshly created system_user so the test can
	// assert that no Casbin mapping survives for exactly that user.
	var createdUserID int64
	captureCallback := fmt.Sprintf("test_capture_created_user_%d", time.Now().UnixNano())
	if err := db.Callback().Create().After("gorm:create").Register(captureCallback, func(tx *gorm.DB) {
		if tx.Statement.Table != "system_user" {
			return
		}
		if created, ok := tx.Statement.Dest.(*model.SystemUser); ok {
			createdUserID = created.ID
		}
	}); err != nil {
		t.Fatalf("register capture callback: %v", err)
	}
	defer func() {
		_ = db.Callback().Create().Remove(captureCallback)
	}()

	// Inject a failure on the Casbin rule write only. system_user writes are unaffected,
	// so CreateAdmin fails exactly at the super_admin binding step.
	injectFailureCallback := fmt.Sprintf("test_fail_casbin_create_%d", time.Now().UnixNano())
	if err := db.Callback().Create().Before("gorm:create").Register(injectFailureCallback, func(tx *gorm.DB) {
		if tx.Statement.Table == "casbin_rule" {
			tx.AddError(errors.New("injected casbin create failure"))
		}
	}); err != nil {
		t.Fatalf("register failure callback: %v", err)
	}
	defer func() {
		_ = db.Callback().Create().Remove(injectFailureCallback)
	}()

	_, err = bootstrap.CreateAdmin(ctx, db, authorizer, username, "TestPass@123", "回滚管理员")
	if err == nil {
		t.Fatal("expected CreateAdmin to fail when Casbin role assignment fails")
	}
	if !strings.Contains(err.Error(), "assign super_admin role") {
		t.Fatalf("expected error to contain 'assign super_admin role', got: %v", err)
	}
	if createdUserID == 0 {
		t.Fatal("capture callback did not observe the created system_user")
	}

	var userCount int64
	if err := db.Model(&model.SystemUser{}).Where("username = ?", username).Count(&userCount).Error; err != nil {
		t.Fatalf("count rolled back user: %v", err)
	}
	if userCount != 0 {
		t.Fatalf("expected rolled back system_user %q to be deleted, found %d record(s)", username, userCount)
	}

	var policyCount int64
	if err := db.Raw("SELECT COUNT(*) FROM casbin_rule WHERE v0 = ?", auth.UserSubject(createdUserID)).Scan(&policyCount).Error; err != nil {
		t.Fatalf("check residual casbin mapping: %v", err)
	}
	if policyCount != 0 {
		t.Fatalf("expected no residual casbin mapping for %s, found %d", auth.UserSubject(createdUserID), policyCount)
	}

	// Recovery: remove the injected failure and retry with the same username.
	if err := db.Callback().Create().Remove(injectFailureCallback); err != nil {
		t.Fatalf("remove failure callback: %v", err)
	}
	retried, err := bootstrap.CreateAdmin(ctx, db, authorizer, username, "TestPass@123", "回滚管理员")
	if err != nil {
		t.Fatalf("retry CreateAdmin must succeed after rollback, got: %v", err)
	}
	defer func() {
		_ = authorizer.DeleteUserRoles(retried.ID)
		_ = db.Delete(&model.SystemUser{}, retried.ID).Error
	}()

	roles, err := authorizer.UserRoles(ctx, retried.ID)
	if err != nil {
		t.Fatalf("UserRoles failed: %v", err)
	}
	hasSuperAdmin := false
	for _, role := range roles {
		if role.Code == auth.SuperAdminRoleCode {
			hasSuperAdmin = true
			break
		}
	}
	if !hasSuperAdmin {
		t.Fatalf("expected retried admin to have super_admin role, got: %+v", roles)
	}
}

// readMigrationVersions returns all schema_migrations versions in stable order.
func readMigrationVersions(t *testing.T, db *gorm.DB) []string {
	t.Helper()
	var versions []string
	if err := db.Raw("SELECT version FROM schema_migrations ORDER BY version").Scan(&versions).Error; err != nil {
		t.Fatalf("read schema_migrations versions: %v", err)
	}
	return versions
}

// containsString reports whether values contains target.
func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

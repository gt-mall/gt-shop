package system_test

import (
	"context"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"mall/admin_core/bootstrap"
	"mall/admin_core/data"
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
func TestMigrationRunFromEmptyCwd(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("TEST_DATABASE_DSN"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("DATABASE_DSN"))
	}
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN / DATABASE_DSN is not configured, skipping MySQL migration test")
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
// - AlignBaseline resets schema_migrations to baseline and applies seeds idempotently
func TestRunSeedsAndAlignBaseline(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("TEST_DATABASE_DSN"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("DATABASE_DSN"))
	}
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN / DATABASE_DSN is not configured, skipping MySQL test")
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

package bootstrap

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"gorm.io/gorm"

	"mall/admin_core/data"
)

// RunMigrations applies each not-yet-applied SQL file in lexical order from embedded data.FS.
func RunMigrations(ctx context.Context, db *gorm.DB) error {
	return RunMigrationsFS(ctx, db, data.FS)
}

// RunMigrationsFS applies each not-yet-applied SQL file in lexical order from the provided fs.FS.
func RunMigrationsFS(ctx context.Context, db *gorm.DB, fsys fs.FS) error {
	if db == nil {
		return nil
	}

	db = db.WithContext(ctx)
	migrationsTableSQL := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`
	if err := db.Exec(migrationsTableSQL).Error; err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := fs.ReadDir(fsys, "migrations")
	if err != nil {
		return fmt.Errorf("read migrations directory: %w", err)
	}
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".sql" {
			files = append(files, entry.Name())
		}
	}
	sort.Strings(files)

	for _, fileName := range files {
		var applied int64
		if err := db.Raw(`SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, fileName).Scan(&applied).Error; err != nil {
			return fmt.Errorf("check migration %s: %w", fileName, err)
		}
		if applied > 0 {
			continue
		}

		log.Printf("applying migration: %s", fileName)
		sql, err := fs.ReadFile(fsys, path.Join("migrations", fileName))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", fileName, err)
		}
		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec(string(sql)).Error; err != nil {
				return fmt.Errorf("apply migration %s: %w", fileName, err)
			}
			if err := tx.Exec(`INSERT INTO schema_migrations(version) VALUES (?)`, fileName).Error; err != nil {
				return fmt.Errorf("record migration %s: %w", fileName, err)
			}
			return nil
		}); err != nil {
			return fmt.Errorf("run migration %s: %w", fileName, err)
		}
		log.Printf("applied migration: %s", fileName)
	}

	return nil
}

// RunInit applies all migrations and seeds in order.
// Typically used during clean installation or dev database initialization.
func RunInit(ctx context.Context, db *gorm.DB) error {
	return RunInitFS(ctx, db, data.FS)
}

// RunInitFS applies migrations and seeds from the provided fs.FS.
func RunInitFS(ctx context.Context, db *gorm.DB, fsys fs.FS) error {
	if err := RunMigrationsFS(ctx, db, fsys); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	if err := RunSeedsFS(ctx, db, fsys); err != nil {
		return fmt.Errorf("run seeds: %w", err)
	}
	return nil
}

const BaselineVersion = "001_baseline.sql"

// BaselineTables defines the 26 core business tables required by V1 Baseline.
var BaselineTables = []string{
	"system_user",
	"system_role",
	"casbin_rule",
	"system_file_group",
	"system_file",
	"system_config",
	"product_category",
	"freight_template",
	"product",
	"product_sku",
	"product_sku_inventory",
	"product_sku_inventory_log",
	"member",
	"member_auth",
	"member_account",
	"member_address",
	"cart_item",
	"order_main",
	"order_item",
	"order_address",
	"order_log",
	"order_payment",
	"after_sale",
	"after_sale_item",
	"after_sale_log",
	"order_refund",
}

// VerifyBaselineSchema checks whether the current database schema satisfies all V1 Baseline requirements:
// 1. All 26 core business tables exist.
// 2. Key column definitions and types match.
// 3. Key UNIQUE constraints exist.
// 4. Inventory log CHECK constraint permits operator_id = 0.
func VerifyBaselineSchema(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database handle is nil")
	}
	db = db.WithContext(ctx)

	// 1. Verify that all 26 core tables exist
	var existingTables []string
	if err := db.Raw(`
		SELECT TABLE_NAME
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_TYPE = 'BASE TABLE'
	`).Scan(&existingTables).Error; err != nil {
		return fmt.Errorf("verify tables failed: %w", err)
	}

	tableSet := make(map[string]bool, len(existingTables))
	for _, tbl := range existingTables {
		tableSet[tbl] = true
	}
	for _, requiredTable := range BaselineTables {
		if !tableSet[requiredTable] {
			return fmt.Errorf("Baseline schema verification failed:\nmissing required table '%s'", requiredTable)
		}
	}

	// 2. Verify key columns and data types
	type columnMeta struct {
		TableName              string `gorm:"column:TABLE_NAME"`
		ColumnName             string `gorm:"column:COLUMN_NAME"`
		DataType               string `gorm:"column:DATA_TYPE"`
		ColumnType             string `gorm:"column:COLUMN_TYPE"`
		CharacterMaximumLength *int64 `gorm:"column:CHARACTER_MAXIMUM_LENGTH"`
	}
	var columns []columnMeta
	if err := db.Raw(`
		SELECT TABLE_NAME, COLUMN_NAME, DATA_TYPE, COLUMN_TYPE, CHARACTER_MAXIMUM_LENGTH
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE()
	`).Scan(&columns).Error; err != nil {
		return fmt.Errorf("verify columns failed: %w", err)
	}

	colMap := make(map[string]columnMeta, len(columns))
	for _, col := range columns {
		colMap[col.TableName+"."+col.ColumnName] = col
	}

	checkVarchar := func(tblCol string, expectedLen int64) error {
		col, ok := colMap[tblCol]
		if !ok {
			return fmt.Errorf("Baseline schema verification failed:\n%s does not exist", tblCol)
		}
		if strings.ToLower(col.DataType) != "varchar" || col.CharacterMaximumLength == nil || *col.CharacterMaximumLength != expectedLen {
			actual := strings.ToUpper(col.ColumnType)
			return fmt.Errorf("Baseline schema verification failed:\n%s expected VARCHAR(%d), got %s", tblCol, expectedLen, actual)
		}
		return nil
	}

	checkInt := func(tblCol string, expectedDataType string, expectedUnsigned bool, displayExpected string) error {
		col, ok := colMap[tblCol]
		if !ok {
			return fmt.Errorf("Baseline schema verification failed:\n%s does not exist", tblCol)
		}
		dt := strings.ToLower(col.DataType)
		isUnsigned := strings.Contains(strings.ToLower(col.ColumnType), "unsigned")
		if dt != strings.ToLower(expectedDataType) || isUnsigned != expectedUnsigned {
			actual := strings.ToUpper(col.ColumnType)
			return fmt.Errorf("Baseline schema verification failed:\n%s expected %s, got %s", tblCol, displayExpected, actual)
		}
		return nil
	}

	if err := checkVarchar("after_sale_item.sku_text", 512); err != nil {
		return err
	}
	if err := checkVarchar("after_sale_item.sku_code", 255); err != nil {
		return err
	}
	if err := checkVarchar("order_item.sku_code", 255); err != nil {
		return err
	}
	if err := checkVarchar("system_file.file_type", 32); err != nil {
		return err
	}
	if err := checkVarchar("system_file.scene", 64); err != nil {
		return err
	}
	if err := checkVarchar("system_file.owner_type", 32); err != nil {
		return err
	}
	if err := checkInt("system_file.owner_id", "bigint", true, "BIGINT UNSIGNED"); err != nil {
		return err
	}
	if err := checkInt("system_file.visibility", "tinyint", true, "TINYINT UNSIGNED"); err != nil {
		return err
	}
	if err := checkInt("product.sales_count", "bigint", false, "BIGINT"); err != nil {
		return err
	}
	if err := checkInt("product.virtual_sales", "int", false, "INT"); err != nil {
		return err
	}
	if err := checkVarchar("order_main.request_id", 64); err != nil {
		return err
	}
	if err := checkVarchar("after_sale.request_id", 64); err != nil {
		return err
	}

	// 3. Verify key UNIQUE constraints
	type statMeta struct {
		TableName  string `gorm:"column:TABLE_NAME"`
		IndexName  string `gorm:"column:INDEX_NAME"`
		ColumnName string `gorm:"column:COLUMN_NAME"`
		SeqInIndex int    `gorm:"column:SEQ_IN_INDEX"`
	}
	var stats []statMeta
	if err := db.Raw(`
		SELECT TABLE_NAME, INDEX_NAME, COLUMN_NAME, SEQ_IN_INDEX
		FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = DATABASE() AND NON_UNIQUE = 0
		ORDER BY TABLE_NAME, INDEX_NAME, SEQ_IN_INDEX
	`).Scan(&stats).Error; err != nil {
		return fmt.Errorf("verify unique constraints failed: %w", err)
	}

	tableIndexes := make(map[string]map[string][]string)
	for _, s := range stats {
		if tableIndexes[s.TableName] == nil {
			tableIndexes[s.TableName] = make(map[string][]string)
		}
		tableIndexes[s.TableName][s.IndexName] = append(tableIndexes[s.TableName][s.IndexName], s.ColumnName)
	}

	checkUnique := func(tbl string, expectedCols []string) error {
		indexes, ok := tableIndexes[tbl]
		if !ok {
			return fmt.Errorf("Baseline schema verification failed:\n%s has no unique constraints", tbl)
		}
		for _, cols := range indexes {
			if len(cols) == len(expectedCols) {
				match := true
				for i := range cols {
					if !strings.EqualFold(cols[i], expectedCols[i]) {
						match = false
						break
					}
				}
				if match {
					return nil
				}
			}
		}
		return fmt.Errorf("Baseline schema verification failed:\n%s expected UNIQUE constraint on (%s), but not found", tbl, strings.Join(expectedCols, ", "))
	}

	if err := checkUnique("order_main", []string{"member_id", "request_id"}); err != nil {
		return err
	}
	if err := checkUnique("after_sale", []string{"member_id", "request_id"}); err != nil {
		return err
	}
	if err := checkUnique("order_main", []string{"order_no"}); err != nil {
		return err
	}
	if err := checkUnique("after_sale", []string{"after_sale_no"}); err != nil {
		return err
	}
	if err := checkUnique("order_payment", []string{"payment_no"}); err != nil {
		return err
	}
	if err := checkUnique("order_refund", []string{"refund_no"}); err != nil {
		return err
	}
	if err := checkUnique("cart_item", []string{"member_id", "sku_id"}); err != nil {
		return err
	}

	// 4. Verify inventory log CHECK constraint: operator_id must allow 0 (operator_id >= 0)
	type checkMeta struct {
		ConstraintName string `gorm:"column:CONSTRAINT_NAME"`
		CheckClause    string `gorm:"column:CHECK_CLAUSE"`
	}
	var checks []checkMeta
	if err := db.Raw(`
		SELECT tc.CONSTRAINT_NAME, cc.CHECK_CLAUSE
		FROM information_schema.TABLE_CONSTRAINTS tc
		JOIN information_schema.CHECK_CONSTRAINTS cc
		  ON tc.CONSTRAINT_SCHEMA = cc.CONSTRAINT_SCHEMA
		 AND tc.CONSTRAINT_NAME = cc.CONSTRAINT_NAME
		WHERE tc.CONSTRAINT_SCHEMA = DATABASE()
		  AND tc.TABLE_NAME = 'product_sku_inventory_log'
		  AND tc.CONSTRAINT_TYPE = 'CHECK'
	`).Scan(&checks).Error; err != nil {
		return fmt.Errorf("verify check constraints failed: %w", err)
	}

	foundOperatorCheck := false
	operatorCheckValid := false
	for _, chk := range checks {
		norm := strings.ToLower(chk.CheckClause)
		norm = strings.ReplaceAll(norm, "`", "")
		if strings.Contains(norm, "operator_id") {
			foundOperatorCheck = true
			if strings.Contains(norm, "operator_id >= 0") || strings.Contains(norm, "operator_id>=0") {
				operatorCheckValid = true
				break
			}
		}
	}
	if !foundOperatorCheck || !operatorCheckValid {
		return fmt.Errorf("Baseline schema verification failed:\nproduct_sku_inventory_log CHECK constraint must allow operator_id = 0 (operator_id >= 0)")
	}

	return nil
}

// AlignBaseline marks only 001_baseline.sql as applied in schema_migrations
// after verifying that the current database satisfies all V1 Baseline schema requirements,
// then applies seeds. It will not alter existing business schema.
func AlignBaseline(ctx context.Context, db *gorm.DB) error {
	return AlignBaselineFS(ctx, db, data.FS)
}

// AlignBaselineFS verifies the schema and marks only 001_baseline.sql as applied.
// It will never mark future migrations (such as 002/003) as applied.
func AlignBaselineFS(ctx context.Context, db *gorm.DB, fsys fs.FS) error {
	if db == nil {
		return nil
	}

	db = db.WithContext(ctx)

	// Ensure 001_baseline.sql exists in fsys
	if _, err := fs.Stat(fsys, path.Join("migrations", BaselineVersion)); err != nil {
		return fmt.Errorf("baseline migration %s not found in migrations directory: %w", BaselineVersion, err)
	}

	// Step 1: Verify current schema matches V1 Baseline before modifying anything
	if err := VerifyBaselineSchema(ctx, db); err != nil {
		return err
	}

	// Step 2: Ensure schema_migrations table exists
	migrationsTableSQL := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`
	if err := db.Exec(migrationsTableSQL).Error; err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	// Step 3: Transactionally reset schema_migrations to ONLY 001_baseline.sql
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`DELETE FROM schema_migrations`).Error; err != nil {
			return fmt.Errorf("reset schema_migrations: %w", err)
		}
		if err := tx.Exec(`INSERT INTO schema_migrations(version) VALUES (?)`, BaselineVersion).Error; err != nil {
			return fmt.Errorf("record baseline %s: %w", BaselineVersion, err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("update schema_migrations: %w", err)
	}
	log.Printf("marked baseline migration as applied: %s", BaselineVersion)

	// Step 4: Apply seeds idempotently
	if err := RunSeedsFS(ctx, db, fsys); err != nil {
		return fmt.Errorf("run seeds: %w", err)
	}
	return nil
}

// RunSeeds applies each SQL file in lexical order from embedded data.FS under seeds/.
// Seed execution is intended for initial setup or explicit seeding, not automated on every server start.
func RunSeeds(ctx context.Context, db *gorm.DB) error {
	return RunSeedsFS(ctx, db, data.FS)
}

// RunSeedsFS applies each SQL file in lexical order from the provided fs.FS under seeds/.
func RunSeedsFS(ctx context.Context, db *gorm.DB, fsys fs.FS) error {
	if db == nil {
		return nil
	}

	db = db.WithContext(ctx)
	entries, err := fs.ReadDir(fsys, "seeds")
	if err != nil {
		return fmt.Errorf("read seeds directory: %w", err)
	}
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".sql" {
			files = append(files, entry.Name())
		}
	}
	sort.Strings(files)

	for _, fileName := range files {
		log.Printf("applying seed: %s", fileName)
		sql, err := fs.ReadFile(fsys, path.Join("seeds", fileName))
		if err != nil {
			return fmt.Errorf("read seed %s: %w", fileName, err)
		}
		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec(string(sql)).Error; err != nil {
				return fmt.Errorf("apply seed %s: %w", fileName, err)
			}
			return nil
		}); err != nil {
			return fmt.Errorf("run seed %s: %w", fileName, err)
		}
		log.Printf("applied seed: %s", fileName)
	}

	return nil
}

package product_test

import (
	"context"
	"os"
	"testing"

	"mall/admin_core/bootstrap"
	"mall/admin_core/internal/model"
	productcategory "mall/admin_core/internal/product/category"
	productproduct "mall/admin_core/internal/product/product"
)

func TestProductMySQLIntegration(t *testing.T) {
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		dsn = os.Getenv("TEST_DATABASE_DSN")
	}
	if dsn == "" {
		t.Skip("DATABASE_DSN or TEST_DATABASE_DSN is not set; skipping MySQL integration test")
	}

	db, err := bootstrap.NewDB(context.Background(), bootstrap.Config{DatabaseDSN: dsn, DatabaseSet: true})
	if err != nil {
		t.Skipf("cannot connect to MySQL: %v; skipping MySQL integration test", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Skipf("cannot get sqlDB: %v", err)
	}
	defer sqlDB.Close()

	// Use an isolated transaction that always rolls back to protect business data
	tx := db.Begin()
	defer tx.Rollback()

	catSvc := productcategory.NewService(tx)
	prodSvc := productproduct.NewService(tx)
	ctx := context.Background()

	// 1. Test Category creation in MySQL
	cat, err := catSvc.Create(ctx, productcategory.CreateRequest{
		Name:   "MySQL集成测试分类",
		Status: func(v int8) *int8 { return &v }(1),
	})
	if err != nil {
		t.Fatalf("MySQL category create: %v", err)
	}

	// 2. Test Product & SKU creation with DECIMAL and JSON
	skuCode := "MYSQL-SKU-TEST-001"
	prodRes, err := prodSvc.Create(ctx, productproduct.ProductCreateRequest{
		Name:       "MySQL集成测试商品",
		Subtitle:   "测试副标题",
		CategoryID: cat.ID,
		Status:     func(v int8) *int8 { return &v }(0),
		Specs: []productproduct.ProductSpec{
			{
				ID:   1,
				Name: "版本",
				Values: []productproduct.ProductSpecValue{
					{ID: 1, Name: "标准版"},
					{ID: 2, Name: "豪华版"},
				},
			},
		},
		SKUs: []productproduct.ProductSKURequest{
			{
				SpecValues: []productproduct.ProductSKUValue{{SpecID: 1, ValueID: 1}},
				SKUCode:    &skuCode,
				Price:      "199.90",
			},
			{
				SpecValues: []productproduct.ProductSKUValue{{SpecID: 1, ValueID: 2}},
				Price:      "299.90",
			},
		},
	})
	if err != nil {
		t.Fatalf("MySQL product create: %v", err)
	}

	// 3. Test Detail retrieval (JSON parse and DECIMAL reading)
	detail, err := prodSvc.Detail(ctx, prodRes.ID)
	if err != nil {
		t.Fatalf("MySQL product detail: %v", err)
	}
	if len(detail.SKUs) != 2 {
		t.Fatalf("MySQL detail SKUs len = %d, want 2", len(detail.SKUs))
	}
	if detail.SKUs[0].Price != "199.90" || detail.SKUs[1].Price != "299.90" {
		t.Fatalf("MySQL detail prices unexpected: %s, %s", detail.SKUs[0].Price, detail.SKUs[1].Price)
	}

	// 4. Test Global unique SKU Code in MySQL
	dupCode := "MYSQL-SKU-TEST-001"
	_, err = prodSvc.Create(ctx, productproduct.ProductCreateRequest{
		Name: "MySQL冲突商品",
		SKUs: []productproduct.ProductSKURequest{
			{Price: "99.00", SKUCode: &dupCode},
		},
	})
	if err == nil {
		t.Fatal("MySQL duplicate sku_code must fail")
	}

	// 5. Test List aggregation in MySQL (MIN/MAX price and SKU count)
	listRows, total, err := prodSvc.List(ctx, productproduct.ProductListRequest{
		Keyword: "MySQL集成测试",
	})
	if err != nil {
		t.Fatalf("MySQL product list: %v", err)
	}
	if total < 1 || len(listRows) < 1 {
		t.Fatalf("MySQL list total = %d, rows = %d", total, len(listRows))
	}

	// 6. Test Soft Delete in MySQL
	if err := prodSvc.Delete(ctx, prodRes.ID); err != nil {
		t.Fatalf("MySQL product delete: %v", err)
	}
	var deletedProd model.Product
	if err := tx.Where("id = ?", prodRes.ID).First(&deletedProd).Error; err == nil {
		t.Fatal("soft deleted product should not be found with default scope")
	}
}

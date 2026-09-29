package tab_counts_test

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"mall/admin_core/internal/model"
	"mall/admin_core/internal/product/product"
)

func TestTabCountsInventoryBoundaries(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&model.Product{}, &model.ProductSKU{}, &model.ProductSkuInventory{}); err != nil {
		t.Fatal(err)
	}

	products := []model.Product{
		{ID: 1, Name: "预警关闭且无库存", Status: 1, FreightFee: "0.00"},
		{ID: 2, Name: "停用SKU有库存", Status: 1, FreightFee: "0.00"},
		{ID: 3, Name: "仓库商品无库存", Status: 0, FreightFee: "0.00"},
		{ID: 4, Name: "低库存预警", Status: 1, FreightFee: "0.00"},
	}
	for i := range products {
		if err := db.Create(&products[i]).Error; err != nil {
			t.Fatal(err)
		}
	}

	skus := []model.ProductSKU{
		{ID: 11, ProductID: 1, SpecValues: "[]", SpecHash: "sku-11", Price: "10.00", Status: 1},
		{ID: 21, ProductID: 2, SpecValues: "[]", SpecHash: "sku-21", Price: "10.00", Status: 1},
		{ID: 22, ProductID: 2, SpecValues: "[]", SpecHash: "sku-22", Price: "10.00", Status: 0},
		{ID: 31, ProductID: 3, SpecValues: "[]", SpecHash: "sku-31", Price: "10.00", Status: 1},
		{ID: 41, ProductID: 4, SpecValues: "[]", SpecHash: "sku-41", Price: "10.00", Status: 1},
	}
	for i := range skus {
		if err := db.Create(&skus[i]).Error; err != nil {
			t.Fatal(err)
		}
	}

	inventory := []model.ProductSkuInventory{
		{ProductID: 1, SKUID: 11, Stock: 0, WarningStock: 0},
		{ProductID: 2, SKUID: 21, Stock: 0, WarningStock: 0},
		{ProductID: 2, SKUID: 22, Stock: 5, WarningStock: 0},
		{ProductID: 3, SKUID: 31, Stock: 0, WarningStock: 0},
		{ProductID: 4, SKUID: 41, Stock: 5, WarningStock: 10},
	}
	for i := range inventory {
		if err := db.Create(&inventory[i]).Error; err != nil {
			t.Fatal(err)
		}
	}

	ctx := context.Background()
	svc := product.NewService(db)
	counts, err := svc.TabCounts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if counts.All != 4 || counts.Selling != 3 || counts.Warehouse != 1 || counts.Warning != 1 || counts.SoldOut != 2 {
		t.Fatalf("unexpected tab counts: %+v", counts)
	}

	for _, tc := range []struct {
		tab  string
		want []int64
	}{
		{tab: "warning", want: []int64{4}},
		{tab: "sold_out", want: []int64{2, 1}},
	} {
		rows, total, err := svc.List(ctx, product.ProductListRequest{Tab: tc.tab})
		if err != nil {
			t.Fatalf("list %s: %v", tc.tab, err)
		}
		if total != int64(len(tc.want)) || len(rows) != len(tc.want) {
			t.Fatalf("list %s: total=%d rows=%d", tc.tab, total, len(rows))
		}
		for i, row := range rows {
			if row.ID != tc.want[i] {
				t.Fatalf("list %s row %d: got %d, want %d", tc.tab, i, row.ID, tc.want[i])
			}
		}
	}

	if err := db.Migrator().DropTable(&model.ProductSkuInventory{}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.TabCounts(ctx); err == nil {
		t.Fatal("tab counts should return an inventory query error")
	}
	if _, _, err := svc.List(ctx, product.ProductListRequest{Tab: "warning"}); err == nil {
		t.Fatal("warning list should return an inventory query error")
	}
}

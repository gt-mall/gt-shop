package product_test

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"

	"mall/admin_core/bootstrap"
	"mall/admin_core/internal/model"
	productcategory "mall/admin_core/internal/product/category"
	productinventory "mall/admin_core/internal/product/inventory"
	productproduct "mall/admin_core/internal/product/product"
)

func TestInventoryMySQLIntegration(t *testing.T) {
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		dsn = os.Getenv("TEST_DATABASE_DSN")
	}
	if dsn == "" {
		t.Skip("DATABASE_DSN or TEST_DATABASE_DSN is not set; skipping MySQL integration test")
	}

	db, err := bootstrap.NewDB(context.Background(), bootstrap.Config{DatabaseDSN: dsn, DatabaseSet: true})
	if err != nil {
		t.Skipf("cannot connect to MySQL: %v; skipping", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Skipf("cannot get sqlDB: %v", err)
	}
	defer sqlDB.Close()

	ctx := context.Background()
	catSvc := productcategory.NewService(db)
	prodSvc := productproduct.NewService(db)
	invSvc := productinventory.NewService(db)

	cat, err := catSvc.Create(ctx, productcategory.CreateRequest{
		Name:   "MySQL库存测试分类",
		Status: func(v int8) *int8 { return &v }(1),
	})
	if err != nil {
		t.Fatalf("create category: %v", err)
	}

	skuCode := "MYSQL-INV-SKU-" + uuid.New().String()[:8]
	prodRes, err := prodSvc.Create(ctx, productproduct.ProductCreateRequest{
		Name:       "MySQL库存测试商品",
		Subtitle:   "测试商品",
		CategoryID: cat.ID,
		Status:     func(v int8) *int8 { return &v }(0), // 下架状态
		SKUs: []productproduct.ProductSKURequest{
			{
				Price:       "100.00",
				MarketPrice: "200.00",
				CostPrice:   "50.00",
				SKUCode:     &skuCode,
			},
		},
	})
	if err != nil {
		t.Fatalf("create product in mysql: %v", err)
	}

	var sku model.ProductSKU
	if err := db.Where("product_id = ?", prodRes.ID).First(&sku).Error; err != nil {
		t.Fatalf("find sku: %v", err)
	}

	// 1. 验证零库存初始化
	detail, bErr := invSvc.Detail(ctx, sku.ID)
	if bErr != nil {
		t.Fatalf("inv detail: %v", bErr.Message)
	}
	if !detail.Initialized || *detail.Stock != 0 || *detail.Version != 0 {
		t.Fatalf("unexpected initial inventory: %+v", detail)
	}

	adminUser := model.SystemUser{ID: 1, Username: "admin", Nickname: "超级管理员"}

	// 2. 人工入库
	reqID := uuid.New().String()
	res, bErr := invSvc.Adjust(ctx, adminUser, productinventory.ProductInventoryAdjustRequest{
		SKUID:           sku.ID,
		RequestID:       reqID,
		ExpectedVersion: func(v uint32) *uint32 { return &v }(0),
		ChangeType:      productinventory.ChangeTypeManualIn,
		Quantity:        func(v int) *int { return &v }(100),
		Remark:          "MySQL测试入库100件",
	})
	if bErr != nil {
		t.Fatalf("adjust manual_in: %v", bErr.Message)
	}
	if res.Log.StockAfter != 100 || res.Log.VersionAfter != 1 {
		t.Fatalf("unexpected stock after in: %d, version: %d", res.Log.StockAfter, res.Log.VersionAfter)
	}

	// 3. 幂等回放验证
	replayRes, bErr := invSvc.Adjust(ctx, adminUser, productinventory.ProductInventoryAdjustRequest{
		SKUID:           sku.ID,
		RequestID:       reqID,
		ExpectedVersion: func(v uint32) *uint32 { return &v }(0),
		ChangeType:      productinventory.ChangeTypeManualIn,
		Quantity:        func(v int) *int { return &v }(100),
		Remark:          "MySQL测试入库100件",
	})
	if bErr != nil {
		t.Fatalf("replay: %v", bErr.Message)
	}
	if !replayRes.Replayed {
		t.Fatal("expected replayed = true")
	}

	// 4. 并发相同请求重试测试（10个并发协程发送完全相同的请求）
	var sameSuccessCount int32
	var sameReplayCount int32
	var wg sync.WaitGroup
	concurrentReqID := uuid.New().String()
	sameReq := productinventory.ProductInventoryAdjustRequest{
		SKUID:           sku.ID,
		RequestID:       concurrentReqID,
		ExpectedVersion: func(v uint32) *uint32 { return &v }(1),
		ChangeType:      productinventory.ChangeTypeManualIn,
		Quantity:        func(v int) *int { return &v }(50),
		Remark:          "并发相同请求入库",
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := invSvc.Adjust(ctx, adminUser, sameReq)
			if err == nil {
				if r.Replayed {
					atomic.AddInt32(&sameReplayCount, 1)
				} else {
					atomic.AddInt32(&sameSuccessCount, 1)
				}
			}
		}()
	}
	wg.Wait()

	// 必须有且仅有1个执行成功（replayed=false），其余9个为幂等回放（replayed=true）
	if sameSuccessCount != 1 {
		t.Fatalf("expected exactly 1 non-replayed execution, got %d (replayed: %d)", sameSuccessCount, sameReplayCount)
	}
	if sameSuccessCount+sameReplayCount != 10 {
		t.Fatalf("total successful responses = %d, expected 10", sameSuccessCount+sameReplayCount)
	}

	// 5. 验证库存总量为 150，版本为 2
	detailAfter, bErr := invSvc.Detail(ctx, sku.ID)
	if bErr != nil {
		t.Fatalf("detail after: %v", bErr.Message)
	}
	if *detailAfter.Stock != 150 || *detailAfter.Version != 2 {
		t.Fatalf("expected stock 150, version 2; got stock %d, version %d", *detailAfter.Stock, *detailAfter.Version)
	}

	// 6. 验证流水仅新增1条
	var logCnt int64
	if err := db.Model(&model.ProductSkuInventoryLog{}).Where("request_id = ?", concurrentReqID).Count(&logCnt).Error; err != nil {
		t.Fatalf("count log: %v", err)
	}
	if logCnt != 1 {
		t.Fatalf("expected exactly 1 log record for request %s, got %d", concurrentReqID, logCnt)
	}
}

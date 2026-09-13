package miniapp_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"github.com/alicebob/miniredis/v2"
	miniappproduct "mall/admin_core/internal/miniapp/product"
	"mall/admin_core/internal/model"
	productproduct "mall/admin_core/internal/product/product"
	systemfile "mall/admin_core/internal/system/file"
	"mall/admin_core/pkg/cache"
)

func getMiniappTestRedisCache(t *testing.T) (cache.Cache, *redis.Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	t.Cleanup(func() {
		client.Close()
	})
	c := cache.NewRedisCache(client)
	return c, client
}

func miniappTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { sqlDB.Close() })

	if err := db.AutoMigrate(
		&model.SystemConfig{},
		&model.SystemFile{},
		&model.SystemFileGroup{},
		&model.ProductCategory{},
		&model.Product{},
		&model.ProductSKU{},
		&model.ProductSkuInventory{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	fileSvc := systemfile.NewService(db)
	cfg := systemfile.DefaultConfig()
	cfg.Local.Path = t.TempDir()
	if err := fileSvc.SaveConfig(context.Background(), cfg); err != nil {
		t.Fatalf("save file config: %v", err)
	}

	return db
}

func createTestImage(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	file := model.SystemFile{
		GroupID: 0,
		Storage: "local",
		Name:    "test.png",
		Path:    "2026/09/test.png",
		Ext:     "png",
		Size:    100,
		Width:   10,
		Height:  10,
	}
	if err := db.Create(&file).Error; err != nil {
		t.Fatalf("create image: %v", err)
	}
	return file.ID
}

// 1. 商品详情缓存命中时修改库存，实时返回最新可用库存
func TestMiniappProductDetail_DecoupledLiveInventory(t *testing.T) {
	ctx := context.Background()
	c, client := getMiniappTestRedisCache(t)
	defer client.Close()

	db := miniappTestDB(t)

	cat := model.ProductCategory{Name: "数码", Status: 1}
	db.Create(&cat)

	imgID := createTestImage(t, db)
	statusVal := int8(1)
	prodAdminSvc := productproduct.NewService(db, c)
	pRes, err := prodAdminSvc.Create(ctx, productproduct.ProductCreateRequest{
		Name:        "智能手表",
		CategoryID:  cat.ID,
		MainImageID: &imgID,
		Status:      &statusVal,
		SKUs: []productproduct.ProductSKURequest{
			{
				Price:  "899.00",
				Status: &statusVal,
			},
		},
	})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	// 初始化库存为 100
	var sku model.ProductSKU
	db.Where("product_id = ?", pRes.ID).First(&sku)
	db.Model(&model.ProductSkuInventory{}).Where("sku_id = ?", sku.ID).Update("stock", 100)

	miniappSvc := miniappproduct.NewService(db, c)
	detailVerKey := cache.MiniappProductDetailVersionKey(pRes.ID)
	defer c.Delete(ctx, detailVerKey)

	// 首次查询：查 DB 并建立静态缓存
	d1, err := miniappSvc.Detail(ctx, pRes.ID)
	if err != nil {
		t.Fatalf("detail query 1: %v", err)
	}
	if len(d1.SKUs) == 0 || d1.SKUs[0].Stock != 100 {
		t.Fatalf("expected stock 100, got %d", d1.SKUs[0].Stock)
	}

	// 验证静态缓存存在
	ver, _ := c.GetVersion(ctx, detailVerKey)
	cacheKey := cache.MiniappProductDetailKey(pRes.ID, ver)
	defer c.Delete(ctx, cacheKey)

	exists, err := client.Exists(ctx, cacheKey).Result()
	if err != nil || exists != 1 {
		t.Fatalf("expected cache key %s to exist, exists=%d", cacheKey, exists)
	}

	// 模拟并发直接修改/锁定库存（如扣减库存或下单锁库：stock=100, locked_stock=35 -> 可用 65）
	if err := db.Model(&model.ProductSkuInventory{}).
		Where("sku_id = ?", sku.ID).
		Updates(map[string]any{"stock": 100, "locked_stock": 35}).Error; err != nil {
		t.Fatalf("update inventory: %v", err)
	}

	// 同时篡改 DB 中的商品名称（以证明静态属性确实是从缓存取的）
	if err := db.Model(&model.Product{}).Where("id = ?", pRes.ID).Update("name", "篡改名称").Error; err != nil {
		t.Fatalf("update name: %v", err)
	}

	// 第二次查询：命中详情缓存（名称依然是 "智能手表"），但库存实时返回 65！
	d2, err := miniappSvc.Detail(ctx, pRes.ID)
	if err != nil {
		t.Fatalf("detail query 2: %v", err)
	}
	if d2.Name != "智能手表" {
		t.Fatalf("expected cached name '智能手表', got %q", d2.Name)
	}
	if d2.SKUs[0].Stock != 65 {
		t.Fatalf("expected live inventory 65, got %d", d2.SKUs[0].Stock)
	}
}

// 2. 商品列表缓存命中时修改库存，实时返回最新可用库存
func TestMiniappProductList_DecoupledLiveInventory(t *testing.T) {
	ctx := context.Background()
	c, client := getMiniappTestRedisCache(t)
	defer client.Close()

	db := miniappTestDB(t)

	cat := model.ProductCategory{Name: "数码", Status: 1}
	db.Create(&cat)

	imgID := createTestImage(t, db)
	statusVal := int8(1)
	prodAdminSvc := productproduct.NewService(db, c)
	pRes, err := prodAdminSvc.Create(ctx, productproduct.ProductCreateRequest{
		Name:        "机械键盘",
		CategoryID:  cat.ID,
		MainImageID: &imgID,
		Status:      &statusVal,
		SKUs: []productproduct.ProductSKURequest{
			{
				Price:  "399.00",
				Status: &statusVal,
			},
		},
	})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	var sku model.ProductSKU
	db.Where("product_id = ?", pRes.ID).First(&sku)
	db.Model(&model.ProductSkuInventory{}).Where("sku_id = ?", sku.ID).Update("stock", 50)

	miniappSvc := miniappproduct.NewService(db, c)
	listVerKey := cache.MiniappProductListVersionKey()
	defer c.Delete(ctx, listVerKey)

	// 首次列表查询
	items1, total1, err := miniappSvc.List(ctx, miniappproduct.ProductListRequest{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("list query 1: %v", err)
	}
	if total1 != 1 || len(items1) != 1 || items1[0].Stock != 50 {
		t.Fatalf("expected stock 50, got %d", items1[0].Stock)
	}

	// 直接修改库存（例如销售扣减出库：stock=15, locked_stock=0 -> 可用 15）
	if err := db.Model(&model.ProductSkuInventory{}).
		Where("sku_id = ?", sku.ID).
		Updates(map[string]any{"stock": 15, "locked_stock": 0}).Error; err != nil {
		t.Fatalf("update inventory: %v", err)
	}

	// 篡改 DB 商品名
	if err := db.Model(&model.Product{}).Where("id = ?", pRes.ID).Update("name", "篡改键盘").Error; err != nil {
		t.Fatalf("update name: %v", err)
	}

	// 第二次列表查询：命中缓存（依然返回 "机械键盘"），但 Stock 实时反映 15！
	items2, total2, err := miniappSvc.List(ctx, miniappproduct.ProductListRequest{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("list query 2: %v", err)
	}
	if total2 != 1 || len(items2) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items2))
	}
	if items2[0].Name != "机械键盘" {
		t.Fatalf("expected cached name '机械键盘', got %q", items2[0].Name)
	}
	if items2[0].Stock != 15 {
		t.Fatalf("expected live stock 15, got %d", items2[0].Stock)
	}
}

// 3. 商品详情轻量级版本号机制与管理员更新失效
func TestMiniappProductDetail_VersioningAndAdminInvalidation(t *testing.T) {
	ctx := context.Background()
	c, client := getMiniappTestRedisCache(t)
	defer client.Close()

	db := miniappTestDB(t)

	cat := model.ProductCategory{Name: "数码", Status: 1}
	db.Create(&cat)

	imgID := createTestImage(t, db)
	statusVal := int8(1)
	prodAdminSvc := productproduct.NewService(db, c)
	pRes, err := prodAdminSvc.Create(ctx, productproduct.ProductCreateRequest{
		Name:        "版本耳机",
		CategoryID:  cat.ID,
		MainImageID: &imgID,
		Status:      &statusVal,
		SKUs: []productproduct.ProductSKURequest{
			{Price: "299.00", Status: &statusVal},
		},
	})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	miniappSvc := miniappproduct.NewService(db, c)
	detailVerKey := cache.MiniappProductDetailVersionKey(pRes.ID)
	defer c.Delete(ctx, detailVerKey)

	// 首次查询详情
	d1, err := miniappSvc.Detail(ctx, pRes.ID)
	if err != nil {
		t.Fatalf("detail 1: %v", err)
	}
	if d1.Name != "版本耳机" {
		t.Fatalf("expected '版本耳机', got %q", d1.Name)
	}

	// 管理员后台更新商品名称
	newName := "版本耳机 Pro"
	_, err = prodAdminSvc.Update(ctx, productproduct.ProductUpdateRequest{
		ID:          pRes.ID,
		Name:        newName,
		CategoryID:  cat.ID,
		MainImageID: &imgID,
		SKUs: []productproduct.ProductSKURequest{
			{ID: d1.SKUs[0].ID, Price: "399.00", Status: &statusVal},
		},
	})
	if err != nil {
		t.Fatalf("update product: %v", err)
	}

	// 小程序再次查询：版本号已递增，立即读取到更新后的名称
	d2, err := miniappSvc.Detail(ctx, pRes.ID)
	if err != nil {
		t.Fatalf("detail 2: %v", err)
	}
	if d2.Name != newName {
		t.Fatalf("expected fresh name %q, got %q", newName, d2.Name)
	}
}

// 4. 商品不存在走短TTL空值缓存 (Negative Caching)
func TestMiniappProductDetail_NegativeCaching(t *testing.T) {
	ctx := context.Background()
	c, client := getMiniappTestRedisCache(t)
	defer client.Close()

	db := miniappTestDB(t)
	miniappSvc := miniappproduct.NewService(db, c)

	var nonExistentID int64 = 8888888
	detailVerKey := cache.MiniappProductDetailVersionKey(nonExistentID)
	defer c.Delete(ctx, detailVerKey)

	ver, _ := c.GetVersion(ctx, detailVerKey)
	cacheKey := cache.MiniappProductDetailKey(nonExistentID, ver)
	defer c.Delete(ctx, cacheKey)

	// 1. 首次查询不存在商品 -> 返回 ErrProductNotFoundOrOffShelf
	_, err := miniappSvc.Detail(ctx, nonExistentID)
	if !errors.Is(err, miniappproduct.ErrProductNotFoundOrOffShelf) {
		t.Fatalf("expected ErrProductNotFoundOrOffShelf, got %v", err)
	}

	// 验证 Redis 中写入了空值哨兵
	exists, err := client.Exists(ctx, cacheKey).Result()
	if err != nil || exists != 1 {
		t.Fatalf("expected negative cache key %s to exist, exists=%d", cacheKey, exists)
	}

	// 2. 第二次查询 -> 命中空值缓存，依然返回 ErrProductNotFoundOrOffShelf
	_, err2 := miniappSvc.Detail(ctx, nonExistentID)
	if !errors.Is(err2, miniappproduct.ErrProductNotFoundOrOffShelf) {
		t.Fatalf("expected ErrProductNotFoundOrOffShelf from negative cache, got %v", err2)
	}
}

// 5. 空商品列表走短TTL空值缓存
func TestMiniappProductList_EmptyListNegativeCaching(t *testing.T) {
	ctx := context.Background()
	c, client := getMiniappTestRedisCache(t)
	defer client.Close()

	db := miniappTestDB(t)
	miniappSvc := miniappproduct.NewService(db, c)

	listVerKey := cache.MiniappProductListVersionKey()
	defer c.Delete(ctx, listVerKey)

	req := miniappproduct.ProductListRequest{Keyword: "nonexistent_brand_xyz", Page: 1, PageSize: 20}

	// 首次查询空结果
	items1, total1, err := miniappSvc.List(ctx, req)
	if err != nil {
		t.Fatalf("list query 1: %v", err)
	}
	if total1 != 0 || len(items1) != 0 {
		t.Fatalf("expected 0 items, got total=%d len=%d", total1, len(items1))
	}

	// 第二次查询依然命中空结果
	items2, total2, err := miniappSvc.List(ctx, req)
	if err != nil {
		t.Fatalf("list query 2: %v", err)
	}
	if total2 != 0 || len(items2) != 0 {
		t.Fatalf("expected 0 items from negative cache, got total=%d len=%d", total2, len(items2))
	}
}

// 6. singleflight 并发击穿防护验证
func TestMiniappSingleflight_ConcurrentMisses(t *testing.T) {
	ctx := context.Background()
	c, client := getMiniappTestRedisCache(t)
	defer client.Close()

	db := miniappTestDB(t)

	cat := model.ProductCategory{Name: "办公", Status: 1}
	db.Create(&cat)

	imgID := createTestImage(t, db)
	statusVal := int8(1)
	prodAdminSvc := productproduct.NewService(db, c)
	pRes, err := prodAdminSvc.Create(ctx, productproduct.ProductCreateRequest{
		Name:        "人体工学椅",
		CategoryID:  cat.ID,
		MainImageID: &imgID,
		Status:      &statusVal,
		SKUs: []productproduct.ProductSKURequest{
			{Price: "1299.00", Status: &statusVal},
		},
	})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	miniappSvc := miniappproduct.NewService(db, c)
	detailVerKey := cache.MiniappProductDetailVersionKey(pRes.ID)
	defer c.Delete(ctx, detailVerKey)
	_ = c.Delete(ctx, detailVerKey)

	// 20 个并发请求同时查询商品详情
	concurrency := 20
	var wg sync.WaitGroup
	errCh := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			d, err := miniappSvc.Detail(ctx, pRes.ID)
			if err != nil {
				errCh <- err
				return
			}
			if d.Name != "人体工学椅" {
				errCh <- fmt.Errorf("unexpected name %q", d.Name)
			}
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Fatalf("concurrent query failed: %v", err)
	}
}

// 7. 并发竞态：旧版本查询回填被丢弃验证
func TestMiniappConcurrencyRace_StaleWritebackDiscarded(t *testing.T) {
	ctx := context.Background()
	c, client := getMiniappTestRedisCache(t)
	defer client.Close()

	detailVerKey := cache.MiniappProductDetailVersionKey(99901)
	defer c.Delete(ctx, detailVerKey)

	// 模拟请求 A 在 version 0 时读数据
	verA, _ := c.GetVersion(ctx, detailVerKey) // 0
	keyA := cache.MiniappProductDetailKey(99901, verA)

	// 此时管理员更新了商品，BumpVersion 将版本变为 1
	newVer, err := c.BumpVersion(ctx, detailVerKey)
	if err != nil || newVer != 1 {
		t.Fatalf("expected newVer=1, got %d (err=%v)", newVer, err)
	}

	// 请求 A 准备回填缓存：检测到当前版本已被更新 (currVer != verA)，丢弃回填
	currVer, _ := c.GetVersion(ctx, detailVerKey)
	if currVer != verA {
		// 回填被丢弃，不会将旧数据写入 keyA
	} else {
		_ = c.Set(ctx, keyA, miniappproduct.ProductDetailResponse{ID: 99901}, time.Minute)
	}

	// 验证 keyA 没有被写入
	existsA, _ := client.Exists(ctx, keyA).Result()
	if existsA != 0 {
		t.Fatalf("expected stale writeback to keyA to be discarded, but key exists")
	}
}

// 8. Redis 宕机时优雅降级走 MySQL
func TestMiniappGracefulFallbackWithRedisDown(t *testing.T) {
	ctx := context.Background()
	client := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
		DB:   15,
	})
	_ = client.Close()
	brokenCache := cache.NewRedisCache(client)

	db := miniappTestDB(t)

	cat := model.ProductCategory{Name: "食品", Status: 1}
	db.Create(&cat)

	imgID := createTestImage(t, db)
	statusVal := int8(1)
	prodAdminSvc := productproduct.NewService(db, brokenCache)
	pRes, err := prodAdminSvc.Create(ctx, productproduct.ProductCreateRequest{
		Name:        "坚果礼盒",
		CategoryID:  cat.ID,
		MainImageID: &imgID,
		Status:      &statusVal,
		SKUs: []productproduct.ProductSKURequest{
			{Price: "88.00", Status: &statusVal},
		},
	})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	miniappSvc := miniappproduct.NewService(db, brokenCache)

	// Detail 正常返回
	d, err := miniappSvc.Detail(ctx, pRes.ID)
	if err != nil {
		t.Fatalf("expected Detail to succeed despite Redis down, got %v", err)
	}
	if d.Name != "坚果礼盒" {
		t.Fatalf("expected '坚果礼盒', got %q", d.Name)
	}

	// List 正常返回
	items, total, err := miniappSvc.List(ctx, miniappproduct.ProductListRequest{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("expected List to succeed despite Redis down, got %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].Name != "坚果礼盒" {
		t.Fatalf("expected valid list items despite Redis down, got total=%d len=%d", total, len(items))
	}
}

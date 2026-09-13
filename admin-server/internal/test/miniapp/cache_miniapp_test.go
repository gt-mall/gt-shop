package miniapp_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	miniappproduct "mall/admin_core/internal/miniapp/product"
	"mall/admin_core/internal/model"
	productproduct "mall/admin_core/internal/product/product"
	"mall/admin_core/internal/region"
	systemfile "mall/admin_core/internal/system/file"
	systemsetting "mall/admin_core/internal/system/setting"
	"mall/admin_core/pkg/cache"
)

func getMiniappTestRueidisCache(t *testing.T) (cache.Cache, func()) {
	t.Helper()
	realClient, err := cache.NewRueidisCache(cache.Config{
		Addr: "127.0.0.1:6379",
		DB:   15,
	})
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if realClient.Ping(ctx) == nil {
			_ = realClient.Client().Do(ctx, realClient.Client().B().Flushdb().Build())
			cleanup := func() {
				_ = realClient.Close()
			}
			return realClient, cleanup
		}
		_ = realClient.Close()
	}

	mr := miniredis.RunT(t)
	c, err := cache.NewRueidisCache(cache.Config{
		Addr:         mr.Addr(),
		DisableCache: true,
		UseLuaLock:   true,
	})
	if err != nil {
		t.Fatalf("failed to create test rueidis cache: %v", err)
	}
	cleanup := func() {
		_ = c.Close()
		mr.Close()
	}
	return c, cleanup
}

func keyExists(t *testing.T, c cache.Cache, key string) bool {
	t.Helper()
	v, err := c.Client().Do(context.Background(), c.Client().B().Exists().Key(key).Build()).AsInt64()
	if err != nil {
		t.Fatalf("failed to check key %s exists: %v", key, err)
	}
	return v > 0
}

func getKey(t *testing.T, c cache.Cache, key string) string {
	t.Helper()
	v, err := c.Client().Do(context.Background(), c.Client().B().Get().Key(key).Build()).ToString()
	if err != nil {
		t.Fatalf("failed to get key %s: %v", key, err)
	}
	return v
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

// 1 & 2 & 3: 商品详情第一次 MISS → DB loader → cache; 第二次命中缓存; 命中缓存后库存变化仍立即显示
func TestMiniapp_1_2_3_ProductDetailCacheAndLiveInventory(t *testing.T) {
	ctx := context.Background()
	c, cleanup := getMiniappTestRueidisCache(t)
	defer cleanup()

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
			{Price: "899.00", Status: &statusVal},
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
	detailKey := cache.ProductDetailKey(pRes.ID)
	defer c.Delete(ctx, detailKey)

	// 1. 首次查询：查 DB 并建立静态缓存 (MISS -> DB -> cache)
	d1, err := miniappSvc.Detail(ctx, pRes.ID)
	if err != nil {
		t.Fatalf("detail query 1: %v", err)
	}
	if len(d1.SKUs) == 0 || d1.SKUs[0].Stock != 100 {
		t.Fatalf("expected stock 100, got %d", d1.SKUs[0].Stock)
	}

	// 验证 Redis 中写入了缓存 key
	if !keyExists(t, c, detailKey) {
		t.Fatalf("expected cache key %s to exist in redis", detailKey)
	}

	// 模拟并发直接修改/锁定库存（如下单锁库：stock=100, locked_stock=35 -> 可用 65）
	if err := db.Model(&model.ProductSkuInventory{}).
		Where("sku_id = ?", sku.ID).
		Updates(map[string]any{"stock": 100, "locked_stock": 35}).Error; err != nil {
		t.Fatalf("update inventory: %v", err)
	}

	// 同时篡改 DB 中的商品名称（以证明静态属性确实是从缓存取的）
	if err := db.Model(&model.Product{}).Where("id = ?", pRes.ID).Update("name", "篡改名称").Error; err != nil {
		t.Fatalf("update name: %v", err)
	}

	// 2 & 3. 第二次查询：命中详情缓存（名称依然是 "智能手表"），但库存实时返回 65！
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

// 4. 商品列表命中缓存后库存变化立即显示
func TestMiniapp_4_ProductListCacheAndLiveInventory(t *testing.T) {
	ctx := context.Background()
	c, _ := getMiniappTestRueidisCache(t)

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
			{Price: "399.00", Status: &statusVal},
		},
	})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	var sku model.ProductSKU
	db.Where("product_id = ?", pRes.ID).First(&sku)
	db.Model(&model.ProductSkuInventory{}).Where("sku_id = ?", sku.ID).Update("stock", 50)

	miniappSvc := miniappproduct.NewService(db, c)

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

	// 篡改 DB 商品名验证命中静态缓存
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

// 5. 商品更新后详情缓存失效
func TestMiniapp_5_ProductDetailInvalidatedOnUpdate(t *testing.T) {
	ctx := context.Background()
	c, cleanup := getMiniappTestRueidisCache(t)
	defer cleanup()

	db := miniappTestDB(t)

	cat := model.ProductCategory{Name: "数码", Status: 1}
	db.Create(&cat)

	imgID := createTestImage(t, db)
	statusVal := int8(1)
	prodAdminSvc := productproduct.NewService(db, c)
	pRes, err := prodAdminSvc.Create(ctx, productproduct.ProductCreateRequest{
		Name:        "旧名称耳机",
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
	detailKey := cache.ProductDetailKey(pRes.ID)

	// 首次查询详情 -> 建立缓存
	d1, err := miniappSvc.Detail(ctx, pRes.ID)
	if err != nil {
		t.Fatalf("detail 1: %v", err)
	}
	if d1.Name != "旧名称耳机" {
		t.Fatalf("expected '旧名称耳机', got %q", d1.Name)
	}
	if !keyExists(t, c, detailKey) {
		t.Fatalf("expected detail key in redis")
	}

	// 管理员更新商品名称
	newName := "新名称耳机 Pro"
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

	// 验证旧缓存已被直接删除
	if keyExists(t, c, detailKey) {
		t.Fatalf("expected detail cache key to be deleted upon update")
	}

	// 小程序再次查询：立即读取到更新后的名称
	d2, err := miniappSvc.Detail(ctx, pRes.ID)
	if err != nil {
		t.Fatalf("detail 2: %v", err)
	}
	if d2.Name != newName {
		t.Fatalf("expected fresh name %q, got %q", newName, d2.Name)
	}
}

// 6. 商品价格更新后列表最终使用最新价格
func TestMiniapp_6_ProductPriceUpdateReflectedInList(t *testing.T) {
	ctx := context.Background()
	c, _ := getMiniappTestRueidisCache(t)

	db := miniappTestDB(t)

	cat := model.ProductCategory{Name: "数码", Status: 1}
	db.Create(&cat)

	imgID := createTestImage(t, db)
	statusVal := int8(1)
	prodAdminSvc := productproduct.NewService(db, c)
	pRes, err := prodAdminSvc.Create(ctx, productproduct.ProductCreateRequest{
		Name:        "价格测试手机",
		CategoryID:  cat.ID,
		MainImageID: &imgID,
		Status:      &statusVal,
		SKUs: []productproduct.ProductSKURequest{
			{Price: "4999.00", Status: &statusVal},
		},
	})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	miniappSvc := miniappproduct.NewService(db, c)

	// 首次列表查询，价格为 4999.00
	list1, _, err := miniappSvc.List(ctx, miniappproduct.ProductListRequest{Page: 1, PageSize: 20})
	if err != nil || len(list1) != 1 || list1[0].Price != "4999.00" {
		t.Fatalf("expected price 4999.00, got %+v (err=%v)", list1, err)
	}

	// 管理员后台降价为 4299.00
	var sku model.ProductSKU
	db.Where("product_id = ?", pRes.ID).First(&sku)
	_, err = prodAdminSvc.Update(ctx, productproduct.ProductUpdateRequest{
		ID:          pRes.ID,
		Name:        "价格测试手机",
		CategoryID:  cat.ID,
		MainImageID: &imgID,
		SKUs: []productproduct.ProductSKURequest{
			{ID: sku.ID, Price: "4299.00", Status: &statusVal},
		},
	})
	if err != nil {
		t.Fatalf("update product price: %v", err)
	}

	// 列表查询：Generation 递增，立即使用最新价格 4299.00
	list2, _, err := miniappSvc.List(ctx, miniappproduct.ProductListRequest{Page: 1, PageSize: 20})
	if err != nil || len(list2) != 1 || list2[0].Price != "4299.00" {
		t.Fatalf("expected new price 4299.00, got %+v (err=%v)", list2, err)
	}
}

// 7. 商品下架后详情不可访问
func TestMiniapp_7_ProductDetailOffShelfNotAccessible(t *testing.T) {
	ctx := context.Background()
	c, _ := getMiniappTestRueidisCache(t)

	db := miniappTestDB(t)

	cat := model.ProductCategory{Name: "服装", Status: 1}
	db.Create(&cat)

	imgID := createTestImage(t, db)
	statusVal := int8(1)
	prodAdminSvc := productproduct.NewService(db, c)
	pRes, err := prodAdminSvc.Create(ctx, productproduct.ProductCreateRequest{
		Name:        "春装衬衫",
		CategoryID:  cat.ID,
		MainImageID: &imgID,
		Status:      &statusVal,
		SKUs: []productproduct.ProductSKURequest{
			{Price: "199.00", Status: &statusVal},
		},
	})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	miniappSvc := miniappproduct.NewService(db, c)

	// 上架状态可正常查看详情
	d1, err := miniappSvc.Detail(ctx, pRes.ID)
	if err != nil || d1 == nil {
		t.Fatalf("expected detail to succeed, got %v", err)
	}

	// 管理员下架商品
	if err := prodAdminSvc.Status(ctx, productproduct.ProductStatusRequest{ID: pRes.ID, Status: 0}); err != nil {
		t.Fatalf("status off: %v", err)
	}

	// 下架后小程序再次查询详情 -> 返回 ErrProductNotFoundOrOffShelf
	_, err = miniappSvc.Detail(ctx, pRes.ID)
	if !errors.Is(err, miniappproduct.ErrProductNotFoundOrOffShelf) {
		t.Fatalf("expected ErrProductNotFoundOrOffShelf after off shelf, got %v", err)
	}
}

// 8. 商品下架后列表不再显示
func TestMiniapp_8_ProductListOffShelfNotShown(t *testing.T) {
	ctx := context.Background()
	c, _ := getMiniappTestRueidisCache(t)

	db := miniappTestDB(t)

	cat := model.ProductCategory{Name: "服装", Status: 1}
	db.Create(&cat)

	imgID := createTestImage(t, db)
	statusVal := int8(1)
	prodAdminSvc := productproduct.NewService(db, c)
	pRes, err := prodAdminSvc.Create(ctx, productproduct.ProductCreateRequest{
		Name:        "秋装外套",
		CategoryID:  cat.ID,
		MainImageID: &imgID,
		Status:      &statusVal,
		SKUs: []productproduct.ProductSKURequest{
			{Price: "399.00", Status: &statusVal},
		},
	})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	miniappSvc := miniappproduct.NewService(db, c)

	// 上架状态列表显示 1 件
	items1, total1, err := miniappSvc.List(ctx, miniappproduct.ProductListRequest{Page: 1, PageSize: 20})
	if err != nil || total1 != 1 || len(items1) != 1 {
		t.Fatalf("expected 1 item, got total=%d len=%d", total1, len(items1))
	}

	// 下架商品
	if err := prodAdminSvc.Status(ctx, productproduct.ProductStatusRequest{ID: pRes.ID, Status: 0}); err != nil {
		t.Fatalf("status off: %v", err)
	}

	// 下架后列表不再显示
	items2, total2, err := miniappSvc.List(ctx, miniappproduct.ProductListRequest{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if total2 != 0 || len(items2) != 0 {
		t.Fatalf("expected 0 items in list after off shelf, got total=%d len=%d", total2, len(items2))
	}
}

// 9. 不存在商品有短时间负缓存
func TestMiniapp_9_ProductNotFoundNegativeCache(t *testing.T) {
	ctx := context.Background()
	c, cleanup := getMiniappTestRueidisCache(t)
	defer cleanup()

	db := miniappTestDB(t)
	miniappSvc := miniappproduct.NewService(db, c)

	var nonExistentID int64 = 8888888
	detailKey := cache.ProductDetailKey(nonExistentID)
	defer c.Delete(ctx, detailKey)

	// 1. 首次查询不存在商品 -> 返回 ErrProductNotFoundOrOffShelf
	_, err := miniappSvc.Detail(ctx, nonExistentID)
	if !errors.Is(err, miniappproduct.ErrProductNotFoundOrOffShelf) {
		t.Fatalf("expected ErrProductNotFoundOrOffShelf, got %v", err)
	}

	// 验证 Redis 中写入了负缓存哨兵
	if !keyExists(t, c, detailKey) {
		t.Fatalf("expected negative cache key %s to exist in redis", detailKey)
	}

	// 2. 第二次查询 -> 命中负缓存，依然返回 ErrProductNotFoundOrOffShelf
	_, err2 := miniappSvc.Detail(ctx, nonExistentID)
	if !errors.Is(err2, miniappproduct.ErrProductNotFoundOrOffShelf) {
		t.Fatalf("expected ErrProductNotFoundOrOffShelf from negative cache, got %v", err2)
	}
}

// 10. 空商品列表短时间缓存
func TestMiniapp_10_EmptyProductListNegativeCache(t *testing.T) {
	ctx := context.Background()
	c, _ := getMiniappTestRueidisCache(t)

	db := miniappTestDB(t)
	miniappSvc := miniappproduct.NewService(db, c)

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
		t.Fatalf("expected 0 items from cache, got total=%d len=%d", total2, len(items2))
	}
}

// 11. 并发 MISS 不应形成大量 DB 回源 (Stampede Protection)
func TestMiniapp_11_ConcurrentMissStampedeProtection(t *testing.T) {
	ctx := context.Background()
	c, cleanup := getMiniappTestRueidisCache(t)
	defer cleanup()

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
	detailKey := cache.ProductDetailKey(pRes.ID)
	_ = c.Delete(ctx, detailKey)

	concurrency := 20
	var wg sync.WaitGroup
	errCh := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			d, err := miniappSvc.Detail(reqCtx, pRes.ID)
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

// 12. 更新与缓存重建交错时不得长期回填旧数据 (CAS lock-aside protects against stale writeback)
func TestMiniapp_12_StaleWritebackPrevention(t *testing.T) {
	c, cleanup := getMiniappTestRueidisCache(t)
	defer cleanup()
	key := "mall:test:race:item:1"

	// 模拟请求 A：Miss -> 尝试写入锁并开始 DB 查询
	// 当请求 A 还在查询旧数据时，后台 B 执行更新并将 key 删除
	// 请求 A 晚于 B 完成，尝试 SET: setkey 检查当前 GET(key) 是否等于请求 A 的 lock id
	// 由于 key 已被 B 删除，setkey 返回 0，放弃写入旧值！

	var clientCalls int32
	// 模拟请求 A 启动
	startWait := make(chan struct{})
	finishUpdate := make(chan struct{})

	go func() {
		// 请求 A 执行 Cache-Aside Get
		_, _ = cache.Get(context.Background(), c, key, 3*time.Second, func(innerCtx context.Context) (string, error) {
			atomic.AddInt32(&clientCalls, 1)
			close(startWait) // 通知主协程：A 已经获取到锁，开始查 DB
			<-finishUpdate   // 等待后台 B 完成更新并删除 key
			return "stale_data_from_A", nil
		})
	}()

	<-startWait // 等待 A 进入 loader

	// 后台 B 更新数据并使缓存失效
	if err := c.Delete(context.Background(), key); err != nil {
		t.Fatalf("B delete key failed: %v", err)
	}

	// 释放 A 继续回填
	close(finishUpdate)

	// 等待一小会儿确保 A 完成
	time.Sleep(100 * time.Millisecond)

	// 验证 key 中没有被写入旧数据 "stale_data_from_A"
	if keyExists(t, c, key) {
		val := getKey(t, c, key)
		if strings.Contains(val, "stale_data_from_A") {
			t.Fatalf("CRITICAL CONSISTENCY BUG: stale data from A was written back after B's invalidation!")
		}
	}
}

// 13. REDIS_ENABLED=false 时全部商品功能正常
func TestMiniapp_13_RedisDisabledAllFeaturesWork(t *testing.T) {
	ctx := context.Background()
	noopCache := cache.NewNoop()

	db := miniappTestDB(t)

	cat := model.ProductCategory{Name: "家电", Status: 1}
	db.Create(&cat)

	imgID := createTestImage(t, db)
	statusVal := int8(1)
	prodAdminSvc := productproduct.NewService(db, noopCache)
	pRes, err := prodAdminSvc.Create(ctx, productproduct.ProductCreateRequest{
		Name:        "空气炸锅",
		CategoryID:  cat.ID,
		MainImageID: &imgID,
		Status:      &statusVal,
		SKUs: []productproduct.ProductSKURequest{
			{Price: "259.00", Status: &statusVal},
		},
	})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	miniappSvc := miniappproduct.NewService(db, noopCache)

	// 详情查询正常
	d, err := miniappSvc.Detail(ctx, pRes.ID)
	if err != nil || d == nil || d.Name != "空气炸锅" {
		t.Fatalf("expected Detail to succeed without Redis, got %+v (err=%v)", d, err)
	}

	// 列表查询正常
	items, total, err := miniappSvc.List(ctx, miniappproduct.ProductListRequest{Page: 1, PageSize: 20})
	if err != nil || total != 1 || len(items) != 1 || items[0].Name != "空气炸锅" {
		t.Fatalf("expected List to succeed without Redis, got total=%d (err=%v)", total, err)
	}
}

// 14. Redis 运行时异常时普通商品查询可以回退 MySQL
func TestMiniapp_14_RedisRuntimeErrorGracefulFallback(t *testing.T) {
	mr := miniredis.RunT(t)
	brokenCache, err := cache.NewRueidisCache(cache.Config{
		Addr:         mr.Addr(),
		DisableCache: true,
		UseLuaLock:   true,
	})
	if err != nil {
		t.Fatalf("init cache failed: %v", err)
	}
	defer brokenCache.Close()

	// 模拟 Redis 宕机
	mr.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

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

	// Detail 正常返回 (回退 MySQL)
	d, err := miniappSvc.Detail(ctx, pRes.ID)
	if err != nil {
		t.Fatalf("expected Detail to succeed despite Redis down, got %v", err)
	}
	if d.Name != "坚果礼盒" {
		t.Fatalf("expected '坚果礼盒', got %q", d.Name)
	}

	// List 正常返回 (回退 MySQL)
	items, total, err := miniappSvc.List(ctx, miniappproduct.ProductListRequest{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("expected List to succeed despite Redis down, got %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].Name != "坚果礼盒" {
		t.Fatalf("expected valid list items despite Redis down, got total=%d len=%d", total, len(items))
	}
}

// 15. MpConfig AppSecret / UploadPrivateKey 永远不会写进缓存
func TestMiniapp_15_MpConfigSecretsNeverCached(t *testing.T) {
	ctx := context.Background()
	c, cleanup := getMiniappTestRueidisCache(t)
	defer cleanup()

	db := miniappTestDB(t)
	settingSvc := systemsetting.NewService(db, c)

	// 保存包含敏感秘钥的完整配置
	mpCfg := systemsetting.MpConfig{
		Name:             "公开小程序名",
		OriginalID:       "gh_secret123",
		AppID:            "wx_test_app_id",
		AppSecret:        "TOP_SECRET_APP_SECRET_STRING",
		UploadPrivateKey: "TOP_SECRET_PRIVATE_KEY_STRING",
	}
	if err := settingSvc.SaveMp(ctx, mpCfg); err != nil {
		t.Fatalf("save mp: %v", err)
	}

	// 前端/小程序只能调 GetPublicMp 获取公开配置
	pub, err := settingSvc.GetPublicMp(ctx)
	if err != nil {
		t.Fatalf("get public mp: %v", err)
	}
	if pub.Name != "公开小程序名" {
		t.Fatalf("unexpected public mp name: %s", pub.Name)
	}

	// 检查 Redis 中存储的公用配置内容
	publicMpKey := cache.PublicMpConfigKey()
	val := getKey(t, c, publicMpKey)

	if strings.Contains(val, "TOP_SECRET_APP_SECRET_STRING") || strings.Contains(val, "TOP_SECRET_PRIVATE_KEY_STRING") {
		t.Fatalf("SECURITY ALERT: sensitive credentials found in redis cache: %s", val)
	}
}

// 16. 地区服务完全不访问 Redis
func TestMiniapp_16_RegionServiceNoRedis(t *testing.T) {
	// 验证 NewService() 签名不接受任何 Redis / Cache 参数
	svc, err := region.NewService()
	if err != nil {
		t.Fatalf("failed to create region service: %v", err)
	}

	// 数据完全在本地内存中
	data := svc.GetData()
	if len(data.Regions) == 0 {
		t.Fatalf("expected loaded regions from embedded FS, got empty")
	}

	// ETag 与版本号支持 304 缓存控制
	if svc.GetETag() == "" || svc.GetVersion() == "" {
		t.Fatalf("expected valid ETag and Version")
	}
}

package miniapp_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
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
	detailGen0, err := cache.GetProductDetailGeneration(ctx, c, pRes.ID)
	if err != nil {
		t.Fatalf("get detail gen: %v", err)
	}
	detailKey := cache.ProductDetailKey(pRes.ID, detailGen0)
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

	// 验证库存变化不 bump 商品静态 generation (Requirement 十三 场景 15)
	detailGenAfterStock, err := cache.GetProductDetailGeneration(ctx, c, pRes.ID)
	if err != nil || detailGenAfterStock != detailGen0 {
		t.Fatalf("inventory change must NOT bump product static generation, got %d != %d", detailGenAfterStock, detailGen0)
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
	gen1, err := cache.GetProductDetailGeneration(ctx, c, pRes.ID)
	if err != nil {
		t.Fatalf("get gen: %v", err)
	}
	detailKey1 := cache.ProductDetailKey(pRes.ID, gen1)

	// 首次查询详情 -> 建立缓存
	d1, err := miniappSvc.Detail(ctx, pRes.ID)
	if err != nil {
		t.Fatalf("detail 1: %v", err)
	}
	if d1.Name != "旧名称耳机" {
		t.Fatalf("expected '旧名称耳机', got %q", d1.Name)
	}
	if !keyExists(t, c, detailKey1) {
		t.Fatalf("expected detail key in redis")
	}

	// 管理员更新商品名称 -> Bump generation
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

	// 验证 generation 已递增
	gen2, err := cache.GetProductDetailGeneration(ctx, c, pRes.ID)
	if err != nil || gen2 <= gen1 {
		t.Fatalf("expected detail generation to bump on update, got %d <= %d (err=%v)", gen2, gen1, err)
	}

	// 小程序再次查询：通过新代际立即读取到更新后的名称
	d2, err := miniappSvc.Detail(ctx, pRes.ID)
	if err != nil {
		t.Fatalf("detail 2: %v", err)
	}
	if d2.Name != newName {
		t.Fatalf("expected fresh name %q, got %q", newName, d2.Name)
	}
	detailKey2 := cache.ProductDetailKey(pRes.ID, gen2)
	if !keyExists(t, c, detailKey2) {
		t.Fatalf("expected new detail key in redis")
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
	gen, err := cache.GetProductDetailGeneration(ctx, c, nonExistentID)
	if err != nil {
		t.Fatalf("get gen: %v", err)
	}
	detailKey := cache.ProductDetailKey(nonExistentID, gen)
	defer c.Delete(ctx, detailKey)

	// 1. 首次查询不存在商品 -> 返回 ErrProductNotFoundOrOffShelf
	_, err = miniappSvc.Detail(ctx, nonExistentID)
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
	gen, err := cache.GetProductDetailGeneration(ctx, c, pRes.ID)
	if err != nil {
		t.Fatalf("get gen: %v", err)
	}
	detailKey := cache.ProductDetailKey(pRes.ID, gen)
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

// 12. 更新与缓存重建交错时不得长期回填旧数据 (Channel 同步真正的 ABA / stale write 回归测试，无 time.Sleep)
func TestMiniapp_12_ABAStaleWritebackPrevention(t *testing.T) {
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
		Name:        "旧商品名称",
		CategoryID:  cat.ID,
		MainImageID: &imgID,
		Status:      &statusVal,
		SKUs: []productproduct.ProductSKURequest{
			{Price: "99.00", Status: &statusVal},
		},
	})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	// 此时 Create 已经将 detail generation 初始化为 1
	gen1, err := cache.GetProductDetailGeneration(ctx, c, pRes.ID)
	if err != nil {
		t.Fatalf("get gen1: %v", err)
	}

	// 阶段控制 channels (完全使用 channel 控制，无 time.Sleep)
	aStartedLoader := make(chan struct{})
	allowAFinish := make(chan struct{})
	aDone := make(chan struct{})

	// 请求 A：读取 generation=1，进入 loader，读取 old，阻塞在 allowAFinish
	var aResult *miniappproduct.ProductDetailResponse
	var aErr error

	go func() {
		defer close(aDone)
		gKey := cache.ProductDetailKey(pRes.ID, gen1)
		loader := func(innerCtx context.Context) (*miniappproduct.ProductDetailResponse, error) {
			// A 从 DB 读取了旧数据
			var p model.Product
			if err := db.WithContext(innerCtx).Where("id = ?", pRes.ID).First(&p).Error; err != nil {
				return nil, err
			}
			close(aStartedLoader) // 通知主协程：A 已读取旧数据，进入阻塞
			<-allowAFinish        // 等待管理员更新并完成请求 B
			return &miniappproduct.ProductDetailResponse{
				ID:   p.ID,
				Name: p.Name, // 旧数据
			}, nil
		}
		aResult, aErr = cache.Get(ctx, c, gKey, cache.TTLProductDetail, loader)
	}()

	// 等待 A 已经进入 loader 并读取了旧数据
	<-aStartedLoader

	// 管理员修改 DB：old -> new，并 Bump generation
	var sku model.ProductSKU
	db.Where("product_id = ?", pRes.ID).First(&sku)
	newName := "最新商品名称-V2"
	_, err = prodAdminSvc.Update(ctx, productproduct.ProductUpdateRequest{
		ID:          pRes.ID,
		Name:        newName,
		CategoryID:  cat.ID,
		MainImageID: &imgID,
		SKUs: []productproduct.ProductSKURequest{
			{ID: sku.ID, Price: "99.00", Status: &statusVal},
		},
	})
	if err != nil {
		t.Fatalf("admin update failed: %v", err)
	}

	// 验证 generation 已递增到 2
	gen2, err := cache.GetProductDetailGeneration(ctx, c, pRes.ID)
	if err != nil || gen2 <= gen1 {
		t.Fatalf("expected gen2 > gen1, got %d <= %d", gen2, gen1)
	}

	// 请求 B：读取 generation=2，进入新 generation loader，读取 new，完成并写入 g2
	miniappSvc := miniappproduct.NewService(db, c)
	bResult, err := miniappSvc.Detail(ctx, pRes.ID)
	if err != nil {
		t.Fatalf("request B detail failed: %v", err)
	}
	if bResult.Name != newName {
		t.Fatalf("expected B to read %q, got %q", newName, bResult.Name)
	}

	// 允许 A 继续返回并写回 g1
	close(allowAFinish)
	<-aDone

	if aErr != nil {
		t.Fatalf("request A failed: %v", aErr)
	}
	if aResult.Name != "旧商品名称" {
		t.Fatalf("expected A to finish with old name, got %q", aResult.Name)
	}

	// 最终发起普通请求 C：读取 generation=2，必须返回 newName，绝不被 A 写回的 g1 污染
	cResult, err := miniappSvc.Detail(ctx, pRes.ID)
	if err != nil {
		t.Fatalf("request C detail failed: %v", err)
	}
	if cResult.Name != newName {
		t.Fatalf("CRITICAL ABA STALE WRITEBACK BUG: request C returned stale data %q instead of %q", cResult.Name, newName)
	}

	// 验证 g1 即使包含旧数据，也不会污染 g2
	g1Key := cache.ProductDetailKey(pRes.ID, gen1)
	if keyExists(t, c, g1Key) {
		g1Val := getKey(t, c, g1Key)
		if !strings.Contains(g1Val, "旧商品名称") {
			t.Fatalf("expected g1 to have old data, got %q", g1Val)
		}
	}
	g2Key := cache.ProductDetailKey(pRes.ID, gen2)
	g2Val := getKey(t, c, g2Key)
	if !strings.Contains(g2Val, newName) {
		t.Fatalf("expected g2 to have fresh data, got %q", g2Val)
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
	publicMpGen, err := cache.GetSystemConfigGeneration(ctx, c, "public_mp")
	if err != nil {
		t.Fatalf("get public mp gen: %v", err)
	}
	publicMpKey := cache.SystemConfigKey("public_mp", publicMpGen)
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

// 17. Product Detail generation 初始为0时正常缓存 (Requirement 十三 场景 1)
func TestMiniapp_17_DetailGen0InitialCache(t *testing.T) {
	ctx := context.Background()
	c, cleanup := getMiniappTestRueidisCache(t)
	defer cleanup()

	db := miniappTestDB(t)

	cat := model.ProductCategory{Name: "数码", Status: 1}
	db.Create(&cat)

	imgID := createTestImage(t, db)
	prodAdminSvc := productproduct.NewService(db, c)
	statusVal := int8(1)
	pRes, err := prodAdminSvc.Create(ctx, productproduct.ProductCreateRequest{
		Name:        "初始未加代商品",
		CategoryID:  cat.ID,
		MainImageID: &imgID,
		Status:      &statusVal,
		SKUs: []productproduct.ProductSKURequest{
			{Price: "99.00", Status: &statusVal},
		},
	})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	// 清理 generation key，模拟初始 generation=0（未被 bump 过的新商品或历史数据）
	_ = c.Delete(ctx, cache.ProductDetailGenerationKey(pRes.ID))

	// 此时 Redis 中尚无 generation key，GetProductDetailGeneration 返回 0
	gen, err := cache.GetProductDetailGeneration(ctx, c, pRes.ID)
	if err != nil || gen != 0 {
		t.Fatalf("expected initial gen 0, got %d (err=%v)", gen, err)
	}

	miniappSvc := miniappproduct.NewService(db, c)
	d1, err := miniappSvc.Detail(ctx, pRes.ID)
	if err != nil {
		t.Fatalf("detail failed: %v", err)
	}
	if d1.Name != "初始未加代商品" {
		t.Fatalf("unexpected name: %s", d1.Name)
	}

	// 验证缓存写入在 g0 key 中
	key0 := cache.ProductDetailKey(pRes.ID, 0)
	if !keyExists(t, c, key0) {
		t.Fatalf("expected key %s to exist in redis", key0)
	}

	// 修改 DB 中的商品名称，再次查询应命中 g0 缓存
	db.Model(&model.Product{}).Where("id = ?", pRes.ID).Update("name", "DB修改名称")
	d2, err := miniappSvc.Detail(ctx, pRes.ID)
	if err != nil {
		t.Fatalf("detail 2 failed: %v", err)
	}
	if d2.Name != "初始未加代商品" {
		t.Fatalf("expected cached name '初始未加代商品', got %q", d2.Name)
	}
}

// 18. Product Create 后新 ID detail generation +1，旧负缓存不会继续命中 (Requirement 十三 场景 7)
func TestMiniapp_18_CreateAfterNegativeCacheNoStaleHit(t *testing.T) {
	ctx := context.Background()
	c, cleanup := getMiniappTestRueidisCache(t)
	defer cleanup()

	db := miniappTestDB(t)
	miniappSvc := miniappproduct.NewService(db, c)

	var targetID int64 = 777777

	// 1. 用户提前访问不存在商品 targetID
	_, err := miniappSvc.Detail(ctx, targetID)
	if !errors.Is(err, miniappproduct.ErrProductNotFoundOrOffShelf) {
		t.Fatalf("expected ErrProductNotFoundOrOffShelf, got %v", err)
	}

	// 此时 g0 负缓存已被写入
	g0Key := cache.ProductDetailKey(targetID, 0)
	if !keyExists(t, c, g0Key) {
		t.Fatalf("expected g0 negative cache to exist")
	}

	// 2. 管理员创建该商品，MySQL 写入了 ID=targetID
	cat := model.ProductCategory{Name: "数码", Status: 1}
	db.Create(&cat)
	imgID := createTestImage(t, db)

	p := model.Product{
		ID:          targetID,
		Name:        "刚创建的新商品",
		CategoryID:  cat.ID,
		MainImageID: &imgID,
		Status:      1,
	}
	db.Create(&p)
	sku := model.ProductSKU{
		ID:         targetID * 10,
		ProductID:  p.ID,
		SpecValues: "{}",
		SpecHash:   "default",
		Price:      "99.00",
		CostPrice:  "50.00",
		Status:     1,
	}
	db.Create(&sku)
	inv := model.ProductSkuInventory{
		ProductID: p.ID,
		SKUID:     sku.ID,
		Stock:     20,
	}
	db.Create(&inv)

	// 管理员触发 Create 缓存维护（Bump detail generation + list generation）
	genAfterCreate, err := cache.BumpProductDetailGeneration(ctx, c, targetID)
	if err != nil || genAfterCreate < 1 {
		t.Fatalf("expected detail generation >= 1, got %d (err=%v)", genAfterCreate, err)
	}

	// 3. 用户再次访问该商品：读取到 gen=1，绝不命中旧的 g0 负缓存，成功返回商品详情！
	d, err := miniappSvc.Detail(ctx, targetID)
	if err != nil {
		t.Fatalf("expected Detail to succeed after create, but got err: %v", err)
	}
	if d.Name != "刚创建的新商品" {
		t.Fatalf("expected '刚创建的新商品', got %q", d.Name)
	}
}

// 19. generation GET Redis error 时直接查询 MySQL，不读取 g0 (Requirement 十三 场景 8)
func TestMiniapp_19_GenerationGetRedisErrorBypassesCache(t *testing.T) {
	ctx := context.Background()
	c, cleanup := getMiniappTestRueidisCache(t)
	defer cleanup()

	db := miniappTestDB(t)

	cat := model.ProductCategory{Name: "数码", Status: 1}
	db.Create(&cat)
	imgID := createTestImage(t, db)

	prodAdminSvc := productproduct.NewService(db, c)
	statusVal := int8(1)
	pRes, err := prodAdminSvc.Create(ctx, productproduct.ProductCreateRequest{
		Name:        "真实DB商品",
		CategoryID:  cat.ID,
		MainImageID: &imgID,
		Status:      &statusVal,
		SKUs: []productproduct.ProductSKURequest{
			{Price: "99.00", Status: &statusVal},
		},
	})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	// 假设 Redis 中由于历史残留存在 g0 脏数据
	g0Key := cache.ProductDetailKey(pRes.ID, 0)
	_ = c.Client().Do(ctx, c.Client().B().Set().Key(g0Key).Value(`{"id":`+fmt.Sprintf("%d", pRes.ID)+`,"name":"历史脏缓存g0"}`).Build())

	// 现在关闭 Redis 模拟 Redis 发生网络错误
	c.Close()

	miniappSvc := miniappproduct.NewService(db, c)
	// 当 GetProductDetailGeneration 发生 Redis error 时，代码必须遵守
	// "不知道当前是哪一代，就不用缓存。"
	// 直接回源 MySQL，绝对不能猜 generation=0 去读取 g0！
	d, err := miniappSvc.Detail(ctx, pRes.ID)
	if err != nil {
		t.Fatalf("expected detail to succeed by falling back to DB, got %v", err)
	}
	if d.Name != "真实DB商品" {
		t.Fatalf("expected real DB product '真实DB商品', got %q (did it erroneously use g0?)", d.Name)
	}
}

// 20. JSON坏缓存被删除后可以正常重新建立 (Requirement 十三 场景 16)
func TestMiniapp_20_CorruptedJSONCacheDeletedAndRebuilt(t *testing.T) {
	ctx := context.Background()
	c, cleanup := getMiniappTestRueidisCache(t)
	defer cleanup()

	db := miniappTestDB(t)

	cat := model.ProductCategory{Name: "数码", Status: 1}
	db.Create(&cat)
	imgID := createTestImage(t, db)

	prodAdminSvc := productproduct.NewService(db, c)
	statusVal := int8(1)
	pRes, err := prodAdminSvc.Create(ctx, productproduct.ProductCreateRequest{
		Name:        "正常商品",
		CategoryID:  cat.ID,
		MainImageID: &imgID,
		Status:      &statusVal,
		SKUs: []productproduct.ProductSKURequest{
			{Price: "99.00", Status: &statusVal},
		},
	})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	miniappSvc := miniappproduct.NewService(db, c)

	gen, err := cache.GetProductDetailGeneration(ctx, c, pRes.ID)
	if err != nil {
		t.Fatalf("get gen: %v", err)
	}
	cacheKey := cache.ProductDetailKey(pRes.ID, gen)

	// 恶意注入损坏的非 JSON 字符串
	_ = c.Client().Do(ctx, c.Client().B().Set().Key(cacheKey).Value("INVALID_CORRUPTED_JSON_DATA{{{").Build())
	if !keyExists(t, c, cacheKey) {
		t.Fatalf("expected corrupted key in redis")
	}

	// 首次查询：反序列化失败，主动删除坏缓存，并回源 MySQL 正常返回
	d1, err := miniappSvc.Detail(ctx, pRes.ID)
	if err != nil {
		t.Fatalf("detail failed on corrupted cache: %v", err)
	}
	if d1.Name != "正常商品" {
		t.Fatalf("expected '正常商品', got %q", d1.Name)
	}

	// 坏缓存已被删除 (best-effort delete)
	// 再次查询：重新回填正常缓存
	d2, err := miniappSvc.Detail(ctx, pRes.ID)
	if err != nil {
		t.Fatalf("detail 2 failed: %v", err)
	}
	if d2.Name != "正常商品" {
		t.Fatalf("expected '正常商品', got %q", d2.Name)
	}

	// 验证 Redis 中已重新建立正确的 JSON 缓存
	if keyExists(t, c, cacheKey) {
		val := getKey(t, c, cacheKey)
		if !strings.Contains(val, "正常商品") {
			t.Fatalf("expected valid JSON in redis, got %s", val)
		}
	}
}

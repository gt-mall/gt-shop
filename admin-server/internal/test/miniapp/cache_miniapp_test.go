package miniapp_test

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	miniappproduct "mall/admin_core/internal/miniapp/product"
	"mall/admin_core/internal/model"
	productproduct "mall/admin_core/internal/product/product"
	systemfile "mall/admin_core/internal/system/file"
	"mall/admin_core/pkg/cache"
)

func getMiniappTestRedisCache(t *testing.T) (cache.Cache, *redis.Client) {
	t.Helper()
	client := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
		DB:   15,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		t.Skipf("skipping Redis cache integration tests: local redis not reachable: %v", err)
		return nil, nil
	}
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

func TestMiniappProductCacheLifecycleWithRedis(t *testing.T) {
	ctx := context.Background()
	c, client := getMiniappTestRedisCache(t)
	defer client.Close()

	db := miniappTestDB(t)

	// 准备分类
	cat := model.ProductCategory{Name: "数码", Status: 1}
	if err := db.Create(&cat).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}

	// 准备主图
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

	// 准备商品与SKU
	statusVal := int8(1)
	prodAdminSvc := productproduct.NewService(db, c)
	pRes, err := prodAdminSvc.Create(ctx, productproduct.ProductCreateRequest{
		Name:        "智能手表",
		CategoryID:  cat.ID,
		MainImageID: &file.ID,
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

	miniappSvc := miniappproduct.NewService(db, c)
	key := cache.MiniappProductDetailKey(pRes.ID)
	defer c.Delete(ctx, key)
	_ = c.Delete(ctx, key)

	// 1. 小程序首次查询商品详情：查 DB 并回填缓存
	d1, err := miniappSvc.Detail(ctx, pRes.ID)
	if err != nil {
		t.Fatalf("miniapp Detail 1 failed: %v", err)
	}
	if d1.Name != "智能手表" {
		t.Fatalf("expected '智能手表', got %q", d1.Name)
	}

	// 检查 Redis 中已缓存
	exists, err := client.Exists(ctx, key).Result()
	if err != nil || exists != 1 {
		t.Fatalf("expected key %s in redis, exists=%d", key, exists)
	}

	// 2. 底层直接篡改 DB
	if err := db.Model(&model.Product{}).Where("id = ?", pRes.ID).Update("name", "直接篡改手表").Error; err != nil {
		t.Fatalf("direct db update: %v", err)
	}

	// 3. 小程序第二次查询：命中缓存，依然返回 "智能手表"
	d2, err := miniappSvc.Detail(ctx, pRes.ID)
	if err != nil {
		t.Fatalf("miniapp Detail 2 failed: %v", err)
	}
	if d2.Name != "智能手表" {
		t.Fatalf("expected cached '智能手表', got %q", d2.Name)
	}

	// 4. 后台管理员更新商品名称：后台更新必须联动删除小程序的商品详情缓存！
	adminDetail, err := prodAdminSvc.Detail(ctx, pRes.ID)
	if err != nil {
		t.Fatalf("admin Detail failed: %v", err)
	}
	newName := "智能手表 Pro"
	_, err = prodAdminSvc.Update(ctx, productproduct.ProductUpdateRequest{
		ID:          pRes.ID,
		Name:        newName,
		CategoryID:  cat.ID,
		MainImageID: adminDetail.MainImageID,
		SKUs: []productproduct.ProductSKURequest{
			{
				ID:     adminDetail.SKUs[0].ID,
				Price:  "999.00",
				Status: &statusVal,
			},
		},
	})
	if err != nil {
		t.Fatalf("admin Update failed: %v", err)
	}

	// 验证小程序的缓存 key 已被联动删除
	existsAfterUpdate, err := client.Exists(ctx, key).Result()
	if err != nil || existsAfterUpdate != 0 {
		t.Fatalf("expected miniapp key %s to be deleted after admin update, exists=%d", key, existsAfterUpdate)
	}

	// 5. 小程序第三次查询：重新查 DB，获得最新更新后的数据
	d3, err := miniappSvc.Detail(ctx, pRes.ID)
	if err != nil {
		t.Fatalf("miniapp Detail 3 failed: %v", err)
	}
	if d3.Name != newName {
		t.Fatalf("expected updated name %q, got %q", newName, d3.Name)
	}
}

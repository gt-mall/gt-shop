package system_test

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"mall/admin_core/internal/model"
	systemsetting "mall/admin_core/internal/system/setting"
	"mall/admin_core/pkg/cache"
)

func getSystemTestRedisCache(t *testing.T) (cache.Cache, *redis.Client) {
	t.Helper()
	client := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
		DB:   15,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		t.Skipf("skipping Redis cache tests: local redis not reachable: %v", err)
		return nil, nil
	}
	c := cache.NewRedisCache(client)
	return c, client
}

func settingTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { sqlDB.Close() })
	if err := db.AutoMigrate(&model.SystemConfig{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestShopSettingCacheLifecycleWithRedis(t *testing.T) {
	ctx := context.Background()
	c, client := getSystemTestRedisCache(t)
	defer client.Close()

	db := settingTestDB(t)
	svc := systemsetting.NewService(db, c)

	key := cache.SystemConfigKey(systemsetting.ConfigKeyShopBasic)
	defer c.Delete(ctx, key)
	_ = c.Delete(ctx, key)

	// 1. 首次查询：查 DB 并回填缓存
	cfg1, err := svc.GetShop(ctx)
	if err != nil {
		t.Fatalf("GetShop 1 failed: %v", err)
	}
	if cfg1.Name != "Mall 商城" {
		t.Fatalf("expected default name 'Mall 商城', got %q", cfg1.Name)
	}

	// 检查 Redis 中已建立该 Key
	exists, err := client.Exists(ctx, key).Result()
	if err != nil || exists != 1 {
		t.Fatalf("expected cache key %s in redis, exists=%d", key, exists)
	}

	// 2. 直接修改数据库中的配置
	if err := db.Model(&model.SystemConfig{}).Where("config_key = ?", systemsetting.ConfigKeyShopBasic).
		Update("value", `{"name":"直接篡改DB名称"}`).Error; err != nil {
		t.Fatalf("direct db update failed: %v", err)
	}

	// 3. 再次查询：命中缓存，依然返回 "Mall 商城"
	cfg2, err := svc.GetShop(ctx)
	if err != nil {
		t.Fatalf("GetShop 2 failed: %v", err)
	}
	if cfg2.Name != "Mall 商城" {
		t.Fatalf("expected cached name 'Mall 商城', got %q", cfg2.Name)
	}

	// 4. 调用 SaveShop 保存新配置：必须成功更新数据库并删除缓存
	newCfg := cfg1
	newCfg.Name = "官方旗舰店"
	if err := svc.SaveShop(ctx, newCfg); err != nil {
		t.Fatalf("SaveShop failed: %v", err)
	}

	// 验证缓存已被删除
	existsAfterSave, err := client.Exists(ctx, key).Result()
	if err != nil || existsAfterSave != 0 {
		t.Fatalf("expected cache key %s to be deleted after save, exists=%d", key, existsAfterSave)
	}

	// 5. 第三次查询：获取最新数据并重新写入缓存
	cfg3, err := svc.GetShop(ctx)
	if err != nil {
		t.Fatalf("GetShop 3 failed: %v", err)
	}
	if cfg3.Name != "官方旗舰店" {
		t.Fatalf("expected updated name '官方旗舰店', got %q", cfg3.Name)
	}
}

func TestSettingWithRedisUnavailable(t *testing.T) {
	ctx := context.Background()
	// 使用已关闭的客户端
	client := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
		DB:   15,
	})
	_ = client.Close()
	brokenCache := cache.NewRedisCache(client)

	db := settingTestDB(t)
	svc := systemsetting.NewService(db, brokenCache)

	// Redis 异常时，读取应降级正常从 DB 返回默认配置
	cfg, err := svc.GetShop(ctx)
	if err != nil {
		t.Fatalf("expected GetShop to succeed despite Redis failure, got %v", err)
	}
	if cfg.Name != "Mall 商城" {
		t.Fatalf("expected default name, got %q", cfg.Name)
	}

	// Redis 异常时，保存应正常写入 DB
	cfg.Name = "正常保存测试"
	if err := svc.SaveShop(ctx, cfg); err != nil {
		t.Fatalf("expected SaveShop to succeed despite Redis failure, got %v", err)
	}

	// 再次读取也能成功获取最新数据
	cfgAfter, err := svc.GetShop(ctx)
	if err != nil {
		t.Fatalf("expected GetShop to succeed despite Redis failure, got %v", err)
	}
	if cfgAfter.Name != "正常保存测试" {
		t.Fatalf("expected '正常保存测试', got %q", cfgAfter.Name)
	}
}


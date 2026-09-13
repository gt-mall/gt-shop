package system_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"mall/admin_core/internal/model"
	systemsetting "mall/admin_core/internal/system/setting"
	"github.com/alicebob/miniredis/v2"
	"mall/admin_core/pkg/cache"
)

func getSystemTestRueidisCache(t *testing.T) (cache.Cache, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	c, err := cache.NewRueidisCache(cache.Config{
		Addr:         mr.Addr(),
		DisableCache: true,
		UseLuaLock:   true,
	})
	if err != nil {
		t.Fatalf("failed to create test rueidis cache: %v", err)
	}
	t.Cleanup(func() {
		_ = c.Close()
	})
	return c, mr
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
	c, mr := getSystemTestRueidisCache(t)

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
	if !mr.Exists(key) {
		t.Fatalf("expected cache key %s in redis", key)
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
	if mr.Exists(key) {
		t.Fatalf("expected cache key %s to be deleted after save", key)
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
	mr := miniredis.RunT(t)
	brokenCache, err := cache.NewRueidisCache(cache.Config{
		Addr:         mr.Addr(),
		DisableCache: true,
		UseLuaLock:   true,
	})
	if err != nil {
		t.Fatalf("failed to init cache: %v", err)
	}
	defer brokenCache.Close()

	// 模拟 Redis 宕机
	mr.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

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

func TestMpConfigSensitiveFieldsNeverInRedis(t *testing.T) {
	ctx := context.Background()
	c, mr := getSystemTestRueidisCache(t)

	db := settingTestDB(t)
	svc := systemsetting.NewService(db, c)

	publicMpKey := cache.PublicMpConfigKey()
	mpBasicKey := cache.SystemConfigKey(systemsetting.ConfigKeyMpBasic)
	defer c.Delete(ctx, publicMpKey, mpBasicKey)
	_ = c.Delete(ctx, publicMpKey, mpBasicKey)

	// 保存包含敏感秘钥的小程序配置
	mpConfig := systemsetting.MpConfig{
		Name:             "测试小程序",
		OriginalID:       "gh_123456",
		AppID:            "wx1234567890",
		AppSecret:        "super_secret_key_never_in_redis",
		UploadPrivateKey: "private_key_content_never_in_redis",
	}
	if err := svc.SaveMp(ctx, mpConfig); err != nil {
		t.Fatalf("SaveMp failed: %v", err)
	}

	// 1. 管理后台查询 GetMp 直接查 MySQL，绝不向 Redis 写入敏感配置
	adminMp, err := svc.GetMp(ctx)
	if err != nil {
		t.Fatalf("GetMp failed: %v", err)
	}
	if adminMp.AppSecret != "super_secret_key_never_in_redis" {
		t.Fatalf("unexpected app secret from DB: %s", adminMp.AppSecret)
	}

	// 验证 Redis 中不存在 mp_basic key
	if mr.Exists(mpBasicKey) {
		t.Fatalf("expected key %s not to exist in redis", mpBasicKey)
	}

	// 2. 前端/小程序公共配置 GetPublicMp 只返回非敏感字段
	pubMp, err := svc.GetPublicMp(ctx)
	if err != nil {
		t.Fatalf("GetPublicMp failed: %v", err)
	}
	if pubMp.Name != "测试小程序" || pubMp.OriginalID != "gh_123456" {
		t.Fatalf("unexpected public mp: %+v", pubMp)
	}

	// 验证 Redis 中存在公共配置缓存
	if !mr.Exists(publicMpKey) {
		t.Fatalf("expected key %s to exist in redis", publicMpKey)
	}

	// 检查 Redis 中该 key 的实际内容，确保绝对不包含敏感词
	redisVal, err := mr.Get(publicMpKey)
	if err != nil {
		t.Fatalf("get redis public mp failed: %v", err)
	}
	if strings.Contains(redisVal, "super_secret_key") || strings.Contains(redisVal, "private_key") {
		t.Fatalf("SECURITY VIOLATION: sensitive secrets leaked into redis cache: %s", redisVal)
	}

	// 3. 再次保存小程序配置，验证公共缓存被联动删除
	mpConfig.Name = "测试小程序V2"
	if err := svc.SaveMp(ctx, mpConfig); err != nil {
		t.Fatalf("SaveMp 2 failed: %v", err)
	}
	if mr.Exists(publicMpKey) {
		t.Fatalf("expected key %s to be deleted after SaveMp", publicMpKey)
	}
}

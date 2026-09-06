package test

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"mall/admin_core/internal/model"
	systemsetting "mall/admin_core/internal/system/setting"
)

func settingTestService(t *testing.T) (*systemsetting.Service, *gorm.DB) {
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
	service := systemsetting.NewService(db)
	return service, db
}

func TestShopBasicSettingLifecycle(t *testing.T) {
	ctx := context.Background()
	service, _ := settingTestService(t)

	// 1. 获取默认配置
	config, err := service.GetShop(ctx)
	if err != nil {
		t.Fatalf("GetShop failed: %v", err)
	}
	if config.Name != "Mall 商城" {
		t.Fatalf("expected default name 'Mall 商城', got %q", config.Name)
	}

	// 2. 保存新配置
	newConfig := systemsetting.ShopBasicConfig{
		Name:                  "我的测试商城",
		Logo:                  "/media/2026/09/logo.png",
		LogoID:                10,
		Favicon:               "/media/2026/09/favicon.ico",
		FaviconID:             11,
		ServiceName:           "专属客服小美",
		ServicePhone:          "400-888-9999",
		ServiceWechat:         "mall_service_vip",
		ServiceWechatQRCode:   "/media/2026/09/qr.png",
		ServiceWechatQRCodeID: 12,
		Description:           "专业电商测试平台",
	}
	if err := service.SaveShop(ctx, newConfig); err != nil {
		t.Fatalf("SaveShop failed: %v", err)
	}

	// 3. 读取验证
	saved, err := service.GetShop(ctx)
	if err != nil {
		t.Fatalf("GetShop after save failed: %v", err)
	}
	if saved.Name != newConfig.Name || saved.LogoID != 10 || saved.ServiceWechat != "mall_service_vip" {
		t.Fatalf("saved data mismatch: %+v", saved)
	}
}

func TestSiteBasicSettingLifecycle(t *testing.T) {
	ctx := context.Background()
	service, _ := settingTestService(t)

	// 1. 获取默认配置
	config, err := service.GetSite(ctx)
	if err != nil {
		t.Fatalf("GetSite failed: %v", err)
	}
	if config.Title != "Mall 管理后台" {
		t.Fatalf("expected default title 'Mall 管理后台', got %q", config.Title)
	}

	// 2. 保存新配置
	newConfig := systemsetting.SiteBasicConfig{
		Title:       "Mall 官方旗舰站",
		Keywords:    "电商,数码,服饰,Mall",
		Description: "一站式优质购物体验",
		ICP:         "浙ICP备12345678号-1",
		Copyright:   "© 2026 Mall Inc.",
	}
	if err := service.SaveSite(ctx, newConfig); err != nil {
		t.Fatalf("SaveSite failed: %v", err)
	}

	// 3. 读取验证
	saved, err := service.GetSite(ctx)
	if err != nil {
		t.Fatalf("GetSite after save failed: %v", err)
	}
	if saved.Title != newConfig.Title || saved.ICP != newConfig.ICP || saved.Copyright != newConfig.Copyright {
		t.Fatalf("saved data mismatch: %+v", saved)
	}
}

func TestShopBasicSettingFileResolution(t *testing.T) {
	ctx := context.Background()
	service, db := settingTestService(t)
	if err := db.AutoMigrate(&model.SystemFile{}); err != nil {
		t.Fatal(err)
	}
	f := model.SystemFile{
		ID:      101,
		Storage: "local",
		Name:    "logo.png",
		Path:    "2026/09/00000000-0000-0000-0000-000000000001.png",
		Ext:     "png",
	}
	if err := db.Create(&f).Error; err != nil {
		t.Fatal(err)
	}

	// 保存时仅传 logo_id: 101，不传 logo url
	if err := service.SaveShop(ctx, systemsetting.ShopBasicConfig{
		Name:   "自动解析商城",
		LogoID: 101,
	}); err != nil {
		t.Fatal(err)
	}

	// 读取时应自动解析出 Logo 的 URL
	saved, err := service.GetShop(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if saved.LogoID != 101 {
		t.Fatalf("expected LogoID 101, got %d", saved.LogoID)
	}
	if saved.Logo == "" {
		t.Fatal("expected Logo URL to be automatically resolved, got empty")
	}
}

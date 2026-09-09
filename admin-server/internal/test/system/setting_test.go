package system_test

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

func TestMpSettingLifecycle(t *testing.T) {
	ctx := context.Background()
	service, _ := settingTestService(t)

	// 1. 获取默认配置
	config, err := service.GetMp(ctx)
	if err != nil {
		t.Fatalf("GetMp failed: %v", err)
	}
	if config.Name != "Mall 小程序" {
		t.Fatalf("expected default name 'Mall 小程序', got %q", config.Name)
	}

	// 2. 保存新配置
	newConfig := systemsetting.MpConfig{
		Name:             "我的测试小程序",
		OriginalID:       "gh_123456789abc",
		QRCode:           "/media/2026/09/mp_qr.png",
		QRCodeID:         20,
		AppID:            "wx9876543210abcdef",
		AppSecret:        "0123456789abcdef0123456789abcdef",
		RequestDomain:    "https://api.mall.test",
		SocketDomain:     "wss://api.mall.test",
		UploadDomain:     "https://api.mall.test",
		DownloadDomain:   "https://res.mall.test",
		UDPDomain:        "udp.mall.test",
		TCPDomain:        "tcp.mall.test",
		BusinessDomain:   "https://m.mall.test",
		UploadPrivateKey: "-----BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAKCAQEA0...\n-----END RSA PRIVATE KEY-----",
	}
	if err := service.SaveMp(ctx, newConfig); err != nil {
		t.Fatalf("SaveMp failed: %v", err)
	}

	// 3. 读取验证
	saved, err := service.GetMp(ctx)
	if err != nil {
		t.Fatalf("GetMp after save failed: %v", err)
	}
	if saved.Name != newConfig.Name ||
		saved.OriginalID != newConfig.OriginalID ||
		saved.QRCodeID != newConfig.QRCodeID ||
		saved.AppID != newConfig.AppID ||
		saved.AppSecret != newConfig.AppSecret ||
		saved.RequestDomain != newConfig.RequestDomain ||
		saved.SocketDomain != newConfig.SocketDomain ||
		saved.UploadDomain != newConfig.UploadDomain ||
		saved.DownloadDomain != newConfig.DownloadDomain ||
		saved.UDPDomain != newConfig.UDPDomain ||
		saved.TCPDomain != newConfig.TCPDomain ||
		saved.BusinessDomain != newConfig.BusinessDomain ||
		saved.UploadPrivateKey != newConfig.UploadPrivateKey {
		t.Fatalf("saved mp data mismatch: %+v", saved)
	}
}

func TestMpSettingFileResolution(t *testing.T) {
	ctx := context.Background()
	service, db := settingTestService(t)
	if err := db.AutoMigrate(&model.SystemFile{}); err != nil {
		t.Fatal(err)
	}
	f := model.SystemFile{
		ID:      201,
		Storage: "local",
		Name:    "mp_qr.png",
		Path:    "2026/09/00000000-0000-0000-0000-000000000002.png",
		Ext:     "png",
	}
	if err := db.Create(&f).Error; err != nil {
		t.Fatal(err)
	}

	// 保存时仅传 qrcode_id: 201，不传 qrcode url
	if err := service.SaveMp(ctx, systemsetting.MpConfig{
		Name:     "自动解析小程序",
		QRCodeID: 201,
	}); err != nil {
		t.Fatal(err)
	}

	// 读取时应自动解析出 QRCode 的 URL
	saved, err := service.GetMp(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if saved.QRCodeID != 201 {
		t.Fatalf("expected QRCodeID 201, got %d", saved.QRCodeID)
	}
	if saved.QRCode == "" {
		t.Fatal("expected QRCode URL to be automatically resolved, got empty")
	}
}

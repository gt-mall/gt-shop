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

func TestNewSettingsLifecycles(t *testing.T) {
	ctx := context.Background()
	service, _ := settingTestService(t)

	// 1. Trade settings & validations
	tradeDef, err := service.GetTrade(ctx)
	if err != nil {
		t.Fatalf("GetTrade failed: %v", err)
	}
	if tradeDef.OrderCloseMinutes != 30 || tradeDef.AutoReceiveDays != 7 || tradeDef.AfterSaleDays != 7 ||
		!tradeDef.AllowCancelUnshipped || !tradeDef.AutoRefundOnCancel {
		t.Fatalf("unexpected trade default: %+v", tradeDef)
	}

	// Validation checks
	if err := service.SaveTrade(ctx, systemsetting.TradeBasicConfig{OrderCloseMinutes: 0}); err == nil {
		t.Fatal("expected error for OrderCloseMinutes 0")
	}
	if err := service.SaveTrade(ctx, systemsetting.TradeBasicConfig{OrderCloseMinutes: 1500}); err == nil {
		t.Fatal("expected error for OrderCloseMinutes 1500")
	}
	if err := service.SaveTrade(ctx, systemsetting.TradeBasicConfig{OrderCloseMinutes: 30, AutoReceiveDays: 0}); err == nil {
		t.Fatal("expected error for AutoReceiveDays 0")
	}
	if err := service.SaveTrade(ctx, systemsetting.TradeBasicConfig{OrderCloseMinutes: 30, AutoReceiveDays: 7, AfterSaleDays: -1}); err == nil {
		t.Fatal("expected error for AfterSaleDays -1")
	}

	validTrade := systemsetting.TradeBasicConfig{
		OrderCloseMinutes:    45,
		AutoReceiveDays:      14,
		AfterSaleDays:        10,
		AllowCancelUnshipped: true,
		AutoRefundOnCancel:   true,
	}
	if err := service.SaveTrade(ctx, validTrade); err != nil {
		t.Fatalf("SaveTrade failed: %v", err)
	}
	savedTrade, err := service.GetTrade(ctx)
	if err != nil {
		t.Fatalf("GetTrade after save failed: %v", err)
	}
	if savedTrade != validTrade {
		t.Fatalf("saved trade mismatch: %+v vs %+v", savedTrade, validTrade)
	}

	if err := service.SaveReview(ctx, systemsetting.ReviewBasicConfig{
		Enabled: true, AutoReviewEnabled: true, AutoReviewDays: 7,
	}); err == nil {
		t.Fatal("unimplemented automatic reviews must not be enabled")
	}

	// 2. Payment settings
	payDef, err := service.GetPayment(ctx)
	if err != nil {
		t.Fatalf("GetPayment failed: %v", err)
	}
	if !payDef.WechatEnabled {
		t.Fatalf("expected default payment wechat_enabled true")
	}
	if err := service.SavePayment(ctx, systemsetting.PaymentBasicConfig{WechatEnabled: false}); err != nil {
		t.Fatalf("SavePayment failed: %v", err)
	}
	savedPay, err := service.GetPayment(ctx)
	if err != nil || savedPay.WechatEnabled {
		t.Fatalf("expected wechat_enabled false, got %+v, err: %v", savedPay, err)
	}

	// 3. Logistics settings
	logisticsDef, err := service.GetLogistics(ctx)
	if err != nil {
		t.Fatalf("GetLogistics failed: %v", err)
	}
	if logisticsDef.SenderName != "" {
		t.Fatalf("expected empty default sender name")
	}
	validLogistics := systemsetting.LogisticsBasicConfig{
		SenderName:    "张三发货部",
		SenderPhone:   "13800000000",
		SenderAddress: "上海市静安区万航渡路88号",
	}
	if err := service.SaveLogistics(ctx, validLogistics); err != nil {
		t.Fatalf("SaveLogistics failed: %v", err)
	}
	savedLogistics, err := service.GetLogistics(ctx)
	if err != nil || savedLogistics != validLogistics {
		t.Fatalf("saved logistics mismatch: %+v vs %+v", savedLogistics, validLogistics)
	}

	// 4. Member settings
	memberDef, err := service.GetMember(ctx)
	if err != nil {
		t.Fatalf("GetMember failed: %v", err)
	}
	if !memberDef.RegisterEnabled || !memberDef.PasswordLoginEnabled || !memberDef.WechatLoginEnabled {
		t.Fatalf("expected all member defaults true, got %+v", memberDef)
	}
	validMember := systemsetting.MemberBasicConfig{
		RegisterEnabled:      false,
		PasswordLoginEnabled: true,
		WechatLoginEnabled:   true,
	}
	if err := service.SaveMember(ctx, validMember); err != nil {
		t.Fatalf("SaveMember failed: %v", err)
	}
	savedMember, err := service.GetMember(ctx)
	if err != nil || savedMember != validMember {
		t.Fatalf("saved member mismatch: %+v vs %+v", savedMember, validMember)
	}

	// 5. Review settings & validation
	reviewDef, err := service.GetReview(ctx)
	if err != nil {
		t.Fatalf("GetReview failed: %v", err)
	}
	if !reviewDef.Enabled {
		t.Fatalf("expected review enabled by default")
	}
	if err := service.SaveReview(ctx, systemsetting.ReviewBasicConfig{AutoReviewDays: 0}); err == nil {
		t.Fatal("expected error for AutoReviewDays 0")
	}
	validReview := systemsetting.ReviewBasicConfig{
		Enabled:           true,
		NeedAudit:         false,
		DefaultAnonymous:  true,
		AutoReviewEnabled: false,
		AutoReviewDays:    7,
	}
	if err := service.SaveReview(ctx, validReview); err != nil {
		t.Fatalf("SaveReview failed: %v", err)
	}
	savedReview, err := service.GetReview(ctx)
	if err != nil || savedReview != validReview {
		t.Fatalf("saved review mismatch: %+v vs %+v", savedReview, validReview)
	}

	// 6. Agreement settings & type resolution
	agreementDef, err := service.GetAgreement(ctx)
	if err != nil {
		t.Fatalf("GetAgreement failed: %v", err)
	}
	if agreementDef.UserAgreement != "" {
		t.Fatalf("expected empty user agreement by default")
	}
	validAgreement := systemsetting.AgreementBasicConfig{
		UserAgreement:   "<p>用户协议条款</p>",
		PrivacyPolicy:   "<p>隐私政策条款</p>",
		AfterSalePolicy: "<p>售后政策说明</p>",
	}
	if err := service.SaveAgreement(ctx, validAgreement); err != nil {
		t.Fatalf("SaveAgreement failed: %v", err)
	}
	savedAgreement, err := service.GetAgreement(ctx)
	if err != nil || savedAgreement != validAgreement {
		t.Fatalf("saved agreement mismatch: %+v vs %+v", savedAgreement, validAgreement)
	}

	// GetAgreementByType checks
	userResp, err := service.GetAgreementByType(ctx, "user")
	if err != nil || userResp.Content != "<p>用户协议条款</p>" || userResp.Title != "用户协议" {
		t.Fatalf("unexpected user agreement detail: %+v, err: %v", userResp, err)
	}
	privResp, err := service.GetAgreementByType(ctx, "privacy_policy")
	if err != nil || privResp.Content != "<p>隐私政策条款</p>" || privResp.Title != "隐私政策" {
		t.Fatalf("unexpected privacy detail: %+v, err: %v", privResp, err)
	}
	afterResp, err := service.GetAgreementByType(ctx, "after-sale")
	if err != nil || afterResp.Content != "<p>售后政策说明</p>" || afterResp.Title != "售后政策" {
		t.Fatalf("unexpected after-sale detail: %+v, err: %v", afterResp, err)
	}
	if _, err := service.GetAgreementByType(ctx, "unknown"); err == nil {
		t.Fatal("expected error for unknown agreement type")
	}
}

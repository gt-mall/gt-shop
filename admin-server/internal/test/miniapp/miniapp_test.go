package miniapp_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"mall/admin_core/bootstrap"
	"mall/admin_core/internal/auth"
	miniappcommon "mall/admin_core/internal/miniapp/common"
	"mall/admin_core/internal/model"
	systemsetting "mall/admin_core/internal/system/setting"
	"mall/admin_core/routers"
)

func TestMiniappHelloService(t *testing.T) {
	service := miniappcommon.NewService(nil)
	res := service.Hello()
	if res.Text != "Hello World" {
		t.Fatalf("want 'Hello World', got %q", res.Text)
	}
}

func TestMiniappHelloEndpoint(t *testing.T) {
	router := routers.NewRouter(bootstrap.Config{}, nil, auth.NewService(nil, nil, "", time.Hour))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/api/miniapp/hello", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != 200 {
		t.Fatalf("GET /api/miniapp/hello status = %d, want 200", recorder.Code)
	}
	var result struct {
		Code    int                         `json:"code"`
		Message string                      `json:"message"`
		Total   int                         `json:"total"`
		Data    miniappcommon.HelloResponse `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if result.Code != 1 || result.Data.Text != "Hello World" {
		t.Fatalf("unexpected payload: %+v", result)
	}
}

func TestMiniappBootstrapDefaultAndNilDB(t *testing.T) {
	service := miniappcommon.NewService(nil)
	res, err := service.Bootstrap(context.Background())
	if err != nil {
		t.Fatalf("Bootstrap with nil db failed: %v", err)
	}
	if res.Shop.Name != "Mall 商城" {
		t.Errorf("expected default shop name 'Mall 商城', got %q", res.Shop.Name)
	}
	if res.Site.Title != "Mall 管理后台" {
		t.Errorf("expected default site title 'Mall 管理后台', got %q", res.Site.Title)
	}
	if res.Mp.Name != "Mall 小程序" {
		t.Errorf("expected default mp name 'Mall 小程序', got %q", res.Mp.Name)
	}
	if len(res.Categories) != 0 {
		t.Errorf("expected 0 categories for nil db, got %d", len(res.Categories))
	}
}

func TestMiniappBootstrapWithDatabase(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { sqlDB.Close() })

	if err := db.AutoMigrate(&model.SystemConfig{}, &model.SystemFile{}, &model.ProductCategory{}); err != nil {
		t.Fatal(err)
	}

	// 1. 插入测试图片素材
	logoFile := model.SystemFile{
		ID:      101,
		Storage: "local",
		Name:    "logo.png",
		Path:    "2026/09/00000000-0000-0000-0000-000000000001.png",
		Ext:     "png",
	}
	catFile := model.SystemFile{
		ID:      102,
		Storage: "local",
		Name:    "cat.png",
		Path:    "2026/09/00000000-0000-0000-0000-000000000002.png",
		Ext:     "png",
	}
	if err := db.Create(&logoFile).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&catFile).Error; err != nil {
		t.Fatal(err)
	}

	// 2. 插入 shop_basic, site_basic, mp_basic 配置
	settingService := systemsetting.NewService(db)
	if err := settingService.SaveShop(ctx, systemsetting.ShopBasicConfig{
		Name:                "酷购商城",
		LogoID:              101,
		Description:         "全品类品质商城",
		ServiceName:         "在线客服",
		ServicePhone:        "400-123-4567",
		ServiceWechat:       "kugou_service",
		ServiceWechatQRCode: "",
	}); err != nil {
		t.Fatal(err)
	}
	if err := settingService.SaveSite(ctx, systemsetting.SiteBasicConfig{
		Title:       "酷购商城官网",
		Keywords:    "电商,数码,服装",
		Description: "正品低价商城",
		ICP:         "浙ICP备99999999号",
		Copyright:   "© 2026 Kugou Mall",
	}); err != nil {
		t.Fatal(err)
	}
	if err := settingService.SaveMp(ctx, systemsetting.MpConfig{
		Name:             "酷购小程序",
		OriginalID:       "gh_abcdef123456",
		QRCodeID:         101,
		AppID:            "wx_secret_appid",
		AppSecret:        "super_secret_key",
		UploadPrivateKey: "private_rsa_key_content",
	}); err != nil {
		t.Fatal(err)
	}

	// 3. 插入商品分类：部分启用、部分禁用、部分已软删除
	catImgID := int64(102)
	now := time.Now()
	categories := []model.ProductCategory{
		{ID: 1, ParentID: 0, Name: "数码数码", ImageID: &catImgID, Sort: 20, Status: 1},
		{ID: 2, ParentID: 0, Name: "热门服装", ImageID: nil, Sort: 10, Status: 1},
		{ID: 3, ParentID: 0, Name: "下架分类", ImageID: nil, Sort: 5, Status: 0},                                                    // status=0 应过滤
		{ID: 4, ParentID: 0, Name: "已删分类", ImageID: nil, Sort: 1, Status: 1, DeletedAt: gorm.DeletedAt{Time: now, Valid: true}}, // 已删除应过滤
	}
	for _, c := range categories {
		if err := db.Create(&c).Error; err != nil {
			t.Fatal(err)
		}
	}

	// 4. 调用 Bootstrap
	service := miniappcommon.NewService(db)
	resp, err := service.Bootstrap(ctx)
	if err != nil {
		t.Fatalf("Bootstrap failed: %v", err)
	}

	// 验证 shop 数据及图片解析
	if resp.Shop.Name != "酷购商城" {
		t.Errorf("expected shop name '酷购商城', got %q", resp.Shop.Name)
	}
	if resp.Shop.Logo == "" {
		t.Errorf("expected logo URL resolved, got empty")
	}
	if resp.Shop.ServiceName != "在线客服" || resp.Shop.ServicePhone != "400-123-4567" {
		t.Errorf("unexpected shop service: %+v", resp.Shop)
	}

	// 验证 site 数据
	if resp.Site.Title != "酷购商城官网" || resp.Site.ICP != "浙ICP备99999999号" {
		t.Errorf("unexpected site info: %+v", resp.Site)
	}

	// 验证 mp 数据及图片解析
	if resp.Mp.Name != "酷购小程序" || resp.Mp.OriginalID != "gh_abcdef123456" {
		t.Errorf("unexpected mp info: %+v", resp.Mp)
	}
	if resp.Mp.QRCode == "" {
		t.Errorf("expected qrcode URL resolved, got empty")
	}

	// 验证 categories：只返回 status=1 且未删除，排序为 sort ASC, id ASC
	if len(resp.Categories) != 2 {
		t.Fatalf("expected 2 active categories, got %d: %+v", len(resp.Categories), resp.Categories)
	}
	if resp.Categories[0].Name != "热门服装" || resp.Categories[0].Sort != 10 {
		t.Errorf("expected first category '热门服装', got %q (sort %d)", resp.Categories[0].Name, resp.Categories[0].Sort)
	}
	if resp.Categories[1].Name != "数码数码" || resp.Categories[1].Sort != 20 {
		t.Errorf("expected second category '数码数码', got %q (sort %d)", resp.Categories[1].Name, resp.Categories[1].Sort)
	}
	if resp.Categories[1].ImageURL == "" {
		t.Errorf("expected category 1 image_url resolved, got empty")
	}

	// 5. 验证 HTTP 接口端到端调用
	router := routers.NewRouter(bootstrap.Config{}, db, auth.NewService(db, nil, "", time.Hour))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/api/miniapp/bootstrap", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != 200 {
		t.Fatalf("GET /api/miniapp/bootstrap status = %d, want 200", recorder.Code)
	}

	var httpResult struct {
		Code    int                             `json:"code"`
		Message string                          `json:"message"`
		Total   int                             `json:"total"`
		Data    miniappcommon.BootstrapResponse `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &httpResult); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if httpResult.Code != 1 {
		t.Fatalf("expected code 1, got %d", httpResult.Code)
	}
	if httpResult.Data.Shop.Name != "酷购商城" {
		t.Errorf("expected shop name '酷购商城', got %q", httpResult.Data.Shop.Name)
	}
	if len(httpResult.Data.Categories) != 2 {
		t.Errorf("expected 2 categories from HTTP response, got %d", len(httpResult.Data.Categories))
	}
}

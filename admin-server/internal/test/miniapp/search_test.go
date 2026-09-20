package miniapp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"mall/admin_core/bootstrap"
	"mall/admin_core/internal/auth"
	"mall/admin_core/internal/model"
	systemfile "mall/admin_core/internal/system/file"
	systemsetting "mall/admin_core/internal/system/setting"
	"mall/admin_core/pkg/response"
	"mall/admin_core/routers"
)

func setupMiniappSearchTestDB(t *testing.T) (*gorm.DB, http.Handler, *systemsetting.Service) {
	t.Helper()
	gin.SetMode(gin.TestMode)
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
		&model.ProductSkuInventoryLog{},
		&model.Member{},
		&model.MemberAddress{},
		&model.ProductReview{},
		&model.ProductReviewImage{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	fileSvc := systemfile.NewService(db)
	cfg := systemfile.DefaultConfig()
	cfg.Local.Path = t.TempDir()
	if err := fileSvc.SaveConfig(context.Background(), cfg); err != nil {
		t.Fatalf("save file config: %v", err)
	}

	settingSvc := systemsetting.NewService(db, nil)
	authSvc := auth.NewService(nil, nil, "", time.Hour)
	router := routers.NewRouter(bootstrap.Config{}, db, authSvc)

	return db, router, settingSvc
}

func TestMiniappSearchConfig(t *testing.T) {
	_, router, settingSvc := setupMiniappSearchTestDB(t)

	// 1. 默认情况，应返回空列表
	req := httptest.NewRequest(http.MethodGet, "/api/miniapp/search/config", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var res1 struct {
		Code int                                      `json:"code"`
		Data systemsetting.MiniappSearchConfigResponse `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res1); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if res1.Code != 1 {
		t.Fatalf("expected code=1, got %d", res1.Code)
	}
	if len(res1.Data.HotKeywords) != 0 {
		t.Fatalf("expected empty hot_keywords, got %v", res1.Data.HotKeywords)
	}

	// 2. 配置热门词：含排序、禁用、重复与空白
	err := settingSvc.SaveSearch(context.Background(), systemsetting.SearchBasicConfig{
		HotKeywords: []systemsetting.HotKeywordItem{
			{Keyword: "苹果手机", Sort: 20, Enabled: true},
			{Keyword: "轻薄笔记本", Sort: 10, Enabled: true},
			{Keyword: "已下架商品", Sort: 5, Enabled: false}, // 禁用不应出现
			{Keyword: "机械键盘", Sort: 30, Enabled: true},
			{Keyword: "   轻薄笔记本   ", Sort: 15, Enabled: true}, // 重复词去重
		},
	})
	if err != nil {
		t.Fatalf("save search: %v", err)
	}

	// 3. 再次获取公开搜索配置
	req = httptest.NewRequest(http.MethodGet, "/api/miniapp/search/config", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var res2 struct {
		Code int                                      `json:"code"`
		Data systemsetting.MiniappSearchConfigResponse `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if res2.Code != 1 {
		t.Fatalf("expected code 1, got %d", res2.Code)
	}

	// 验证排序：轻薄笔记本 (sort=10) -> 苹果手机 (sort=20) -> 机械键盘 (sort=30)
	expected := []string{"轻薄笔记本", "苹果手机", "机械键盘"}
	if len(res2.Data.HotKeywords) != len(expected) {
		t.Fatalf("expected len %d, got %d: %v", len(expected), len(res2.Data.HotKeywords), res2.Data.HotKeywords)
	}
	for i, v := range expected {
		if res2.Data.HotKeywords[i] != v {
			t.Errorf("at index %d: expected %s, got %s", i, v, res2.Data.HotKeywords[i])
		}
	}
}

func TestMiniappProductSearchLikeEscaping(t *testing.T) {
	db, router, _ := setupMiniappSearchTestDB(t)

	// 创建测试商品：
	// P1: 含有 "%" 的特殊商品
	// P2: 不含有 "%" 但名字含有普通字符的商品
	// P3: 含有 "_" 的特殊商品
	// P4: 名字有普通字符的商品
	products := []model.Product{
		{
			ID:         1,
			CategoryID: 1,
			Name:       "100% Pure Cotton T-Shirt",
			Subtitle:   "Best quality cotton",
			Status:     1, // 上架
			Sort:       1,
		},
		{
			ID:         2,
			CategoryID: 1,
			Name:       "Cotton Blend Shirt",
			Subtitle:   "Regular shirt",
			Status:     1,
			Sort:       2,
		},
		{
			ID:         3,
			CategoryID: 1,
			Name:       "Item_Special_SKU",
			Subtitle:   "Underline test",
			Status:     1,
			Sort:       3,
		},
		{
			ID:         4,
			CategoryID: 1,
			Name:       "ItemXSpecialYSKU",
			Subtitle:   "Matches underscore if unescaped",
			Status:     1,
			Sort:       4,
		},
	}

	for _, p := range products {
		if err := db.Create(&p).Error; err != nil {
			t.Fatalf("create product: %v", err)
		}
		sku := model.ProductSKU{
			ID:          p.ID * 10,
			ProductID:   p.ID,
			SpecValues:  "[]",
			Price:       "20.00",
			MarketPrice: "30.00",
			CostPrice:   "10.00",
			Status:      1,
		}
		if err := db.Create(&sku).Error; err != nil {
			t.Fatalf("create sku: %v", err)
		}
	}

	// 1. 搜索 "%"，在转义后只应匹配 p1（"100% Pure Cotton T-Shirt"），不应匹配全部商品！
	req := httptest.NewRequest(http.MethodGet, "/api/miniapp/products?keyword=%25", nil) // %25 is URL encoded '%'
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var res1 struct {
		Code  int               `json:"code"`
		Data  []json.RawMessage `json:"data"`
		Total int               `json:"total"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res1); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if res1.Total != 1 {
		t.Fatalf("expected total 1 for '%%' search, got %d", res1.Total)
	}

	// 2. 搜索 "Item_Special"，转义后只应匹配 p3，不应通过下划线通配符匹配 p4
	req = httptest.NewRequest(http.MethodGet, "/api/miniapp/products?keyword=Item_Special", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var res2 struct {
		Code  int               `json:"code"`
		Data  []json.RawMessage `json:"data"`
		Total int               `json:"total"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if res2.Total != 1 {
		t.Fatalf("expected total 1 for 'Item_Special' search, got %d", res2.Total)
	}

	// 3. 超长关键词截断：输入 60 字符，不应发生 SQL 错误或 panic
	longKw := fmt.Sprintf("Cotton%060d", 1)
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/miniapp/products?keyword=%s", longKw), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for long keyword, got %d", w.Code)
	}
}

func TestAdminSystemSettingSearch(t *testing.T) {
	_, router, _ := setupMiniappSearchTestDB(t)

	// 未授权请求管理端接口应被拦截 (code = -1)
	req := httptest.NewRequest(http.MethodGet, "/api/system/settings/search", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var res1 struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res1); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if res1.Code != -1 {
		t.Fatalf("expected code -1 without auth, got %d", res1.Code)
	}

	body, _ := json.Marshal(systemsetting.SearchBasicConfig{
		HotKeywords: []systemsetting.HotKeywordItem{
			{Keyword: "测试词", Sort: 1, Enabled: true},
		},
	})
	req = httptest.NewRequest(http.MethodPost, "/api/system/settings/search/save", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var res2 struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if res2.Code != -1 {
		t.Fatalf("expected code -1 for save without auth, got %d", res2.Code)
	}
}

// 保证 response 包被引用
var _ = response.Success

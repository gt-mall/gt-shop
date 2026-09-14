package miniapp_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"mall/admin_core/bootstrap"
	"mall/admin_core/internal/auth"
	miniappproduct "mall/admin_core/internal/miniapp/product"
	"mall/admin_core/internal/model"
	systemfile "mall/admin_core/internal/system/file"
	"mall/admin_core/routers"

	"github.com/gin-gonic/gin"
)

func setupMiniappProductTestDB(t *testing.T) (*gorm.DB, http.Handler) {
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
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	fileSvc := systemfile.NewService(db)
	cfg := systemfile.DefaultConfig()
	cfg.Local.Path = t.TempDir()
	if err := fileSvc.SaveConfig(context.Background(), cfg); err != nil {
		t.Fatalf("save file config: %v", err)
	}

	router := routers.NewRouter(bootstrap.Config{}, db, auth.NewService(nil, nil, "", time.Hour))
	return db, router
}

func createTestFile(t *testing.T, db *gorm.DB, id int64, filename string) int64 {
	t.Helper()
	f := model.SystemFile{
		ID:      id,
		GroupID: 0,
		Storage: "local",
		Name:    filename,
		Path:    fmt.Sprintf("2026/09/%08d-0000-0000-0000-000000000001.png", id),
		Ext:     "png",
		Size:    1024,
		Width:   100,
		Height:  100,
	}
	if err := db.Create(&f).Error; err != nil {
		t.Fatalf("create test file: %v", err)
	}
	return f.ID
}

// TestMiniappProductList_BasicPagination tests 1. 正常分页 and 2. page_size 生效
func TestMiniappProductList_BasicPagination(t *testing.T) {
	db, router := setupMiniappProductTestDB(t)

	// Create 5 active products with valid SKUs
	for i := 1; i <= 5; i++ {
		p := model.Product{
			ID:         int64(i),
			CategoryID: 1,
			Name:       fmt.Sprintf("商品%d", i),
			Subtitle:   fmt.Sprintf("副标题%d", i),
			Status:     1,
			Sort:       i,
		}
		db.Create(&p)
		sku := model.ProductSKU{
			ID:          int64(i * 10),
			ProductID:   int64(i),
			SpecValues:  "[]",
			Price:       "20.00",
			MarketPrice: "30.00",
			CostPrice:   "10.00",
			Status:      1,
		}
		db.Create(&sku)
	}

	// Page 1, PageSize 2
	req := httptest.NewRequest("GET", "/api/miniapp/products?page=1&page_size=2", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var res struct {
		Code    int                              `json:"code"`
		Message string                           `json:"message"`
		Total   int                              `json:"total"`
		Data    []miniappproduct.ProductListItem `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if res.Code != 1 || res.Total != 5 || len(res.Data) != 2 {
		t.Fatalf("expected total=5, len=2, got total=%d, len=%d", res.Total, len(res.Data))
	}

	// Page 2, PageSize 2
	req2 := httptest.NewRequest("GET", "/api/miniapp/products?page=2&page_size=2", nil)
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)
	var res2 struct {
		Code  int                              `json:"code"`
		Total int                              `json:"total"`
		Data  []miniappproduct.ProductListItem `json:"data"`
	}
	_ = json.Unmarshal(rec2.Body.Bytes(), &res2)
	if len(res2.Data) != 2 {
		t.Fatalf("expected page 2 len=2, got %d", len(res2.Data))
	}
	if res.Data[0].ID == res2.Data[0].ID {
		t.Fatalf("page 1 and page 2 returned same product ID %d", res.Data[0].ID)
	}

	// Max page_size cap test (>100 should be capped to 100)
	svc := miniappproduct.NewService(db)
	items, _, err := svc.List(context.Background(), miniappproduct.ProductListRequest{Page: 1, PageSize: 200})
	if err != nil {
		t.Fatalf("List with page_size=200 failed: %v", err)
	}
	if len(items) != 5 {
		t.Fatalf("expected 5 items, got %d", len(items))
	}
}

// TestMiniappProductList_KeywordAndCategory tests 3. keyword 搜索 name, 4. keyword 搜索 subtitle, 5. category_id 筛选
func TestMiniappProductList_KeywordAndCategory(t *testing.T) {
	db, router := setupMiniappProductTestDB(t)

	p1 := model.Product{ID: 1, CategoryID: 10, Name: "苹果手机 iPhone 15", Subtitle: "超强动力", Status: 1, Sort: 1}
	p2 := model.Product{ID: 2, CategoryID: 20, Name: "华为 Mate 60", Subtitle: "遥遥领先", Status: 1, Sort: 2}
	db.Create(&p1)
	db.Create(&p2)
	db.Create(&model.ProductSKU{ID: 11, ProductID: 1, SpecValues: "[]", Price: "5999.00", Status: 1})
	db.Create(&model.ProductSKU{ID: 21, ProductID: 2, SpecValues: "[]", Price: "6999.00", Status: 1})

	// 3. keyword match name
	req := httptest.NewRequest("GET", "/api/miniapp/products?keyword=iPhone", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	var res struct {
		Total int                              `json:"total"`
		Data  []miniappproduct.ProductListItem `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	if res.Total != 1 || len(res.Data) != 1 || res.Data[0].ID != 1 {
		t.Fatalf("keyword name search failed: %+v", res)
	}

	// 4. keyword match subtitle
	req = httptest.NewRequest("GET", "/api/miniapp/products?keyword=遥遥领先", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	if res.Total != 1 || len(res.Data) != 1 || res.Data[0].ID != 2 {
		t.Fatalf("keyword subtitle search failed: %+v", res)
	}

	// 5. category_id filter
	req = httptest.NewRequest("GET", "/api/miniapp/products?category_id=10", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	if res.Total != 1 || len(res.Data) != 1 || res.Data[0].ID != 1 {
		t.Fatalf("category_id search failed: %+v", res)
	}
}

// TestMiniappProductList_Sortings tests 6. default, 7. sales, 8. price_asc, 9. price_desc
func TestMiniappProductList_Sortings(t *testing.T) {
	db, router := setupMiniappProductTestDB(t)

	// Product A: sort=2, sales=10+5=15, min_price=99.00
	pA := model.Product{ID: 1, CategoryID: 1, Name: "商品A", Status: 1, Sort: 2, SalesCount: 10, VirtualSales: 5}
	// Product B: sort=1, sales=2+1=3, min_price=19.90
	pB := model.Product{ID: 2, CategoryID: 1, Name: "商品B", Status: 1, Sort: 1, SalesCount: 2, VirtualSales: 1}
	// Product C: sort=3, sales=20+10=30, min_price=49.00
	pC := model.Product{ID: 3, CategoryID: 1, Name: "商品C", Status: 1, Sort: 3, SalesCount: 20, VirtualSales: 10}

	db.Create(&pA)
	db.Create(&pB)
	db.Create(&pC)

	db.Create(&model.ProductSKU{ID: 101, ProductID: 1, SpecValues: "[]", Price: "99.00", Status: 1})
	db.Create(&model.ProductSKU{ID: 201, ProductID: 2, SpecValues: "[]", Price: "19.90", Status: 1})
	db.Create(&model.ProductSKU{ID: 301, ProductID: 3, SpecValues: "[]", Price: "49.00", Status: 1})

	// 6. default: sort ASC, id DESC -> B(1), A(2), C(3)
	req := httptest.NewRequest("GET", "/api/miniapp/products?sort=default", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	var res struct {
		Data []miniappproduct.ProductListItem `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	if len(res.Data) != 3 || res.Data[0].ID != 2 || res.Data[1].ID != 1 || res.Data[2].ID != 3 {
		t.Fatalf("default sort order wrong: %v, want [2, 1, 3]", []int64{res.Data[0].ID, res.Data[1].ID, res.Data[2].ID})
	}

	// 7. sales: sales DESC -> C(30), A(15), B(3)
	req = httptest.NewRequest("GET", "/api/miniapp/products?sort=sales", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	if len(res.Data) != 3 || res.Data[0].ID != 3 || res.Data[1].ID != 1 || res.Data[2].ID != 2 {
		t.Fatalf("sales sort order wrong: %v, want [3, 1, 2]", []int64{res.Data[0].ID, res.Data[1].ID, res.Data[2].ID})
	}

	// 8. price_asc: min_price ASC -> B(19.90), C(49.00), A(99.00)
	req = httptest.NewRequest("GET", "/api/miniapp/products?sort=price_asc", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	if len(res.Data) != 3 || res.Data[0].ID != 2 || res.Data[1].ID != 3 || res.Data[2].ID != 1 {
		t.Fatalf("price_asc sort order wrong: %v, want [2, 3, 1]", []int64{res.Data[0].ID, res.Data[1].ID, res.Data[2].ID})
	}

	// 9. price_desc: min_price DESC -> A(99.00), C(49.00), B(19.90)
	req = httptest.NewRequest("GET", "/api/miniapp/products?sort=price_desc", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	if len(res.Data) != 3 || res.Data[0].ID != 1 || res.Data[1].ID != 3 || res.Data[2].ID != 2 {
		t.Fatalf("price_desc sort order wrong: %v, want [1, 3, 2]", []int64{res.Data[0].ID, res.Data[1].ID, res.Data[2].ID})
	}
}

// TestMiniappProductList_FilterOffShelfAndDeleted tests 10. 下架不返回, 11. 删除不返回, 12. 无有效SKU不返回
func TestMiniappProductList_FilterOffShelfAndDeleted(t *testing.T) {
	db, router := setupMiniappProductTestDB(t)
	now := time.Now()

	// Normal valid product
	pValid := model.Product{ID: 1, Name: "正常商品", Status: 1}
	db.Create(&pValid)
	db.Create(&model.ProductSKU{ID: 10, ProductID: 1, SpecValues: "[]", Price: "10.00", Status: 1})

	// 10. Status = 0 (下架)
	pOff := model.Product{ID: 2, Name: "下架商品", Status: 0}
	db.Create(&pOff)
	db.Create(&model.ProductSKU{ID: 20, ProductID: 2, SpecValues: "[]", Price: "10.00", Status: 1})

	// 11. Deleted product (软删除)
	pDel := model.Product{ID: 3, Name: "删除商品", Status: 1, DeletedAt: gorm.DeletedAt{Time: now, Valid: true}}
	db.Create(&pDel)
	db.Create(&model.ProductSKU{ID: 30, ProductID: 3, SpecValues: "[]", Price: "10.00", Status: 1})

	// 12a. Product with no SKUs at all
	pNoSKU := model.Product{ID: 4, Name: "无SKU商品", Status: 1}
	db.Create(&pNoSKU)

	// 12b. Product with SKUs all disabled (status = 0)
	pDisabledSKU := model.Product{ID: 5, Name: "SKU全下架商品", Status: 1}
	db.Create(&pDisabledSKU)
	db.Create(&model.ProductSKU{ID: 50, ProductID: 5, SpecValues: "[]", Price: "10.00", Status: 0})

	// 12c. Product with SKUs all deleted
	pDelSKU := model.Product{ID: 6, Name: "SKU全删除商品", Status: 1}
	db.Create(&pDelSKU)
	db.Create(&model.ProductSKU{ID: 60, ProductID: 6, SpecValues: "[]", Price: "10.00", Status: 1, DeletedAt: gorm.DeletedAt{Time: now, Valid: true}})

	req := httptest.NewRequest("GET", "/api/miniapp/products", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var res struct {
		Total int                              `json:"total"`
		Data  []miniappproduct.ProductListItem `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &res)

	if res.Total != 1 || len(res.Data) != 1 || res.Data[0].ID != 1 {
		t.Fatalf("expected only product 1, got total=%d, items=%+v", res.Total, res.Data)
	}
}

// TestMiniappProduct_PriceStockAndImages tests:
// 13. 最低 SKU 价格正确
// 14. available stock = stock - locked_stock
// 15. 库存不能返回负数
// 16. main/gallery/SKU 图片 URL 正确
// 20. 绝不能暴露 cost_price
func TestMiniappProduct_PriceStockAndImages(t *testing.T) {
	db, router := setupMiniappProductTestDB(t)

	img1 := createTestFile(t, db, 101, "main.png")
	img2 := createTestFile(t, db, 102, "gallery1.png")
	img3 := createTestFile(t, db, 103, "sku1.png")

	galleryJSON := fmt.Sprintf("[%d]", img2)
	specsJSON := `[{"id":1,"name":"颜色","values":[{"id":11,"name":"红"},{"id":12,"name":"蓝"}]}]`
	paramsJSON := `[{"name":"材质","value":"纯棉"}]`
	detailText := "<p>商品详情内容</p>"

	p := model.Product{
		ID:             1,
		CategoryID:     5,
		Name:           "高级T恤",
		Subtitle:       "透气舒适",
		MainImageID:    &img1,
		GalleryFileIDs: &galleryJSON,
		Specs:          &specsJSON,
		Params:         &paramsJSON,
		Detail:         &detailText,
		Status:         1,
		Sort:           1,
		SalesCount:     50,
		VirtualSales:   20,
	}
	db.Create(&p)

	// SKU 1: Price=19.90, MarketPrice=39.90, CostPrice=9.90, Image=img3, Stock=100, Locked=20 -> Available=80
	unitPiece := "件"
	weight200 := 200
	sku1 := model.ProductSKU{
		ID:          1,
		ProductID:   1,
		SpecValues:  `[{"spec_id":1,"value_id":11}]`,
		Price:       "19.90",
		MarketPrice: "39.90",
		CostPrice:   "9.90",
		ImageID:     &img3,
		Unit:        &unitPiece,
		WeightGram:  &weight200,
		Sort:        1,
		Status:      1,
	}
	// SKU 2: Price=29.90, MarketPrice=49.90, CostPrice=15.90, Stock=5, Locked=10 -> Available=0 (not negative!)
	sku2 := model.ProductSKU{
		ID:          2,
		ProductID:   1,
		SpecValues:  `[{"spec_id":1,"value_id":12}]`,
		Price:       "29.90",
		MarketPrice: "49.90",
		CostPrice:   "15.90",
		Sort:        2,
		Status:      1,
	}
	db.Create(&sku1)
	db.Create(&sku2)

	// Inventory
	db.Create(&model.ProductSkuInventory{ProductID: 1, SKUID: 1, Stock: 100, LockedStock: 20})
	db.Create(&model.ProductSkuInventory{ProductID: 1, SKUID: 2, Stock: 5, LockedStock: 10})

	// === List Verification ===
	reqList := httptest.NewRequest("GET", "/api/miniapp/products", nil)
	recList := httptest.NewRecorder()
	router.ServeHTTP(recList, reqList)

	bodyList := recList.Body.String()
	// 20. cost_price must never be exposed
	if strings.Contains(bodyList, "cost_price") {
		t.Fatalf("cost_price found in product list response: %s", bodyList)
	}

	var resList struct {
		Code int                              `json:"code"`
		Data []miniappproduct.ProductListItem `json:"data"`
	}
	_ = json.Unmarshal([]byte(bodyList), &resList)
	if len(resList.Data) != 1 {
		t.Fatalf("expected 1 product in list, got %d", len(resList.Data))
	}
	item := resList.Data[0]

	// 13. 最低 SKU 价格: 19.90, market_price: 39.90
	if item.Price != "19.90" {
		t.Errorf("expected list price '19.90', got %q", item.Price)
	}
	if item.MarketPrice != "39.90" {
		t.Errorf("expected list market_price '39.90', got %q", item.MarketPrice)
	}

	// 14 & 15. Available stock: sum(80 + 0) = 80
	if item.Stock != 80 {
		t.Errorf("expected list stock 80, got %d", item.Stock)
	}

	// 16. Main image URL
	if !strings.Contains(item.MainImageURL, "00000101") {
		t.Errorf("expected 00000101 in image url, got %q", item.MainImageURL)
	}

	// Sales = 50 + 20 = 70
	if item.Sales != 70 {
		t.Errorf("expected sales 70, got %d", item.Sales)
	}

	// === Detail Verification ===
	reqDetail := httptest.NewRequest("GET", "/api/miniapp/products/1", nil)
	recDetail := httptest.NewRecorder()
	router.ServeHTTP(recDetail, reqDetail)

	bodyDetail := recDetail.Body.String()
	// 20. cost_price must never be exposed
	if strings.Contains(bodyDetail, "cost_price") {
		t.Fatalf("cost_price found in product detail response: %s", bodyDetail)
	}
	// Check sku_code and barcode not exposed
	if strings.Contains(bodyDetail, "sku_code") || strings.Contains(bodyDetail, "barcode") {
		t.Fatalf("sku_code or barcode found in detail response: %s", bodyDetail)
	}

	var resDetail struct {
		Code int                                  `json:"code"`
		Data miniappproduct.ProductDetailResponse `json:"data"`
	}
	_ = json.Unmarshal([]byte(bodyDetail), &resDetail)
	detail := resDetail.Data

	// 16. Images
	if !strings.Contains(detail.MainImageURL, "00000101") {
		t.Errorf("detail main image mismatch: %q", detail.MainImageURL)
	}
	if len(detail.GalleryURLs) != 1 || !strings.Contains(detail.GalleryURLs[0], "00000102") {
		t.Errorf("detail gallery URLs mismatch: %v", detail.GalleryURLs)
	}

	// 17. Specs
	if len(detail.Specs) != 1 || detail.Specs[0].Name != "颜色" || len(detail.Specs[0].Values) != 2 {
		t.Errorf("detail specs mismatch: %+v", detail.Specs)
	}

	// 18. SKU spec_values & 14, 15 SKU stocks
	if len(detail.SKUs) != 2 {
		t.Fatalf("expected 2 skus, got %d", len(detail.SKUs))
	}
	// SKU 1: stock=80
	if detail.SKUs[0].Stock != 80 {
		t.Errorf("sku1 stock expected 80, got %d", detail.SKUs[0].Stock)
	}
	if len(detail.SKUs[0].SpecValues) != 1 || detail.SKUs[0].SpecValues[0].ValueID != 11 {
		t.Errorf("sku1 spec_values mismatch: %+v", detail.SKUs[0].SpecValues)
	}
	if !strings.Contains(detail.SKUs[0].ImageURL, "00000103") {
		t.Errorf("sku1 image url mismatch: %q", detail.SKUs[0].ImageURL)
	}

	// SKU 2: stock=0 (max(5-10, 0))
	if detail.SKUs[1].Stock != 0 {
		t.Errorf("sku2 stock expected 0, got %d", detail.SKUs[1].Stock)
	}

	// 19. Params and Detail
	if len(detail.Params) != 1 || detail.Params[0].Name != "材质" || detail.Params[0].Value != "纯棉" {
		t.Errorf("detail params mismatch: %+v", detail.Params)
	}
	if detail.Detail != "<p>商品详情内容</p>" {
		t.Errorf("detail text mismatch: %q", detail.Detail)
	}
}

// TestMiniappProductDetail_NotFoundOrOffShelf tests 21. 不存在/下架商品详情不可访问
func TestMiniappProductDetail_NotFoundOrOffShelf(t *testing.T) {
	db, router := setupMiniappProductTestDB(t)

	// Off-shelf product
	db.Create(&model.Product{ID: 2, Name: "已下架", Status: 0})
	db.Create(&model.ProductSKU{ID: 20, ProductID: 2, SpecValues: "[]", Price: "10.00", Status: 1})

	// Deleted product
	db.Create(&model.Product{ID: 3, Name: "已删除", Status: 1, DeletedAt: gorm.DeletedAt{Time: time.Now(), Valid: true}})
	db.Create(&model.ProductSKU{ID: 30, ProductID: 3, SpecValues: "[]", Price: "10.00", Status: 1})

	// Product with no active SKU
	db.Create(&model.Product{ID: 4, Name: "无有效SKU", Status: 1})
	db.Create(&model.ProductSKU{ID: 40, ProductID: 4, SpecValues: "[]", Price: "10.00", Status: 0})

	testCases := []struct {
		id int64
	}{
		{id: 999}, // Non-existent
		{id: 2},   // Status 0
		{id: 3},   // Deleted
		{id: 4},   // No active SKU
	}

	for _, tc := range testCases {
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/miniapp/products/%d", tc.id), nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		var res struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &res)

		if res.Code != 0 || res.Message != "商品不存在或已下架" {
			t.Errorf("id %d expected code=0 and '商品不存在或已下架', got code=%d message=%q", tc.id, res.Code, res.Message)
		}
	}
}

// TestMiniappProductList_NoNPlusOneQueries tests that list does not execute N+1 queries
func TestMiniappProductList_NoNPlusOneQueries(t *testing.T) {
	db, _ := setupMiniappProductTestDB(t)

	// Create 20 products, each with 2 SKUs
	for i := 1; i <= 20; i++ {
		p := model.Product{
			ID:         int64(i),
			CategoryID: 1,
			Name:       fmt.Sprintf("商品%d", i),
			Status:     1,
			Sort:       i,
		}
		db.Create(&p)
		db.Create(&model.ProductSKU{
			ID:         int64(i*10 + 1),
			ProductID:  int64(i),
			SpecValues: "[]",
			Price:      "10.00",
			Status:     1,
		})
		db.Create(&model.ProductSKU{
			ID:         int64(i*10 + 2),
			ProductID:  int64(i),
			SpecValues: "[]",
			Price:      "20.00",
			Status:     1,
		})
		db.Create(&model.ProductSkuInventory{
			ProductID:   int64(i),
			SKUID:       int64(i*10 + 1),
			Stock:       50,
			LockedStock: 5,
		})
	}

	// Count queries executed during List
	var queryCount int32
	sqlLogger := &queryCountLogger{count: &queryCount}
	dbWithLogger := db.Session(&gorm.Session{
		Logger: sqlLogger,
	})

	svc := miniappproduct.NewService(dbWithLogger)
	items, total, err := svc.List(context.Background(), miniappproduct.ProductListRequest{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("List error: %v", err)
	}

	if total != 20 || len(items) != 20 {
		t.Fatalf("expected 20 items, got total=%d len=%d", total, len(items))
	}

	// For 20 products:
	// Query 1: COUNT DISTINCT
	// Query 2: SELECT products
	// Query 3: Batch SELECT product_sku WHERE product_id IN (?)
	// Query 4: Batch SELECT product_sku_inventory WHERE sku_id IN (?)
	// (Query 5: Batch resolve images if images exist)
	// Total queries must be <= 5, NOT 20+ (N+1)!
	if queryCount > 5 {
		t.Fatalf("expected <= 5 queries (constant batch), got %d queries (N+1 detected!)", queryCount)
	}
}

type queryCountLogger struct {
	count *int32
}

func (l *queryCountLogger) LogMode(logger.LogLevel) logger.Interface      { return l }
func (l *queryCountLogger) Info(context.Context, string, ...interface{})  {}
func (l *queryCountLogger) Warn(context.Context, string, ...interface{})  {}
func (l *queryCountLogger) Error(context.Context, string, ...interface{}) {}
func (l *queryCountLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	atomic.AddInt32(l.count, 1)
}

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

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"mall/admin_core/bootstrap"
	"mall/admin_core/internal/auth"
	miniappauth "mall/admin_core/internal/miniapp/auth"
	miniappcart "mall/admin_core/internal/miniapp/cart"
	miniapporder "mall/admin_core/internal/miniapp/order"
	"mall/admin_core/internal/model"
	systemfile "mall/admin_core/internal/system/file"
	systemsetting "mall/admin_core/internal/system/setting"
	"mall/admin_core/pkg/response"
	"mall/admin_core/routers"

	"github.com/gin-gonic/gin"
)

const cartTestJWTSecret = "cart_test_jwt_secret_key_1234567890"

func setupCartTestApp(t *testing.T) (*gorm.DB, http.Handler, string, int64, string, int64) {
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
		&model.ProductCategory{},
		&model.Product{},
		&model.ProductSKU{},
		&model.ProductSkuInventory{},
		&model.ProductSkuInventoryLog{},
		&model.Member{},
		&model.MemberAuth{},
		&model.MemberAddress{},
		&model.MemberAccount{},
		&model.OrderMain{},
		&model.OrderItem{},
		&model.OrderAddress{},
		&model.OrderLog{},
		&model.CartItem{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	fileSvc := systemfile.NewService(db)
	cfg := systemfile.DefaultConfig()
	cfg.Local.Path = t.TempDir()
	if err := fileSvc.SaveConfig(context.Background(), cfg); err != nil {
		t.Fatalf("save file config: %v", err)
	}

	settingService := systemsetting.NewService(db)
	_ = settingService.SaveMp(context.Background(), systemsetting.MpConfig{
		Name:      "Test MP",
		AppID:     "wx_test_appid",
		AppSecret: "wx_test_secret",
	})

	authSvc := miniappauth.NewService(db, cartTestJWTSecret, 24*time.Hour)
	mockClient := &mockWeChatClient{
		openID:     "openid_cart_user_1",
		sessionKey: "session_key_1",
	}
	authSvc.SetWeChatClient(mockClient)
	res1, err := authSvc.WechatLogin(context.Background(), "code1")
	if err != nil {
		t.Fatalf("login member 1 failed: %v", err)
	}

	mockClient2 := &mockWeChatClient{
		openID:     "openid_cart_user_2",
		sessionKey: "session_key_2",
	}
	authSvc.SetWeChatClient(mockClient2)
	res2, err := authSvc.WechatLogin(context.Background(), "code2")
	if err != nil {
		t.Fatalf("login member 2 failed: %v", err)
	}

	bConfig := bootstrap.Config{
		JWTSecret:     cartTestJWTSecret,
		JWTExpireHour: 24,
	}
	router := routers.NewRouter(bConfig, db, auth.NewService(nil, nil, "", time.Hour))
	return db, router, res1.Token, res1.Member.ID, res2.Token, res2.Member.ID
}

func createProductWithSKUs(t *testing.T, db *gorm.DB, prodStatus int8, skuStatus int8, stock int, lockedStock int) (int64, int64) {
	t.Helper()
	specsJSON := `[{"id":1,"name":"颜色","values":[{"id":101,"name":"曜石黑"},{"id":102,"name":"雪域白"}]}]`
	prod := model.Product{
		CategoryID:   1,
		Name:         "测试手机",
		Subtitle:     "旗舰性能",
		Specs:        &specsJSON,
		Status:       prodStatus,
		Sort:         1,
		SalesCount:   10,
		VirtualSales: 5,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := db.Create(&prod).Error; err != nil {
		t.Fatalf("create product: %v", err)
	}

	skuSpecValues := `[{"spec_id":1,"value_id":101}]`
	skuCode := "SKU-TEST-001"
	sku := model.ProductSKU{
		ProductID:   prod.ID,
		SpecValues:  skuSpecValues,
		SpecHash:    "test_hash",
		SKUCode:     &skuCode,
		Price:       "1999.00",
		MarketPrice: "2299.00",
		CostPrice:   "1500.00",
		Status:      skuStatus,
		Sort:        1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := db.Create(&sku).Error; err != nil {
		t.Fatalf("create sku: %v", err)
	}

	inv := model.ProductSkuInventory{
		ProductID:   prod.ID,
		SKUID:       sku.ID,
		Stock:       stock,
		LockedStock: lockedStock,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := db.Create(&inv).Error; err != nil {
		t.Fatalf("create inventory: %v", err)
	}

	return prod.ID, sku.ID
}

func TestCart_Comprehensive(t *testing.T) {
	db, router, token1, memberID1, token2, memberID2 := setupCartTestApp(t)

	_, skuID1 := createProductWithSKUs(t, db, 1, 1, 100, 10)
	_, skuID2 := createProductWithSKUs(t, db, 1, 1, 50, 0)

	// Helper for HTTP requests
	sendReq := func(method, path, token string, body any) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		if body != nil {
			_ = json.NewEncoder(&buf).Encode(body)
		}
		req := httptest.NewRequest(method, path, &buf)
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}

	// 1. Add 新 SKU 成功
	t.Run("1. Add new SKU succeeds", func(t *testing.T) {
		w := sendReq("POST", "/api/miniapp/cart/add", token1, miniappcart.AddRequest{
			SKUID:    skuID1,
			Quantity: 2,
		})
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp response.Response[any]
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Code != 1 {
			t.Fatalf("expected code 1, got %d (%s)", resp.Code, resp.Message)
		}

		var count int64
		db.Model(&model.CartItem{}).Where("member_id = ? AND sku_id = ?", memberID1, skuID1).Count(&count)
		if count != 1 {
			t.Fatalf("expected 1 cart item in db, got %d", count)
		}
	})

	// 2. 同 SKU 再次 Add，quantity 累加
	t.Run("2. Add same SKU again increments quantity", func(t *testing.T) {
		w := sendReq("POST", "/api/miniapp/cart/add", token1, miniappcart.AddRequest{
			SKUID:    skuID1,
			Quantity: 3,
		})
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		var item model.CartItem
		db.Where("member_id = ? AND sku_id = ?", memberID1, skuID1).First(&item)
		if item.Quantity != 5 {
			t.Fatalf("expected quantity 5, got %d", item.Quantity)
		}
	})

	// 3. 不同会员同 SKU 各自独立
	t.Run("3. Different members have independent cart items", func(t *testing.T) {
		w := sendReq("POST", "/api/miniapp/cart/add", token2, miniappcart.AddRequest{
			SKUID:    skuID1,
			Quantity: 1,
		})
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		var item1, item2 model.CartItem
		db.Where("member_id = ? AND sku_id = ?", memberID1, skuID1).First(&item1)
		db.Where("member_id = ? AND sku_id = ?", memberID2, skuID1).First(&item2)
		if item1.Quantity != 5 || item2.Quantity != 1 {
			t.Fatalf("expected member1: 5, member2: 1, got %d and %d", item1.Quantity, item2.Quantity)
		}
	})

	// 4. 非法 quantity 拒绝
	t.Run("4. Invalid quantity rejected", func(t *testing.T) {
		w1 := sendReq("POST", "/api/miniapp/cart/add", token1, map[string]any{
			"sku_id":   skuID1,
			"quantity": 0,
		})
		var r1 response.Response[any]
		_ = json.Unmarshal(w1.Body.Bytes(), &r1)
		if r1.Code == 1 {
			t.Fatalf("expected failure for quantity 0, got success")
		}

		w2 := sendReq("POST", "/api/miniapp/cart/add", token1, map[string]any{
			"sku_id":   skuID1,
			"quantity": -5,
		})
		var r2 response.Response[any]
		_ = json.Unmarshal(w2.Body.Bytes(), &r2)
		if r2.Code == 1 {
			t.Fatalf("expected failure for quantity -5, got success")
		}
	})

	// 5. SKU 不存在拒绝加入
	t.Run("5. Non-existent SKU rejected", func(t *testing.T) {
		w := sendReq("POST", "/api/miniapp/cart/add", token1, miniappcart.AddRequest{
			SKUID:    999999,
			Quantity: 1,
		})
		var resp response.Response[any]
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Code == 1 {
			t.Fatalf("expected error for non-existent SKU")
		}
	})

	// 6. 商品下架禁止新加入
	t.Run("6. Off-shelf product rejected on add", func(t *testing.T) {
		_, offProdSkuID := createProductWithSKUs(t, db, 0, 1, 100, 0)
		w := sendReq("POST", "/api/miniapp/cart/add", token1, miniappcart.AddRequest{
			SKUID:    offProdSkuID,
			Quantity: 1,
		})
		var resp response.Response[any]
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Code == 1 {
			t.Fatalf("expected rejection for off-shelf product")
		}
	})

	// 7. SKU 停售禁止新加入
	t.Run("7. Off-sale SKU rejected on add", func(t *testing.T) {
		_, offSaleSkuID := createProductWithSKUs(t, db, 1, 0, 100, 0)
		w := sendReq("POST", "/api/miniapp/cart/add", token1, miniappcart.AddRequest{
			SKUID:    offSaleSkuID,
			Quantity: 1,
		})
		var resp response.Response[any]
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Code == 1 {
			t.Fatalf("expected rejection for off-sale SKU")
		}
	})

	// 8. 加购物车不修改库存
	t.Run("8. Adding to cart does not touch inventory", func(t *testing.T) {
		var invBefore model.ProductSkuInventory
		db.Where("sku_id = ?", skuID2).First(&invBefore)

		w := sendReq("POST", "/api/miniapp/cart/add", token1, miniappcart.AddRequest{
			SKUID:    skuID2,
			Quantity: 5,
		})
		var resp response.Response[any]
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Code != 1 {
			t.Fatalf("add to cart failed: %s", resp.Message)
		}

		var invAfter model.ProductSkuInventory
		db.Where("sku_id = ?", skuID2).First(&invAfter)
		if invBefore.Stock != invAfter.Stock || invBefore.LockedStock != invAfter.LockedStock {
			t.Fatalf("inventory changed! before: stock=%d, locked=%d; after: stock=%d, locked=%d",
				invBefore.Stock, invBefore.LockedStock, invAfter.Stock, invAfter.LockedStock)
		}
	})

	// 9. 修改自己的 quantity 成功
	t.Run("9. Update own quantity succeeds", func(t *testing.T) {
		var item model.CartItem
		db.Where("member_id = ? AND sku_id = ?", memberID1, skuID1).First(&item)

		w := sendReq("POST", "/api/miniapp/cart/quantity", token1, miniappcart.UpdateQuantityRequest{
			ID:       item.ID,
			Quantity: 10,
		})
		var resp response.Response[any]
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Code != 1 {
			t.Fatalf("update quantity failed: %s", resp.Message)
		}

		var updated model.CartItem
		db.First(&updated, item.ID)
		if updated.Quantity != 10 {
			t.Fatalf("expected quantity 10, got %d", updated.Quantity)
		}
	})

	// 10. 修改别人购物车失败
	t.Run("10. Cannot update another member's cart item", func(t *testing.T) {
		var item1 model.CartItem
		db.Where("member_id = ? AND sku_id = ?", memberID1, skuID1).First(&item1)

		w := sendReq("POST", "/api/miniapp/cart/quantity", token2, miniappcart.UpdateQuantityRequest{
			ID:       item1.ID,
			Quantity: 99,
		})
		var resp response.Response[any]
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Code == 1 {
			t.Fatalf("expected error modifying another member's cart")
		}

		var unchanged model.CartItem
		db.First(&unchanged, item1.ID)
		if unchanged.Quantity != 10 {
			t.Fatalf("item quantity was unexpectedly changed to %d", unchanged.Quantity)
		}
	})

	// 11. 批量删除自己的购物车 & 12. 删除不能影响其他会员
	t.Run("11 & 12. Batch delete own cart items does not affect others", func(t *testing.T) {
		var item2 model.CartItem
		db.Where("member_id = ? AND sku_id = ?", memberID2, skuID1).First(&item2)

		var item1 model.CartItem
		db.Where("member_id = ? AND sku_id = ?", memberID1, skuID2).First(&item1)

		// Member 1 attempts to delete both item1 and item2
		w := sendReq("POST", "/api/miniapp/cart/delete", token1, miniappcart.DeleteRequest{
			IDs: []int64{item1.ID, item2.ID},
		})
		var resp response.Response[any]
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Code != 1 {
			t.Fatalf("delete failed: %s", resp.Message)
		}

		// item1 deleted
		var count1 int64
		db.Model(&model.CartItem{}).Where("id = ?", item1.ID).Count(&count1)
		if count1 != 0 {
			t.Fatalf("expected item1 deleted, found %d", count1)
		}

		// item2 still exists!
		var count2 int64
		db.Model(&model.CartItem{}).Where("id = ?", item2.ID).Count(&count2)
		if count2 != 1 {
			t.Fatalf("expected item2 preserved, found %d", count2)
		}
	})

	// 13. List 可以返回正常商品
	t.Run("13. List returns active cart items with enriched details", func(t *testing.T) {
		w := sendReq("GET", "/api/miniapp/cart", token1, nil)
		var resp response.Response[[]miniappcart.CartItemResponse]
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Code != 1 {
			t.Fatalf("list cart failed: %s", resp.Message)
		}
		if len(resp.Data) != 1 {
			t.Fatalf("expected 1 item, got %d", len(resp.Data))
		}
		item := resp.Data[0]
		if item.SKUID != skuID1 || !item.Available || item.ProductName != "测试手机" {
			t.Fatalf("unexpected item data: %+v", item)
		}
		if len(item.SpecValues) == 0 || item.SpecValues[0].ValueName != "曜石黑" {
			t.Fatalf("expected enriched spec value 曜石黑, got %+v", item.SpecValues)
		}
		if item.Stock != 90 { // stock 100 - locked 10 = 90
			t.Fatalf("expected stock 90, got %d", item.Stock)
		}
	})

	// 14. Product 下架后 List 仍返回，但 available=false
	t.Run("14. Off-shelf product returned in List with available=false", func(t *testing.T) {
		pID, sID := createProductWithSKUs(t, db, 1, 1, 20, 0)
		_ = sendReq("POST", "/api/miniapp/cart/add", token1, miniappcart.AddRequest{
			SKUID:    sID,
			Quantity: 1,
		})
		// Take product off shelf
		db.Model(&model.Product{}).Where("id = ?", pID).Update("status", 0)

		w := sendReq("GET", "/api/miniapp/cart", token1, nil)
		var resp response.Response[[]miniappcart.CartItemResponse]
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		var found *miniappcart.CartItemResponse
		for i := range resp.Data {
			if resp.Data[i].SKUID == sID {
				found = &resp.Data[i]
				break
			}
		}
		if found == nil {
			t.Fatalf("cart item missing after product off-shelf")
		}
		if found.Available || found.UnavailableReason != "商品已下架" {
			t.Fatalf("expected available=false and reason '商品已下架', got available=%v, reason=%s", found.Available, found.UnavailableReason)
		}
	})

	// 15. SKU 停售后 List 仍返回，但 available=false
	t.Run("15. Off-sale SKU returned in List with available=false", func(t *testing.T) {
		_, sID := createProductWithSKUs(t, db, 1, 1, 20, 0)
		_ = sendReq("POST", "/api/miniapp/cart/add", token1, miniappcart.AddRequest{
			SKUID:    sID,
			Quantity: 1,
		})
		// Disable SKU
		db.Model(&model.ProductSKU{}).Where("id = ?", sID).Update("status", 0)

		w := sendReq("GET", "/api/miniapp/cart", token1, nil)
		var resp response.Response[[]miniappcart.CartItemResponse]
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		var found *miniappcart.CartItemResponse
		for i := range resp.Data {
			if resp.Data[i].SKUID == sID {
				found = &resp.Data[i]
				break
			}
		}
		if found == nil {
			t.Fatalf("cart item missing after SKU disabled")
		}
		if found.Available || found.UnavailableReason != "规格已停售" {
			t.Fatalf("expected available=false and reason '规格已停售', got available=%v, reason=%s", found.Available, found.UnavailableReason)
		}
	})

	// 16. stock=0 返回 unavailable, unavailable_reason="暂时缺货"
	t.Run("16. Zero stock item returned as unavailable", func(t *testing.T) {
		_, sID := createProductWithSKUs(t, db, 1, 1, 10, 0)
		_ = sendReq("POST", "/api/miniapp/cart/add", token1, miniappcart.AddRequest{
			SKUID:    sID,
			Quantity: 1,
		})
		// Set stock to 0
		db.Model(&model.ProductSkuInventory{}).Where("sku_id = ?", sID).Update("stock", 0)

		w := sendReq("GET", "/api/miniapp/cart", token1, nil)
		var resp response.Response[[]miniappcart.CartItemResponse]
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		var found *miniappcart.CartItemResponse
		for i := range resp.Data {
			if resp.Data[i].SKUID == sID {
				found = &resp.Data[i]
				break
			}
		}
		if found == nil {
			t.Fatalf("cart item missing after stock zero")
		}
		if found.Available || found.UnavailableReason != "暂时缺货" {
			t.Fatalf("expected available=false, reason '暂时缺货', got available=%v, reason=%s", found.Available, found.UnavailableReason)
		}
	})

	// 17. available stock = stock - locked_stock
	t.Run("17. Available stock calculation stock - locked_stock", func(t *testing.T) {
		_, sID := createProductWithSKUs(t, db, 1, 1, 100, 40)
		_ = sendReq("POST", "/api/miniapp/cart/add", token1, miniappcart.AddRequest{
			SKUID:    sID,
			Quantity: 1,
		})

		w := sendReq("GET", "/api/miniapp/cart", token1, nil)
		var resp response.Response[[]miniappcart.CartItemResponse]
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		var found *miniappcart.CartItemResponse
		for i := range resp.Data {
			if resp.Data[i].SKUID == sID {
				found = &resp.Data[i]
				break
			}
		}
		if found == nil {
			t.Fatalf("cart item not found")
		}
		if found.Stock != 60 {
			t.Fatalf("expected available stock 60 (100-40), got %d", found.Stock)
		}
	})

	// 18. 购物车 List 无明显 N+1
	t.Run("18. Cart List executes batch queries without N+1", func(t *testing.T) {
		// Clean member 2 cart and insert 5 items
		db.Where("member_id = ?", memberID2).Delete(&model.CartItem{})
		skuIDs := make([]int64, 5)
		for i := 0; i < 5; i++ {
			_, sID := createProductWithSKUs(t, db, 1, 1, 50, 0)
			skuIDs[i] = sID
			_ = sendReq("POST", "/api/miniapp/cart/add", token2, miniappcart.AddRequest{
				SKUID:    sID,
				Quantity: 1,
			})
		}

		queryCount := 0
		counterDB := db.Session(&gorm.Session{})
		counterDB.Callback().Query().Before("gorm:query").Register("count_queries", func(d *gorm.DB) {
			queryCount++
		})

		cartSvc := miniappcart.NewService(counterDB)
		items, err := cartSvc.List(context.Background(), memberID2)
		if err != nil {
			t.Fatalf("list error: %v", err)
		}
		if len(items) != 5 {
			t.Fatalf("expected 5 items, got %d", len(items))
		}
		// Batch strategy: 1 cart_item query + 1 SKU query + 1 Product query + 1 Inventory query = 4 queries (<= 5)
		if queryCount > 6 {
			t.Fatalf("N+1 detected! Total queries: %d for 5 items", queryCount)
		}
	})

	// 19. 多商品进入 checkout preview 成功
	t.Run("19. Multiple items checkout preview succeeds", func(t *testing.T) {
		// Address for member 1
		addr := model.MemberAddress{
			MemberID:     memberID1,
			Name:         "张三",
			Mobile:       "13800138000",
			Province:     "北京市",
			City:         "北京市",
			District:     "海淀区",
			Address:      "中关村大街1号",
			IsDefault:    1,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		db.Create(&addr)

		w := sendReq("POST", "/api/miniapp/checkout/preview", token1, miniapporder.CheckoutPreviewRequest{
			Items: []miniapporder.CheckoutItemRequest{
				{SKUID: skuID1, Quantity: 2},
				{SKUID: skuID2, Quantity: 3},
			},
		})
		var resp response.Response[miniapporder.CheckoutPreviewResponse]
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Code != 1 {
			t.Fatalf("checkout preview failed: %s", resp.Message)
		}
		if len(resp.Data.Items) != 2 {
			t.Fatalf("expected 2 preview items, got %d", len(resp.Data.Items))
		}
		if resp.Data.TotalQuantity != 5 {
			t.Fatalf("expected total quantity 5, got %d", resp.Data.TotalQuantity)
		}
	})

	// 20. 购物车本身不会改变订单库存逻辑
	t.Run("20. Cart does not interfere with order creation and inventory deduction", func(t *testing.T) {
		var addr model.MemberAddress
		db.Where("member_id = ?", memberID1).First(&addr)

		var invBefore model.ProductSkuInventory
		db.Where("sku_id = ?", skuID2).First(&invBefore)

		// Create order for skuID2 x2
		reqID := fmt.Sprintf("req-%d", time.Now().UnixNano())
		w := sendReq("POST", "/api/miniapp/orders/create", token1, miniapporder.OrderCreateRequest{
			RequestID: reqID,
			AddressID: addr.ID,
			Items: []miniapporder.CheckoutItemRequest{
				{SKUID: skuID2, Quantity: 2},
			},
		})
		var resp response.Response[miniapporder.OrderDetailResponse]
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Code != 1 {
			t.Fatalf("order create failed: %s", resp.Message)
		}

		var invAfter model.ProductSkuInventory
		db.Where("sku_id = ?", skuID2).First(&invAfter)
		if invAfter.LockedStock != invBefore.LockedStock+2 || invAfter.Stock != invBefore.Stock {
			t.Fatalf("expected locked_stock increased by 2 (%d -> %d) and stock unchanged (%d), got stock=%d locked=%d",
				invBefore.LockedStock, invBefore.LockedStock+2, invBefore.Stock, invAfter.Stock, invAfter.LockedStock)
		}
	})
}

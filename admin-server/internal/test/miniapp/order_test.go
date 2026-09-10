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
	miniapporder "mall/admin_core/internal/miniapp/order"
	"mall/admin_core/internal/model"
	systemfile "mall/admin_core/internal/system/file"
	systemsetting "mall/admin_core/internal/system/setting"
	"mall/admin_core/pkg/response"
	"mall/admin_core/routers"

	"github.com/gin-gonic/gin"
)

const testJWTSecret = "order_test_jwt_secret_key_1234567890"

func setupOrderTestApp(t *testing.T) (*gorm.DB, http.Handler, string, int64, string, int64) {
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

	authSvc := miniappauth.NewService(db, testJWTSecret, 24*time.Hour)
	mockClient := &mockWeChatClient{
		openID:     "openid_order_user_1",
		sessionKey: "session_key_1",
	}
	authSvc.SetWeChatClient(mockClient)
	res1, err := authSvc.WechatLogin(context.Background(), "code1")
	if err != nil {
		t.Fatalf("login member 1 failed: %v", err)
	}
	token1 := res1.Token
	memberID1 := res1.Member.ID

	// Member 2 for isolation test
	mockClient2 := &mockWeChatClient{
		openID:     "openid_order_user_2",
		sessionKey: "session_key_2",
	}
	authSvc.SetWeChatClient(mockClient2)
	res2, err := authSvc.WechatLogin(context.Background(), "code2")
	if err != nil {
		t.Fatalf("login member 2 failed: %v", err)
	}
	token2 := res2.Token
	memberID2 := res2.Member.ID

	bConfig := bootstrap.Config{
		JWTSecret:     testJWTSecret,
		JWTExpireHour: 24,
	}
	router := routers.NewRouter(bConfig, db, auth.NewService(nil, nil, "", time.Hour))
	return db, router, token1, memberID1, token2, memberID2
}

func createOrderTestData(t *testing.T, db *gorm.DB, memberID int64) (productID int64, skuID int64, addressID int64) {
	t.Helper()

	// 1. Create Address
	addr := model.MemberAddress{
		MemberID:     memberID,
		Name:         "张三",
		Mobile:       "13800138000",
		ProvinceCode: "110000",
		Province:     "北京市",
		CityCode:     "110100",
		City:         "北京市",
		DistrictCode: "110101",
		District:     "东城区",
		Address:      "长安街 1 号",
		IsDefault:    1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := db.Create(&addr).Error; err != nil {
		t.Fatalf("create address: %v", err)
	}
	addressID = addr.ID

	// 2. Create Product
	prod := model.Product{
		Name:         "测试机械键盘",
		CategoryID:   1,
		Status:       1,
		Sort:         0,
		SalesCount:   0,
		VirtualSales: 0,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := db.Create(&prod).Error; err != nil {
		t.Fatalf("create product: %v", err)
	}
	productID = prod.ID

	// 3. Create SKU
	skuCode := "KB-001"
	sku := model.ProductSKU{
		ProductID:   prod.ID,
		SKUCode:     &skuCode,
		Price:       "199.90",
		MarketPrice: "299.90",
		CostPrice:   "100.00",
		Status:      1,
		SpecValues:  `[{"spec_id":1,"value_id":1,"value_name":"青轴"}]`,
		SpecHash:    "hash123",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := db.Create(&sku).Error; err != nil {
		t.Fatalf("create sku: %v", err)
	}
	skuID = sku.ID

	// 4. Create Inventory
	inv := model.ProductSkuInventory{
		ProductID:    prod.ID,
		SKUID:        sku.ID,
		Stock:        50,
		LockedStock:  0,
		WarningStock: 5,
		Version:      1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := db.Create(&inv).Error; err != nil {
		t.Fatalf("create inventory: %v", err)
	}

	return productID, skuID, addressID
}

func TestCheckoutPreview(t *testing.T) {
	db, router, token, memberID, _, _ := setupOrderTestApp(t)
	_, skuID, addrID := createOrderTestData(t, db, memberID)

	t.Run("Normal preview with address and valid items", func(t *testing.T) {
		body, _ := json.Marshal(miniapporder.CheckoutPreviewRequest{
			AddressID: &addrID,
			Items: []miniapporder.CheckoutItemRequest{
				{SKUID: skuID, Quantity: 2},
			},
		})
		req := httptest.NewRequest(http.MethodPost, "/api/miniapp/checkout/preview", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		var resp response.Response[miniapporder.CheckoutPreviewResponse]
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if resp.Code != 1 {
			t.Fatalf("expected code=1, got %d, msg=%s", resp.Code, resp.Message)
		}
		if resp.Data.TotalQuantity != 2 {
			t.Errorf("expected total_quantity=2, got %d", resp.Data.TotalQuantity)
		}
		// 199.90 * 2 = 399.80
		if resp.Data.TotalAmount != "399.80" || resp.Data.PayAmount != "399.80" {
			t.Errorf("expected 399.80, got total=%s, pay=%s", resp.Data.TotalAmount, resp.Data.PayAmount)
		}
		if resp.Data.Address == nil || resp.Data.Address.Name != "张三" {
			t.Errorf("expected address name 张三, got %v", resp.Data.Address)
		}
	})

	t.Run("Preview without address falls back to default", func(t *testing.T) {
		body, _ := json.Marshal(miniapporder.CheckoutPreviewRequest{
			Items: []miniapporder.CheckoutItemRequest{
				{SKUID: skuID, Quantity: 1},
			},
		})
		req := httptest.NewRequest(http.MethodPost, "/api/miniapp/checkout/preview", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var resp response.Response[miniapporder.CheckoutPreviewResponse]
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Code != 1 {
			t.Fatalf("expected code=1, got %d: %s", resp.Code, resp.Message)
		}
		if resp.Data.Address == nil || resp.Data.Address.ID != addrID {
			t.Errorf("expected fallback to default address")
		}
		if resp.Data.PayAmount != "199.90" {
			t.Errorf("expected pay_amount 199.90, got %s", resp.Data.PayAmount)
		}
	})

	t.Run("Preview with insufficient stock fails", func(t *testing.T) {
		body, _ := json.Marshal(miniapporder.CheckoutPreviewRequest{
			Items: []miniapporder.CheckoutItemRequest{
				{SKUID: skuID, Quantity: 9999},
			},
		})
		req := httptest.NewRequest(http.MethodPost, "/api/miniapp/checkout/preview", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var resp response.Response[any]
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Code != 0 {
			t.Errorf("expected error response (code 0), got %d", resp.Code)
		}
	})
}

func TestOrderCreateAndLifecycle(t *testing.T) {
	db, router, token1, memberID1, token2, _ := setupOrderTestApp(t)
	_, skuID, addrID := createOrderTestData(t, db, memberID1)

	var createdOrderID int64
	var orderNo string

	t.Run("Create order successfully with stock deduction", func(t *testing.T) {
		body, _ := json.Marshal(miniapporder.OrderCreateRequest{
			RequestID: "req-uuid-test-001",
			AddressID: addrID,
			Items: []miniapporder.CheckoutItemRequest{
				{SKUID: skuID, Quantity: 2},
			},
			Remark: "麻烦尽快发货",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/miniapp/orders/create", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token1)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		var resp response.Response[miniapporder.OrderDetailResponse]
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if resp.Code != 1 {
			t.Fatalf("expected code=1, got %d, msg=%s", resp.Code, resp.Message)
		}

		createdOrderID = resp.Data.ID
		orderNo = resp.Data.OrderNo

		if resp.Data.Status != model.OrderStatusPendingShipment {
			t.Errorf("expected status=20 (pending_shipment), got %d", resp.Data.Status)
		}
		if resp.Data.PayAmount != "399.80" {
			t.Errorf("expected pay_amount 399.80, got %s", resp.Data.PayAmount)
		}
		if len(resp.Data.Items) != 1 || resp.Data.Items[0].Quantity != 2 {
			t.Errorf("expected 1 item with qty 2")
		}
		if resp.Data.Address == nil || resp.Data.Address.Name != "张三" {
			t.Errorf("expected address snapshot with name 张三")
		}

		// Verify inventory deducted: stock 50 -> 48
		var inv model.ProductSkuInventory
		if err := db.Where("sku_id = ?", skuID).First(&inv).Error; err != nil {
			t.Fatalf("query inv: %v", err)
		}
		if inv.Stock != 48 {
			t.Errorf("expected inventory stock=48, got %d", inv.Stock)
		}

		// Verify inventory log written
		var invLog model.ProductSkuInventoryLog
		if err := db.Where("sku_id = ? AND change_type = ?", skuID, "order_out").First(&invLog).Error; err != nil {
			t.Fatalf("query inv log: %v", err)
		}
		if invLog.Quantity != 2 || invLog.StockDelta != -2 {
			t.Errorf("expected log qty=2 delta=-2, got qty=%d delta=%d", invLog.Quantity, invLog.StockDelta)
		}
	})

	t.Run("Idempotent submission returns existing order without extra stock deduction", func(t *testing.T) {
		body, _ := json.Marshal(miniapporder.OrderCreateRequest{
			RequestID: "req-uuid-test-001", // duplicate request_id
			AddressID: addrID,
			Items: []miniapporder.CheckoutItemRequest{
				{SKUID: skuID, Quantity: 2},
			},
		})
		req := httptest.NewRequest(http.MethodPost, "/api/miniapp/orders/create", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token1)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var resp response.Response[miniapporder.OrderDetailResponse]
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Code != 1 {
			t.Fatalf("expected code=1 for idempotent replay, got %d: %s", resp.Code, resp.Message)
		}
		if resp.Data.ID != createdOrderID {
			t.Errorf("expected order ID %d, got %d", createdOrderID, resp.Data.ID)
		}

		// Verify inventory remains 48, NOT deducted again!
		var inv model.ProductSkuInventory
		_ = db.Where("sku_id = ?", skuID).First(&inv)
		if inv.Stock != 48 {
			t.Errorf("idempotent replay deducted stock again! stock is %d, expected 48", inv.Stock)
		}
	})

	t.Run("Snapshot immutability: changing member address does not change order address", func(t *testing.T) {
		// Update member address in DB
		_ = db.Model(&model.MemberAddress{}).Where("id = ?", addrID).Updates(map[string]any{
			"name":    "李四",
			"address": "上海市浦东新区",
		})

		// Query order detail
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/miniapp/orders/%d", createdOrderID), nil)
		req.Header.Set("Authorization", "Bearer "+token1)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var resp response.Response[miniapporder.OrderDetailResponse]
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Data.Address.Name != "张三" || resp.Data.Address.Address != "长安街 1 号" {
			t.Errorf("order address snapshot was modified! got name=%s, addr=%s", resp.Data.Address.Name, resp.Data.Address.Address)
		}
	})

	t.Run("Member isolation: member 2 cannot access member 1 order", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/miniapp/orders/%d", createdOrderID), nil)
		req.Header.Set("Authorization", "Bearer "+token2)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var resp response.Response[any]
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Code != 0 {
			t.Errorf("expected access denied for member 2, got code=%d", resp.Code)
		}
	})

	t.Run("List orders for member", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/miniapp/orders?page=1&page_size=10", nil)
		req.Header.Set("Authorization", "Bearer "+token1)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var resp response.Response[[]miniapporder.OrderListItemResponse]
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Code != 1 || len(resp.Data) != 1 {
			t.Fatalf("expected 1 order in list, got %d", len(resp.Data))
		}
		if resp.Data[0].OrderNo != orderNo {
			t.Errorf("expected order_no %s, got %s", orderNo, resp.Data[0].OrderNo)
		}
	})

	t.Run("Cancel order restores stock atomically", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/miniapp/orders/%d/cancel", createdOrderID), nil)
		req.Header.Set("Authorization", "Bearer "+token1)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var resp response.Response[any]
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Code != 1 {
			t.Fatalf("expected cancel success, got %d: %s", resp.Code, resp.Message)
		}

		// Verify order status is 50 (cancelled)
		var o model.OrderMain
		_ = db.First(&o, createdOrderID)
		if o.Status != model.OrderStatusCancelled {
			t.Errorf("expected status=50, got %d", o.Status)
		}
		if o.CancelledAt == nil {
			t.Errorf("expected cancelled_at to be set")
		}

		// Verify stock restored from 48 back to 50
		var inv model.ProductSkuInventory
		_ = db.Where("sku_id = ?", skuID).First(&inv)
		if inv.Stock != 50 {
			t.Errorf("expected stock restored to 50, got %d", inv.Stock)
		}

		// Verify cancel inventory log
		var cancelLog model.ProductSkuInventoryLog
		if err := db.Where("sku_id = ? AND change_type = ?", skuID, "order_cancel").First(&cancelLog).Error; err != nil {
			t.Fatalf("find cancel inv log: %v", err)
		}
		if cancelLog.Quantity != 2 || cancelLog.StockDelta != 2 {
			t.Errorf("expected cancel log delta=+2, got %d", cancelLog.StockDelta)
		}
	})

	t.Run("Cannot cancel already cancelled order", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/miniapp/orders/%d/cancel", createdOrderID), nil)
		req.Header.Set("Authorization", "Bearer "+token1)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var resp response.Response[any]
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Code != 0 {
			t.Errorf("expected error on re-cancelling order, got code=%d", resp.Code)
		}
	})

	t.Run("Ship and Confirm Receipt flow", func(t *testing.T) {
		// 1. Create another order
		body, _ := json.Marshal(miniapporder.OrderCreateRequest{
			RequestID: "req-uuid-test-002",
			AddressID: addrID,
			Items: []miniapporder.CheckoutItemRequest{
				{SKUID: skuID, Quantity: 1},
			},
		})
		cReq := httptest.NewRequest(http.MethodPost, "/api/miniapp/orders/create", bytes.NewReader(body))
		cReq.Header.Set("Authorization", "Bearer "+token1)
		cReq.Header.Set("Content-Type", "application/json")
		cW := httptest.NewRecorder()
		router.ServeHTTP(cW, cReq)
		var cResp response.Response[miniapporder.OrderDetailResponse]
		_ = json.Unmarshal(cW.Body.Bytes(), &cResp)
		orderID2 := cResp.Data.ID

		// 2. Try confirming unshipped order -> error
		confReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/miniapp/orders/%d/confirm", orderID2), nil)
		confReq.Header.Set("Authorization", "Bearer "+token1)
		confW := httptest.NewRecorder()
		router.ServeHTTP(confW, confReq)
		var confResp response.Response[any]
		_ = json.Unmarshal(confW.Body.Bytes(), &confResp)
		if confResp.Code != 0 {
			t.Errorf("expected error when confirming unshipped order")
		}

		// 3. Mark shipped in DB (simulating admin shipping)
		now := time.Now()
		company := "顺丰速运"
		no := "SF1234567890"
		db.Model(&model.OrderMain{}).Where("id = ?", orderID2).Updates(map[string]any{
			"status":          model.OrderStatusPendingReceipt,
			"express_company": company,
			"express_no":      no,
			"shipped_at":      &now,
		})

		// 4. Confirm receipt now -> succeeds
		confReq2 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/miniapp/orders/%d/confirm", orderID2), nil)
		confReq2.Header.Set("Authorization", "Bearer "+token1)
		confW2 := httptest.NewRecorder()
		router.ServeHTTP(confW2, confReq2)
		var confResp2 response.Response[any]
		_ = json.Unmarshal(confW2.Body.Bytes(), &confResp2)
		if confResp2.Code != 1 {
			t.Fatalf("expected confirm success, got %d: %s", confResp2.Code, confResp2.Message)
		}

		// Verify order status is 40 (completed)
		var o2 model.OrderMain
		_ = db.First(&o2, orderID2)
		if o2.Status != model.OrderStatusCompleted {
			t.Errorf("expected status=40 (completed), got %d", o2.Status)
		}
		if o2.CompletedAt == nil {
			t.Errorf("expected completed_at to be set")
		}
	})
}

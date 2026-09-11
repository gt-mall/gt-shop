package miniapp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
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
		&model.FreightTemplate{},
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

	t.Run("Preview with specified invalid address fails without fallback", func(t *testing.T) {
		invalidAddrID := int64(999999)
		body, _ := json.Marshal(miniapporder.CheckoutPreviewRequest{
			AddressID: &invalidAddrID,
			Items: []miniapporder.CheckoutItemRequest{
				{SKUID: skuID, Quantity: 1},
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
			t.Errorf("expected error when specified address does not exist, got code %d", resp.Code)
		}
	})

	t.Run("Preview with another member address fails without fallback", func(t *testing.T) {
		otherAddr := model.MemberAddress{
			MemberID:     memberID + 100,
			Name:         "其他用户",
			Mobile:       "13900000000",
			ProvinceCode: "110000",
			Province:     "北京市",
			CityCode:     "110100",
			City:         "北京市",
			DistrictCode: "110101",
			District:     "东城区",
			Address:      "朝阳门 1 号",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		_ = db.Create(&otherAddr)

		body, _ := json.Marshal(miniapporder.CheckoutPreviewRequest{
			AddressID: &otherAddr.ID,
			Items: []miniapporder.CheckoutItemRequest{
				{SKUID: skuID, Quantity: 1},
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
			t.Errorf("expected error when using another member's address, got code %d", resp.Code)
		}
	})

	t.Run("Preview when member has no addresses returns null address", func(t *testing.T) {
		db2, router2, token2, memberID2, _, _ := setupOrderTestApp(t)
		_, m2SkuID, _ := createOrderTestData(t, db2, memberID2)
		_ = db2.Where("member_id = ?", memberID2).Delete(&model.MemberAddress{})

		body, _ := json.Marshal(miniapporder.CheckoutPreviewRequest{
			Items: []miniapporder.CheckoutItemRequest{
				{SKUID: m2SkuID, Quantity: 1},
			},
		})
		req := httptest.NewRequest(http.MethodPost, "/api/miniapp/checkout/preview", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token2)
		w := httptest.NewRecorder()
		router2.ServeHTTP(w, req)

		var resp response.Response[miniapporder.CheckoutPreviewResponse]
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Code != 1 {
			t.Fatalf("expected code 1, got %d: %s", resp.Code, resp.Message)
		}
		if resp.Data.Address != nil {
			t.Errorf("expected null address when user has no address, got %+v", resp.Data.Address)
		}
	})

	t.Run("Preview with multiple SKUs and duplicate SKUs consolidated", func(t *testing.T) {
		body, _ := json.Marshal(miniapporder.CheckoutPreviewRequest{
			Items: []miniapporder.CheckoutItemRequest{
				{SKUID: skuID, Quantity: 1},
				{SKUID: skuID, Quantity: 2},
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
			t.Fatalf("expected code 1, got %d: %s", resp.Code, resp.Message)
		}
		if resp.Data.TotalQuantity != 3 {
			t.Errorf("expected total quantity 3, got %d", resp.Data.TotalQuantity)
		}
		if len(resp.Data.Items) != 1 || resp.Data.Items[0].Quantity != 3 {
			t.Errorf("expected 1 consolidated item with qty 3, got %v", resp.Data.Items)
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

		if orderNo == "" {
			t.Fatalf("expected non-empty order_no")
		}
		if len(orderNo) != 15 {
			t.Errorf("expected order_no length 15, got %d (%s)", len(orderNo), orderNo)
		}
		pattern := `^D[0-9]{6}[0-9ABCDEFGHJKMNPQRSTVWXYZ]{8}$`
		matched, regErr := regexp.MatchString(pattern, orderNo)
		if regErr != nil || !matched {
			t.Errorf("order_no %q does not match expected format %s", orderNo, pattern)
		}

		// Verify database uniqueness
		var countWithOrderNo int64
		if err := db.Model(&model.OrderMain{}).Where("order_no = ?", orderNo).Count(&countWithOrderNo).Error; err != nil {
			t.Fatalf("count order_no: %v", err)
		}
		if countWithOrderNo != 1 {
			t.Errorf("expected exactly 1 order in db with order_no %s, got %d", orderNo, countWithOrderNo)
		}

		if resp.Data.Status != model.OrderStatusPendingShipment {
			t.Errorf("expected status=20 (pending_shipment), got %d", resp.Data.Status)
		}
		if resp.Data.PaidAt != nil {
			t.Errorf("expected paid_at == nil for unpaid order, got %v", resp.Data.PaidAt)
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

		// Verify create log in DB: 0 -> 20
		var createLogs []model.OrderLog
		db.Where("order_id = ? AND action = ?", createdOrderID, "create").Find(&createLogs)
		if len(createLogs) != 1 || createLogs[0].FromStatus != 0 || createLogs[0].ToStatus != model.OrderStatusPendingShipment {
			t.Errorf("expected create log from 0 to 20, got %+v", createLogs)
		}

		// Verify inventory deducted: stock 50 -> 48
		var inv model.ProductSkuInventory
		if err := db.Where("sku_id = ?", skuID).First(&inv).Error; err != nil {
			t.Fatalf("query inv: %v", err)
		}
		if inv.Stock != 48 {
			t.Errorf("expected inventory stock=48, got %d", inv.Stock)
		}

		// Verify inventory log written and references new order_no
		var invLog model.ProductSkuInventoryLog
		if err := db.Where("sku_id = ? AND change_type = ?", skuID, "order_out").First(&invLog).Error; err != nil {
			t.Fatalf("query inv log: %v", err)
		}
		if invLog.Quantity != 2 || invLog.StockDelta != -2 {
			t.Errorf("expected log qty=2 delta=-2, got qty=%d delta=%d", invLog.Quantity, invLog.StockDelta)
		}
		expectedRemark := fmt.Sprintf("订单 %s 扣减库存", orderNo)
		if invLog.Remark != expectedRemark {
			t.Errorf("expected inv log remark %q, got %q", expectedRemark, invLog.Remark)
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

	t.Run("Cancel order restores stock atomically and records 20 -> 50 log", func(t *testing.T) {
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

		// Verify cancel log: 20 -> 50
		var cancelLogs []model.OrderLog
		db.Where("order_id = ? AND action = ?", createdOrderID, "cancel").Find(&cancelLogs)
		if len(cancelLogs) != 1 || cancelLogs[0].FromStatus != model.OrderStatusPendingShipment || cancelLogs[0].ToStatus != model.OrderStatusCancelled {
			t.Errorf("expected cancel log from 20 to 50, got %+v", cancelLogs)
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

	t.Run("Cannot cancel already cancelled order and does not restore stock twice", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/miniapp/orders/%d/cancel", createdOrderID), nil)
		req.Header.Set("Authorization", "Bearer "+token1)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var resp response.Response[any]
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Code != 0 {
			t.Errorf("expected error on re-cancelling order, got code=%d", resp.Code)
		}

		// Verify stock remains 50
		var inv model.ProductSkuInventory
		_ = db.Where("sku_id = ?", skuID).First(&inv)
		if inv.Stock != 50 {
			t.Errorf("expected stock remains 50, got %d", inv.Stock)
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

		// Verify confirm log: 30 -> 40
		var confirmLogs []model.OrderLog
		db.Where("order_id = ? AND action = ?", orderID2, "confirm").Find(&confirmLogs)
		if len(confirmLogs) != 1 || confirmLogs[0].FromStatus != model.OrderStatusPendingReceipt || confirmLogs[0].ToStatus != model.OrderStatusCompleted {
			t.Errorf("expected confirm log from 30 to 40, got %+v", confirmLogs)
		}

		// 5. Try confirming already completed order -> error
		confReq3 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/miniapp/orders/%d/confirm", orderID2), nil)
		confReq3.Header.Set("Authorization", "Bearer "+token1)
		confW3 := httptest.NewRecorder()
		router.ServeHTTP(confW3, confReq3)
		var confResp3 response.Response[any]
		_ = json.Unmarshal(confW3.Body.Bytes(), &confResp3)
		if confResp3.Code != 0 {
			t.Errorf("expected error on re-confirming already completed order, got code %d", confResp3.Code)
		}
	})
}

func TestOrderImageResolutionAndDomainChange(t *testing.T) {
	db, router, token1, memberID1, _, _ := setupOrderTestApp(t)
	_, skuID, addrID := createOrderTestData(t, db, memberID1)

	// Create system file
	fileObj := model.SystemFile{
		Storage:   "local",
		Name:      "keyboard.png",
		Path:      "2026/09/11111111-2222-3333-4444-555555555555.png",
		Ext:       "png",
		Size:      1024,
		Width:     800,
		Height:    600,
		CreatedAt: time.Now(),
	}
	if err := db.Create(&fileObj).Error; err != nil {
		t.Fatalf("create system file: %v", err)
	}

	// Associate file with SKU
	_ = db.Model(&model.ProductSKU{}).Where("id = ?", skuID).Update("image_id", fileObj.ID)

	// Create order
	body, _ := json.Marshal(miniapporder.OrderCreateRequest{
		RequestID: "req-img-test-001",
		AddressID: addrID,
		Items: []miniapporder.CheckoutItemRequest{
			{SKUID: skuID, Quantity: 1},
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
		t.Fatalf("create order failed: %s", resp.Message)
	}

	orderID := resp.Data.ID
	if len(resp.Data.Items) == 0 || resp.Data.Items[0].ImageURL == "" {
		t.Fatalf("expected item image_url resolved, got %v", resp.Data.Items)
	}
	initialImageURL := resp.Data.Items[0].ImageURL

	// 1. Image resolution verified on Detail
	dReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/miniapp/orders/%d", orderID), nil)
	dReq.Header.Set("Authorization", "Bearer "+token1)
	dW := httptest.NewRecorder()
	router.ServeHTTP(dW, dReq)
	var dResp response.Response[miniapporder.OrderDetailResponse]
	_ = json.Unmarshal(dW.Body.Bytes(), &dResp)
	if dResp.Data.Items[0].ImageURL != initialImageURL {
		t.Errorf("expected image url %s, got %s", initialImageURL, dResp.Data.Items[0].ImageURL)
	}

	// 2. Modify system_file storage domain / config
	fileSvc := systemfile.NewService(db)
	newCfg := systemfile.DefaultConfig()
	newCfg.Local.Domain = "https://cdn.example.com"
	if err := fileSvc.SaveConfig(context.Background(), newCfg); err != nil {
		t.Fatalf("save new file config: %v", err)
	}

	// 3. Query Detail again: URL should be updated to reflect new domain!
	dW2 := httptest.NewRecorder()
	router.ServeHTTP(dW2, dReq)
	var dResp2 response.Response[miniapporder.OrderDetailResponse]
	_ = json.Unmarshal(dW2.Body.Bytes(), &dResp2)
	expectedPrefix := "https://cdn.example.com"
	if len(dResp2.Data.Items) == 0 || len(dResp2.Data.Items[0].ImageURL) < len(expectedPrefix) || dResp2.Data.Items[0].ImageURL[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("expected image url updated with new domain prefix %s, got %s", expectedPrefix, dResp2.Data.Items[0].ImageURL)
	}

	// 4. If file is deleted from system_file, fallback to sku_image_url snapshot
	_ = db.Delete(&fileObj)
	dW3 := httptest.NewRecorder()
	router.ServeHTTP(dW3, dReq)
	var dResp3 response.Response[miniapporder.OrderDetailResponse]
	_ = json.Unmarshal(dW3.Body.Bytes(), &dResp3)
	if len(dResp3.Data.Items) == 0 || dResp3.Data.Items[0].ImageURL == "" {
		t.Errorf("expected fallback to snapshot sku_image_url when file deleted, got empty")
	}
}

func TestOrderConcurrentStockDeduction(t *testing.T) {
	db, router, token1, memberID1, _, _ := setupOrderTestApp(t)
	_, skuID, addrID := createOrderTestData(t, db, memberID1)

	// Set stock to exactly 3
	if err := db.Model(&model.ProductSkuInventory{}).Where("sku_id = ?", skuID).Update("stock", 3).Error; err != nil {
		t.Fatalf("update stock: %v", err)
	}

	const totalGoroutines = 10
	var successCount int32
	var failCount int32
	var wg sync.WaitGroup

	for i := 0; i < totalGoroutines; i++ {
		wg.Add(1)
		reqID := fmt.Sprintf("req-conc-stock-%d", i)
		go func(rID string) {
			defer wg.Done()
			body, _ := json.Marshal(miniapporder.OrderCreateRequest{
				RequestID: rID,
				AddressID: addrID,
				Items: []miniapporder.CheckoutItemRequest{
					{SKUID: skuID, Quantity: 1},
				},
			})
			req := httptest.NewRequest(http.MethodPost, "/api/miniapp/orders/create", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token1)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			var resp response.Response[miniapporder.OrderDetailResponse]
			_ = json.Unmarshal(w.Body.Bytes(), &resp)
			if resp.Code == 1 {
				atomic.AddInt32(&successCount, 1)
			} else {
				atomic.AddInt32(&failCount, 1)
			}
		}(reqID)
	}
	wg.Wait()

	if successCount != 3 {
		t.Errorf("expected exactly 3 orders to succeed, got %d (failed: %d)", successCount, failCount)
	}
	if failCount != 7 {
		t.Errorf("expected 7 orders to fail, got %d", failCount)
	}

	// Verify inventory in DB: stock must be 0
	var inv model.ProductSkuInventory
	if err := db.Where("sku_id = ?", skuID).First(&inv).Error; err != nil {
		t.Fatalf("fetch inventory: %v", err)
	}
	if inv.Stock != 0 {
		t.Errorf("expected stock 0 after 3 purchases, got %d", inv.Stock)
	}

	// Verify order count in DB
	var orderCount int64
	db.Model(&model.OrderMain{}).Count(&orderCount)
	if orderCount != 3 {
		t.Errorf("expected exactly 3 orders in db, got %d", orderCount)
	}
}

func TestOrderConcurrentIdenticalRequest(t *testing.T) {
	db, router, token1, memberID1, _, _ := setupOrderTestApp(t)
	_, skuID, addrID := createOrderTestData(t, db, memberID1)

	const totalGoroutines = 10
	var successCount int32
	var orderIDs sync.Map
	var wg sync.WaitGroup

	sameReqID := "req-idempotent-race-001"

	for i := 0; i < totalGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			body, _ := json.Marshal(miniapporder.OrderCreateRequest{
				RequestID: sameReqID,
				AddressID: addrID,
				Items: []miniapporder.CheckoutItemRequest{
					{SKUID: skuID, Quantity: 1},
				},
			})
			req := httptest.NewRequest(http.MethodPost, "/api/miniapp/orders/create", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token1)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			var resp response.Response[miniapporder.OrderDetailResponse]
			_ = json.Unmarshal(w.Body.Bytes(), &resp)
			if resp.Code == 1 {
				atomic.AddInt32(&successCount, 1)
				orderIDs.Store(resp.Data.ID, true)
			}
		}()
	}
	wg.Wait()

	if successCount != totalGoroutines {
		t.Errorf("expected all %d concurrent identical requests to succeed, got %d", totalGoroutines, successCount)
	}

	// All returned order IDs must be identical
	distinctIDs := 0
	orderIDs.Range(func(key, value any) bool {
		distinctIDs++
		return true
	})
	if distinctIDs != 1 {
		t.Errorf("expected exactly 1 distinct order ID from concurrent identical requests, got %d", distinctIDs)
	}

	// Stock must have been deducted only ONCE: 50 - 1 = 49
	var inv model.ProductSkuInventory
	if err := db.Where("sku_id = ?", skuID).First(&inv).Error; err != nil {
		t.Fatalf("fetch inventory: %v", err)
	}
	if inv.Stock != 49 {
		t.Errorf("expected stock 49 (deducted once), got %d", inv.Stock)
	}

	// Order count in DB must be exactly 1
	var orderCount int64
	db.Model(&model.OrderMain{}).Where("request_id = ?", sameReqID).Count(&orderCount)
	if orderCount != 1 {
		t.Errorf("expected 1 order with request_id in db, got %d", orderCount)
	}
}

func TestOrderSpecValuesEnrichment(t *testing.T) {
	db, router, token, memberID, _, _ := setupOrderTestApp(t)

	// Create product with specs
	specsJSON := `[{"id":1,"name":"颜色","values":[{"id":101,"name":"曜石黑"},{"id":102,"name":"极光银"}]},{"id":2,"name":"配置","values":[{"id":201,"name":"8GB+128GB"},{"id":202,"name":"12GB+256GB"}]}]`
	prod := model.Product{
		Name:       "智能手机 Pro",
		CategoryID: 1,
		Status:     1,
		Specs:      &specsJSON,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := db.Create(&prod).Error; err != nil {
		t.Fatalf("create product: %v", err)
	}

	// SKU has only spec_id and value_id in spec_values
	skuCode := "PHONE-PRO-BLK"
	sku := model.ProductSKU{
		ProductID:   prod.ID,
		SKUCode:     &skuCode,
		Price:       "2999.00",
		MarketPrice: "3499.00",
		CostPrice:   "2000.00",
		Status:      1,
		SpecValues:  `[{"spec_id":1,"value_id":101},{"spec_id":2,"value_id":202}]`,
		SpecHash:    "hash_phone_pro",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := db.Create(&sku).Error; err != nil {
		t.Fatalf("create sku: %v", err)
	}

	inv := model.ProductSkuInventory{
		ProductID:    prod.ID,
		SKUID:        sku.ID,
		Stock:        20,
		LockedStock:  0,
		WarningStock: 2,
		Version:      1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := db.Create(&inv).Error; err != nil {
		t.Fatalf("create inventory: %v", err)
	}

	addr := model.MemberAddress{
		MemberID:  memberID,
		Name:      "张三",
		Mobile:    "13800000000",
		Province:  "北京市",
		City:      "北京市",
		District:  "海淀区",
		Address:   "中关村大街1号",
		IsDefault: 1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := db.Create(&addr).Error; err != nil {
		t.Fatalf("create address: %v", err)
	}

	t.Run("Preview enriches spec_values with spec_name and value_name", func(t *testing.T) {
		body, _ := json.Marshal(miniapporder.CheckoutPreviewRequest{
			AddressID: &addr.ID,
			Items: []miniapporder.CheckoutItemRequest{
				{SKUID: sku.ID, Quantity: 1},
			},
		})
		req, _ := http.NewRequest(http.MethodPost, "/api/miniapp/checkout/preview", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("preview status %d: %s", w.Code, w.Body.String())
		}
		var resp struct {
			Code int                                  `json:"code"`
			Data miniapporder.CheckoutPreviewResponse `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if len(resp.Data.Items) != 1 {
			t.Fatalf("expected 1 item, got %d", len(resp.Data.Items))
		}
		item := resp.Data.Items[0]
		if !strings.Contains(item.SKUSpecValues, "曜石黑") || !strings.Contains(item.SKUSpecValues, "12GB+256GB") {
			t.Errorf("expected preview to contain enriched spec names, got %s", item.SKUSpecValues)
		}
	})

	t.Run("Create and Detail enrich spec_values snapshot", func(t *testing.T) {
		reqBody, _ := json.Marshal(miniapporder.OrderCreateRequest{
			RequestID: "req-spec-test-01",
			AddressID: addr.ID,
			Items: []miniapporder.CheckoutItemRequest{
				{SKUID: sku.ID, Quantity: 1},
			},
		})
		req, _ := http.NewRequest(http.MethodPost, "/api/miniapp/orders/create", bytes.NewBuffer(reqBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("create status %d: %s", w.Code, w.Body.String())
		}
		var resp struct {
			Code int                                `json:"code"`
			Data miniapporder.OrderDetailResponse `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if len(resp.Data.Items) != 1 {
			t.Fatalf("expected 1 item, got %d", len(resp.Data.Items))
		}
		item := resp.Data.Items[0]
		if item.SKUSpecValues == nil || !strings.Contains(*item.SKUSpecValues, "曜石黑") || !strings.Contains(*item.SKUSpecValues, "12GB+256GB") {
			t.Errorf("expected order item to contain enriched spec names, got %v", item.SKUSpecValues)
		}

		// Also check Detail endpoint
		getReq, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/miniapp/orders/%d", resp.Data.ID), nil)
		getReq.Header.Set("Authorization", "Bearer "+token)
		wGet := httptest.NewRecorder()
		router.ServeHTTP(wGet, getReq)

		var detailResp struct {
			Code int                                `json:"code"`
			Data miniapporder.OrderDetailResponse `json:"data"`
		}
		_ = json.Unmarshal(wGet.Body.Bytes(), &detailResp)
		detailItem := detailResp.Data.Items[0]
		if detailItem.SKUSpecValues == nil || !strings.Contains(*detailItem.SKUSpecValues, "曜石黑") {
			t.Errorf("expected detail item to contain enriched spec names, got %v", detailItem.SKUSpecValues)
		}
	})
}


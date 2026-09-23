package miniapp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	miniapporder "mall/admin_core/internal/miniapp/order"
	miniapppayment "mall/admin_core/internal/miniapp/payment"
	"mall/admin_core/internal/model"
	productfreight "mall/admin_core/internal/product/freight"
	productsvc "mall/admin_core/internal/product/product"
	"mall/admin_core/internal/region"
	"mall/admin_core/pkg/response"

	"gorm.io/gorm"
)

// Production routes must not expose the previous simulated payment path.
func TestPaymentRoutesRejectMockAndRequireMerchantConfig(t *testing.T) {
	db, router, token, memberID, _, _ := setupOrderTestApp(t)
	_, skuID, addrID := createOrderTestData(t, db, memberID)
	order, err := miniapporder.NewService(db).Create(context.Background(), model.Member{ID: memberID}, miniapporder.OrderCreateRequest{RequestID: "payment-real-route", AddressID: addrID, Items: []miniapporder.CheckoutItemRequest{{SKUID: skuID, Quantity: 1}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/api/miniapp/payments/create", "/api/miniapp/payments/mock-pay"} {
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader([]byte(`{"order_id":1}`)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("obsolete payment route %s remains reachable: %d", path, w.Code)
		}
	}
	body, _ := json.Marshal(miniapppayment.CreatePaymentRequest{OrderID: order.ID})
	req := httptest.NewRequest(http.MethodPost, "/api/miniapp/payments/prepay", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var result response.Response[any]
	_ = json.Unmarshal(w.Body.Bytes(), &result)
	if result.Code != 0 {
		t.Fatalf("prepay without merchant credentials must fail, got %s", w.Body.String())
	}
	var saved model.OrderMain
	if err := db.First(&saved, order.ID).Error; err != nil || saved.Status != model.OrderStatusPendingPayment {
		t.Fatalf("failed prepay changed order: %v, status %d", err, saved.Status)
	}
	var inv model.ProductSkuInventory
	if err := db.Where("sku_id = ?", skuID).First(&inv).Error; err != nil || inv.LockedStock != 1 {
		t.Fatalf("failed prepay changed locked stock: %v, %+v", err, inv)
	}
}

func TestLegacyMockPendingPaymentIsRetiredBeforeWechatPrepay(t *testing.T) {
	db, _, _, memberID, _, _ := setupOrderTestApp(t)
	_, skuID, addrID := createOrderTestData(t, db, memberID)
	order, err := miniapporder.NewService(db).Create(context.Background(), model.Member{ID: memberID}, miniapporder.OrderCreateRequest{RequestID: "legacy-payment-switch", AddressID: addrID, Items: []miniapporder.CheckoutItemRequest{{SKUID: skuID, Quantity: 1}}})
	if err != nil {
		t.Fatal(err)
	}
	svc := miniapppayment.NewService(db)
	old, err := svc.CreatePayment(context.Background(), memberID, miniapppayment.CreatePaymentRequest{OrderID: order.ID, Channel: "mock"})
	if err != nil {
		t.Fatal(err)
	}
	current, err := svc.CreatePayment(context.Background(), memberID, miniapppayment.CreatePaymentRequest{OrderID: order.ID, Channel: "wechat"})
	if err != nil {
		t.Fatal(err)
	}
	if current.PaymentNo == old.PaymentNo || current.Channel != "wechat" {
		t.Fatal("legacy mock payment reused for WeChat")
	}
	if err := db.First(old, old.ID).Error; err != nil || old.Status != model.PaymentStatusClosed {
		t.Fatalf("legacy payment remains pending: %v, status %d", err, old.Status)
	}
}

// 4. 支付超时关闭订单并释放锁定库存
func TestOrderPayment_TimeoutCloseAndReleaseLockedStock(t *testing.T) {
	db, router, token1, memberID1, _, _ := setupOrderTestApp(t)
	_, skuID, addrID := createOrderTestData(t, db, memberID1)

	// 创建订单 (锁定 3 件库存)
	body, _ := json.Marshal(miniapporder.OrderCreateRequest{
		RequestID: "req-payment-timeout-01",
		AddressID: addrID,
		Items:     []miniapporder.CheckoutItemRequest{{SKUID: skuID, Quantity: 3}},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/miniapp/orders/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token1)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var createResp response.Response[miniapporder.OrderDetailResponse]
	_ = json.Unmarshal(w.Body.Bytes(), &createResp)
	orderID := createResp.Data.ID

	// 确认锁定库存为 3
	var inv model.ProductSkuInventory
	_ = db.Where("sku_id = ?", skuID).First(&inv)
	if inv.Stock != 50 || inv.LockedStock != 3 {
		t.Fatalf("expected stock 50, locked 3, got stock=%d, locked=%d", inv.Stock, inv.LockedStock)
	}

	// 执行超时关闭服务
	paymentSvc := miniapppayment.NewService(db)
	err := paymentSvc.CloseTimeoutOrder(context.Background(), orderID)
	if err != nil {
		t.Fatalf("close timeout order failed: %v", err)
	}

	// 验证订单状态流转至 50 (已取消)
	var orderMain model.OrderMain
	_ = db.First(&orderMain, orderID)
	if orderMain.Status != model.OrderStatusCancelled {
		t.Errorf("expected order status 50 (cancelled), got %d", orderMain.Status)
	}

	// 验证锁定库存已全额释放：stock 50, locked_stock 0
	_ = db.Where("sku_id = ?", skuID).First(&inv)
	if inv.Stock != 50 || inv.LockedStock != 0 {
		t.Errorf("expected stock=50, locked=0 after timeout, got stock=%d, locked=%d", inv.Stock, inv.LockedStock)
	}

	// 验证超时取消日志: 10 -> 50, action="timeout_close"
	var orderLog model.OrderLog
	if err := db.Where("order_id = ? AND action = ?", orderID, "timeout_close").First(&orderLog).Error; err != nil {
		t.Fatalf("expected timeout_close log: %v", err)
	}
	if orderLog.FromStatus != model.OrderStatusPendingPayment || orderLog.ToStatus != model.OrderStatusCancelled {
		t.Errorf("expected timeout log 10 -> 50, got %d -> %d", orderLog.FromStatus, orderLog.ToStatus)
	}

	// 再次执行超时关闭应幂等无害
	err2 := paymentSvc.CloseTimeoutOrder(context.Background(), orderID)
	if err2 != nil {
		t.Errorf("expected CloseTimeoutOrder to be idempotent, got err: %v", err2)
	}
}

// 5. 库存不足并发保护
func TestOrderPayment_ConcurrencyProtectionWhenStockInsufficient(t *testing.T) {
	db, router, token1, memberID1, _, _ := setupOrderTestApp(t)
	_, skuID, addrID := createOrderTestData(t, db, memberID1)

	// 设置初始库存仅为 2 件
	if err := db.Model(&model.ProductSkuInventory{}).Where("sku_id = ?", skuID).Updates(map[string]any{
		"stock":        2,
		"locked_stock": 0,
	}).Error; err != nil {
		t.Fatalf("update stock: %v", err)
	}

	const totalRequests = 6
	var successCount int32
	var failCount int32
	var wg sync.WaitGroup

	for i := 0; i < totalRequests; i++ {
		wg.Add(1)
		reqID := fmt.Sprintf("req-insufficient-conc-%d", i)
		go func(rID string) {
			defer wg.Done()
			b, _ := json.Marshal(miniapporder.OrderCreateRequest{
				RequestID: rID,
				AddressID: addrID,
				Items:     []miniapporder.CheckoutItemRequest{{SKUID: skuID, Quantity: 1}},
			})
			r := httptest.NewRequest(http.MethodPost, "/api/miniapp/orders/create", bytes.NewReader(b))
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("Authorization", "Bearer "+token1)
			rw := httptest.NewRecorder()
			router.ServeHTTP(rw, r)

			var res response.Response[miniapporder.OrderDetailResponse]
			_ = json.Unmarshal(rw.Body.Bytes(), &res)
			if res.Code == 1 {
				atomic.AddInt32(&successCount, 1)
			} else {
				atomic.AddInt32(&failCount, 1)
			}
		}(reqID)
	}
	wg.Wait()

	if successCount != 2 {
		t.Errorf("expected exactly 2 orders to succeed, got %d", successCount)
	}
	if failCount != 4 {
		t.Errorf("expected 4 orders to fail with insufficient stock, got %d", failCount)
	}

	// 验证库存：stock=2, locked_stock=2 (可用为 0)
	var inv model.ProductSkuInventory
	_ = db.Where("sku_id = ?", skuID).First(&inv)
	if inv.Stock != 2 || inv.LockedStock != 2 {
		t.Errorf("expected stock=2, locked=2, got stock=%d, locked=%d", inv.Stock, inv.LockedStock)
	}
}

// 6. 商品存在未完成订单时禁止删除 (10/20/30 禁止删除)
func TestProductDelete_ForbiddenWhenUnfinishedOrdersExist(t *testing.T) {
	db, router, token1, memberID1, _, _ := setupOrderTestApp(t)
	productID, skuID, addrID := createOrderTestData(t, db, memberID1)

	prodSvc := productsvc.NewService(db)

	// Case 1: 存在状态 10 (待支付) 订单时，禁止删除
	b1, _ := json.Marshal(miniapporder.OrderCreateRequest{
		RequestID: "req-delete-check-01",
		AddressID: addrID,
		Items:     []miniapporder.CheckoutItemRequest{{SKUID: skuID, Quantity: 1}},
	})
	r1 := httptest.NewRequest(http.MethodPost, "/api/miniapp/orders/create", bytes.NewReader(b1))
	r1.Header.Set("Content-Type", "application/json")
	r1.Header.Set("Authorization", "Bearer "+token1)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, r1)
	var res1 response.Response[miniapporder.OrderDetailResponse]
	_ = json.Unmarshal(w1.Body.Bytes(), &res1)
	orderID1 := res1.Data.ID

	errDel1 := prodSvc.Delete(context.Background(), productID)
	if !errors.Is(errDel1, productsvc.ErrProductHasUnfinishedOrders) {
		t.Fatalf("expected ErrProductHasUnfinishedOrders when order is status 10, got: %v", errDel1)
	}

	// Case 2: 订单支付成功变为状态 20 (待发货)，仍然禁止删除
	paySvc := miniapppayment.NewService(db)
	_, errPay := paySvc.MockPay(context.Background(), memberID1, miniapppayment.MockPayRequest{OrderID: orderID1})
	if errPay != nil {
		t.Fatalf("mock pay failed: %v", errPay)
	}

	errDel2 := prodSvc.Delete(context.Background(), productID)
	if !errors.Is(errDel2, productsvc.ErrProductHasUnfinishedOrders) {
		t.Fatalf("expected ErrProductHasUnfinishedOrders when order is status 20, got: %v", errDel2)
	}

	// Case 3: 订单变更为状态 30 (待收货)，仍然禁止删除
	now := time.Now()
	_ = db.Model(&model.OrderMain{}).Where("id = ?", orderID1).Updates(map[string]any{
		"status":     model.OrderStatusPendingReceipt,
		"shipped_at": &now,
	})

	errDel3 := prodSvc.Delete(context.Background(), productID)
	if !errors.Is(errDel3, productsvc.ErrProductHasUnfinishedOrders) {
		t.Fatalf("expected ErrProductHasUnfinishedOrders when order is status 30, got: %v", errDel3)
	}

	// Case 4: 订单变更为状态 40 (已完成)，且库存清零后，允许正常软删除
	_ = db.Model(&model.OrderMain{}).Where("id = ?", orderID1).Updates(map[string]any{
		"status":       model.OrderStatusCompleted,
		"completed_at": &now,
	})
	// 清零库存以满足删除条件
	_ = db.Model(&model.ProductSkuInventory{}).Where("product_id = ?", productID).Updates(map[string]any{
		"stock":        0,
		"locked_stock": 0,
	})

	errDel4 := prodSvc.Delete(context.Background(), productID)
	if errDel4 != nil {
		t.Fatalf("expected delete to succeed after order completed and stock 0, got err: %v", errDel4)
	}

	// 确认商品与 SKU 已被软删除
	var prod model.Product
	if err := db.Where("id = ? AND deleted_at IS NULL", productID).First(&prod).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("expected product to be soft deleted")
	}
}

// 7. 运费模板新增/修改时校验省份和城市编码是否真实有效
func TestFreightTemplate_RegionValidation(t *testing.T) {
	db, _, _, _, _, _ := setupOrderTestApp(t)

	regSvc, err := region.NewService()
	if err != nil {
		t.Fatalf("init region service: %v", err)
	}
	freightSvc := productfreight.NewService(db)
	freightSvc.SetRegionService(regSvc)

	// 1. 合法省份与城市规则 -> 验证通过
	validExtra := []productfreight.ExtraFeeRule{
		{ProvinceCode: "110000", CityCode: "110100", Fee: "10.00"}, // 北京市市辖区
		{ProvinceCode: "440000", CityCode: "440100", Fee: "5.00"},  // 广东省广州市
		{ProvinceCode: "440000", CityCode: "0", Fee: "2.00"},       // 广东省全省
	}
	validNoDel := []productfreight.NoDeliveryRule{
		{ProvinceCode: "650000", CityCode: "0"}, // 新疆全省不配送
	}
	_, _, _, errValid := freightSvc.ValidateRules("0.00", validExtra, validNoDel)
	if errValid != nil {
		t.Fatalf("expected valid rules to pass, got: %v", errValid)
	}

	// 2. 无效省份编码 -> 验证失败
	invalidProvExtra := []productfreight.ExtraFeeRule{
		{ProvinceCode: "999999", CityCode: "0", Fee: "10.00"},
	}
	_, _, _, errInvProv := freightSvc.ValidateRules("0.00", invalidProvExtra, nil)
	if errInvProv == nil {
		t.Fatalf("expected invalid province code to fail")
	}

	// 3. 城市不属于对应省份（北京编码 110000 搭配 广州城市编码 440100） -> 验证失败
	mismatchedCityExtra := []productfreight.ExtraFeeRule{
		{ProvinceCode: "110000", CityCode: "440100", Fee: "10.00"},
	}
	_, _, _, errMismatch := freightSvc.ValidateRules("0.00", mismatchedCityExtra, nil)
	if errMismatch == nil {
		t.Fatalf("expected mismatched city code to fail")
	}

	// 4. 不配送规则中的无效省份与城市 -> 验证失败
	invalidNoDelProv := []productfreight.NoDeliveryRule{
		{ProvinceCode: "888888", CityCode: "0"},
	}
	_, _, _, errInvNoDel := freightSvc.ValidateRules("0.00", nil, invalidNoDelProv)
	if errInvNoDel == nil {
		t.Fatalf("expected invalid no-delivery province code to fail")
	}
}

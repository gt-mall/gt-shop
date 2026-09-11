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

// 1. 创建待支付订单、锁库存、支付成功转待发货并真实扣库存
func TestOrderPayment_HappyPathAndStockTransitions(t *testing.T) {
	db, router, token1, memberID1, _, _ := setupOrderTestApp(t)
	_, skuID, addrID := createOrderTestData(t, db, memberID1)

	// Step 1: 下单创建待支付订单
	body, _ := json.Marshal(miniapporder.OrderCreateRequest{
		RequestID: "req-payment-happypath-01",
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

	var createResp response.Response[miniapporder.OrderDetailResponse]
	_ = json.Unmarshal(w.Body.Bytes(), &createResp)
	if createResp.Code != 1 {
		t.Fatalf("create order failed: %s", createResp.Message)
	}
	orderID := createResp.Data.ID

	// 验证订单状态为 10 (待支付)
	if createResp.Data.Status != model.OrderStatusPendingPayment {
		t.Fatalf("expected order status 10 (pending payment), got %d", createResp.Data.Status)
	}

	// 验证库存已锁定：stock 不变仍为 50，locked_stock 为 2
	var inv model.ProductSkuInventory
	_ = db.Where("sku_id = ?", skuID).First(&inv)
	if inv.Stock != 50 || inv.LockedStock != 2 {
		t.Fatalf("expected stock 50, locked_stock 2, got stock=%d, locked=%d", inv.Stock, inv.LockedStock)
	}

	// Step 2: 创建支付单
	payCreateBody, _ := json.Marshal(miniapppayment.CreatePaymentRequest{
		OrderID: orderID,
		Channel: "mock",
	})
	payReq := httptest.NewRequest(http.MethodPost, "/api/miniapp/payments/create", bytes.NewReader(payCreateBody))
	payReq.Header.Set("Content-Type", "application/json")
	payReq.Header.Set("Authorization", "Bearer "+token1)
	payW := httptest.NewRecorder()
	router.ServeHTTP(payW, payReq)

	var payResp response.Response[miniapppayment.PaymentResponse]
	_ = json.Unmarshal(payW.Body.Bytes(), &payResp)
	if payResp.Code != 1 {
		t.Fatalf("create payment failed: %s", payResp.Message)
	}
	if payResp.Data.Status != model.PaymentStatusPending {
		t.Errorf("expected payment status 10, got %d", payResp.Data.Status)
	}
	if len(payResp.Data.PaymentNo) == 0 || payResp.Data.PaymentNo[0] != 'P' {
		t.Errorf("expected payment_no starting with P, got %s", payResp.Data.PaymentNo)
	}
	paymentNo := payResp.Data.PaymentNo

	// 幂等测试：再次调用 create payment 返回相同的 payment_no
	payReq2 := httptest.NewRequest(http.MethodPost, "/api/miniapp/payments/create", bytes.NewReader(payCreateBody))
	payReq2.Header.Set("Content-Type", "application/json")
	payReq2.Header.Set("Authorization", "Bearer "+token1)
	payW2 := httptest.NewRecorder()
	router.ServeHTTP(payW2, payReq2)
	var payResp2 response.Response[miniapppayment.PaymentResponse]
	_ = json.Unmarshal(payW2.Body.Bytes(), &payResp2)
	if payResp2.Data.PaymentNo != paymentNo {
		t.Errorf("expected same payment_no %s on idempotent call, got %s", paymentNo, payResp2.Data.PaymentNo)
	}

	// Step 3: 模拟支付成功
	mockPayBody, _ := json.Marshal(miniapppayment.MockPayRequest{
		PaymentNo: paymentNo,
		Channel:   "mock",
	})
	mockReq := httptest.NewRequest(http.MethodPost, "/api/miniapp/payments/mock-pay", bytes.NewReader(mockPayBody))
	mockReq.Header.Set("Content-Type", "application/json")
	mockReq.Header.Set("Authorization", "Bearer "+token1)
	mockW := httptest.NewRecorder()
	router.ServeHTTP(mockW, mockReq)

	var mockResp response.Response[miniapppayment.PaymentResponse]
	_ = json.Unmarshal(mockW.Body.Bytes(), &mockResp)
	if mockResp.Code != 1 {
		t.Fatalf("mock pay failed: %s", mockResp.Message)
	}
	if mockResp.Data.Status != model.PaymentStatusSuccess {
		t.Errorf("expected payment status 20 (success), got %d", mockResp.Data.Status)
	}
	if mockResp.Data.PaidAt == nil {
		t.Errorf("expected paid_at to be set on payment")
	}

	// 验证订单状态流转至 20 (待发货) 并记录 paid_at
	var orderMain model.OrderMain
	_ = db.First(&orderMain, orderID)
	if orderMain.Status != model.OrderStatusPendingShipment {
		t.Errorf("expected order status 20 (pending shipment), got %d", orderMain.Status)
	}
	if orderMain.PaidAt == nil {
		t.Errorf("expected order paid_at to be set")
	}

	// 验证实际扣减库存：stock 50 -> 48, locked_stock 2 -> 0
	_ = db.Where("sku_id = ?", skuID).First(&inv)
	if inv.Stock != 48 || inv.LockedStock != 0 {
		t.Errorf("expected stock=48, locked_stock=0 after pay, got stock=%d, locked=%d", inv.Stock, inv.LockedStock)
	}

	// 验证库存流水记录为 order_out 且 delta = -2
	var invLog model.ProductSkuInventoryLog
	if err := db.Where("sku_id = ? AND change_type = ?", skuID, "order_out").First(&invLog).Error; err != nil {
		t.Fatalf("expected order_out inventory log: %v", err)
	}
	if invLog.Quantity != 2 || invLog.StockDelta != -2 || invLog.LockedDelta != -2 {
		t.Errorf("expected inv log qty=2, stock_delta=-2, locked_delta=-2, got qty=%d, sdelta=%d, ldelta=%d",
			invLog.Quantity, invLog.StockDelta, invLog.LockedDelta)
	}

	// 验证订单日志流转: 10 -> 20, action="pay"
	var payLog model.OrderLog
	if err := db.Where("order_id = ? AND action = ?", orderID, "pay").First(&payLog).Error; err != nil {
		t.Fatalf("expected pay order log: %v", err)
	}
	if payLog.FromStatus != model.OrderStatusPendingPayment || payLog.ToStatus != model.OrderStatusPendingShipment {
		t.Errorf("expected order log 10 -> 20, got %d -> %d", payLog.FromStatus, payLog.ToStatus)
	}
}

// 2. 重复支付幂等，重复回调不能重复扣库存
func TestOrderPayment_IdempotencyAndNoDuplicateStockDeduction(t *testing.T) {
	db, router, token1, memberID1, _, _ := setupOrderTestApp(t)
	_, skuID, addrID := createOrderTestData(t, db, memberID1)

	// 创建订单
	body, _ := json.Marshal(miniapporder.OrderCreateRequest{
		RequestID: "req-payment-idempotency-01",
		AddressID: addrID,
		Items:     []miniapporder.CheckoutItemRequest{{SKUID: skuID, Quantity: 1}},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/miniapp/orders/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token1)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var createResp response.Response[miniapporder.OrderDetailResponse]
	_ = json.Unmarshal(w.Body.Bytes(), &createResp)
	orderID := createResp.Data.ID

	// 创建支付单
	payCreateBody, _ := json.Marshal(miniapppayment.CreatePaymentRequest{OrderID: orderID})
	pReq := httptest.NewRequest(http.MethodPost, "/api/miniapp/payments/create", bytes.NewReader(payCreateBody))
	pReq.Header.Set("Content-Type", "application/json")
	pReq.Header.Set("Authorization", "Bearer "+token1)
	pW := httptest.NewRecorder()
	router.ServeHTTP(pW, pReq)
	var pResp response.Response[miniapppayment.PaymentResponse]
	_ = json.Unmarshal(pW.Body.Bytes(), &pResp)
	paymentNo := pResp.Data.PaymentNo

	// 第一次模拟支付成功
	mockBody, _ := json.Marshal(miniapppayment.MockPayRequest{PaymentNo: paymentNo})
	mReq1 := httptest.NewRequest(http.MethodPost, "/api/miniapp/payments/mock-pay", bytes.NewReader(mockBody))
	mReq1.Header.Set("Content-Type", "application/json")
	mReq1.Header.Set("Authorization", "Bearer "+token1)
	mW1 := httptest.NewRecorder()
	router.ServeHTTP(mW1, mReq1)
	var mResp1 response.Response[miniapppayment.PaymentResponse]
	_ = json.Unmarshal(mW1.Body.Bytes(), &mResp1)
	if mResp1.Code != 1 {
		t.Fatalf("first pay failed: %s", mResp1.Message)
	}

	// 第二次模拟支付成功（模拟重复网络重试或重复回调）
	mReq2 := httptest.NewRequest(http.MethodPost, "/api/miniapp/payments/mock-pay", bytes.NewReader(mockBody))
	mReq2.Header.Set("Content-Type", "application/json")
	mReq2.Header.Set("Authorization", "Bearer "+token1)
	mW2 := httptest.NewRecorder()
	router.ServeHTTP(mW2, mReq2)
	var mResp2 response.Response[miniapppayment.PaymentResponse]
	_ = json.Unmarshal(mW2.Body.Bytes(), &mResp2)
	if mResp2.Code != 1 {
		t.Fatalf("second pay should succeed idempotently, got code=%d: %s", mResp2.Code, mResp2.Message)
	}

	// 验证库存只扣减了1件（50 -> 49），绝不重复扣库存
	var inv model.ProductSkuInventory
	_ = db.Where("sku_id = ?", skuID).First(&inv)
	if inv.Stock != 49 || inv.LockedStock != 0 {
		t.Fatalf("duplicate payment deducted stock twice! stock=%d, locked=%d", inv.Stock, inv.LockedStock)
	}

	// 验证已支付订单不能再次创建支付单
	payReqFail := httptest.NewRequest(http.MethodPost, "/api/miniapp/payments/create", bytes.NewReader(payCreateBody))
	payReqFail.Header.Set("Content-Type", "application/json")
	payReqFail.Header.Set("Authorization", "Bearer "+token1)
	payWFail := httptest.NewRecorder()
	router.ServeHTTP(payWFail, payReqFail)
	var failResp response.Response[any]
	_ = json.Unmarshal(payWFail.Body.Bytes(), &failResp)
	if failResp.Code != 0 {
		t.Errorf("expected create payment to fail on paid order, got code=%d", failResp.Code)
	}
}

// 3. 已取消订单不能再次支付
func TestOrderPayment_CannotPayCancelledOrder(t *testing.T) {
	db, router, token1, memberID1, _, _ := setupOrderTestApp(t)
	_, skuID, addrID := createOrderTestData(t, db, memberID1)

	// 创建订单
	body, _ := json.Marshal(miniapporder.OrderCreateRequest{
		RequestID: "req-payment-cancel-01",
		AddressID: addrID,
		Items:     []miniapporder.CheckoutItemRequest{{SKUID: skuID, Quantity: 1}},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/miniapp/orders/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token1)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var createResp response.Response[miniapporder.OrderDetailResponse]
	_ = json.Unmarshal(w.Body.Bytes(), &createResp)
	orderID := createResp.Data.ID

	// 创建支付单
	payCreateBody, _ := json.Marshal(miniapppayment.CreatePaymentRequest{OrderID: orderID})
	pReq := httptest.NewRequest(http.MethodPost, "/api/miniapp/payments/create", bytes.NewReader(payCreateBody))
	pReq.Header.Set("Content-Type", "application/json")
	pReq.Header.Set("Authorization", "Bearer "+token1)
	pW := httptest.NewRecorder()
	router.ServeHTTP(pW, pReq)
	var pResp response.Response[miniapppayment.PaymentResponse]
	_ = json.Unmarshal(pW.Body.Bytes(), &pResp)
	paymentNo := pResp.Data.PaymentNo

	// 用户取消订单
	cReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/miniapp/orders/%d/cancel", orderID), nil)
	cReq.Header.Set("Authorization", "Bearer "+token1)
	cW := httptest.NewRecorder()
	router.ServeHTTP(cW, cReq)
	var cResp response.Response[any]
	_ = json.Unmarshal(cW.Body.Bytes(), &cResp)
	if cResp.Code != 1 {
		t.Fatalf("cancel order failed: %s", cResp.Message)
	}

	// 确认锁定库存已释放 (stock=50, locked=0)
	var inv model.ProductSkuInventory
	_ = db.Where("sku_id = ?", skuID).First(&inv)
	if inv.Stock != 50 || inv.LockedStock != 0 {
		t.Fatalf("expected stock=50, locked=0 after cancel, got stock=%d, locked=%d", inv.Stock, inv.LockedStock)
	}

	// 尝试对已取消订单进行支付 -> 必须被拒绝
	mockBody, _ := json.Marshal(miniapppayment.MockPayRequest{PaymentNo: paymentNo})
	mReq := httptest.NewRequest(http.MethodPost, "/api/miniapp/payments/mock-pay", bytes.NewReader(mockBody))
	mReq.Header.Set("Content-Type", "application/json")
	mReq.Header.Set("Authorization", "Bearer "+token1)
	mW := httptest.NewRecorder()
	router.ServeHTTP(mW, mReq)
	var mResp response.Response[any]
	_ = json.Unmarshal(mW.Body.Bytes(), &mResp)
	if mResp.Code != 0 {
		t.Fatalf("expected payment on cancelled order to fail, but got success")
	}

	// 确认库存仍保持 50 且 locked_stock 仍为 0
	_ = db.Where("sku_id = ?", skuID).First(&inv)
	if inv.Stock != 50 || inv.LockedStock != 0 {
		t.Fatalf("illegal pay on cancelled order modified inventory: stock=%d, locked=%d", inv.Stock, inv.LockedStock)
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

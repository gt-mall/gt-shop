package order_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/casbin/casbin/v2"
	casbinmodel "github.com/casbin/casbin/v2/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"mall/admin_core/bootstrap"
	"mall/admin_core/internal/auth"
	miniapporder "mall/admin_core/internal/miniapp/order"
	"mall/admin_core/internal/model"
	orderaftersale "mall/admin_core/internal/order/aftersale"
	orderorder "mall/admin_core/internal/order/order"
	"mall/admin_core/internal/order/refund"
	"mall/admin_core/pkg/bizno"
	"mall/admin_core/pkg/response"
	"mall/admin_core/routers"
)

type testAfterSaleEnv struct {
	db             *gorm.DB
	handler        http.Handler
	admin          model.SystemUser
	adminToken     string
	memberID       int64
	otherMemberID  int64
	asService      *orderaftersale.Service
	refundProvider *refund.MockProvider
	bizNoGen       *bizno.Generator
}

func setupAfterSaleTestEnv(t *testing.T) *testAfterSaleEnv {
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
		&model.SystemUser{},
		&model.SystemRole{},
		&model.Member{},
		&model.MemberAddress{},
		&model.Product{},
		&model.ProductSKU{},
		&model.ProductSkuInventory{},
		&model.ProductSkuInventoryLog{},
		&model.OrderMain{},
		&model.OrderItem{},
		&model.OrderAddress{},
		&model.OrderPayment{},
		&model.OrderLog{},
		&model.AfterSale{},
		&model.AfterSaleItem{},
		&model.AfterSaleLog{},
		&model.OrderRefund{},
		&model.SystemFile{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// Super Admin
	admin := model.SystemUser{
		Username: "admin",
		Nickname: "管理员",
		Status:   1,
	}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin user: %v", err)
	}

	role := model.SystemRole{
		Code:   auth.SuperAdminRoleCode,
		Name:   "超级管理员",
		Status: 1,
	}
	if err := db.Create(&role).Error; err != nil {
		t.Fatalf("create role: %v", err)
	}

	m := casbinmodel.NewModel()
	m.AddDef("r", "r", "sub, perm")
	m.AddDef("p", "p", "sub, perm")
	m.AddDef("g", "g", "_, _")
	m.AddDef("e", "e", "some(where (p_eft == allow))")
	m.AddDef("m", "m", "g(r_sub, p_sub) && r_perm == p_perm")
	enforcer, err := casbin.NewSyncedEnforcer(m)
	if err != nil {
		t.Fatalf("create enforcer: %v", err)
	}
	_, _ = enforcer.AddRoleForUser(auth.UserSubject(admin.ID), auth.SuperAdminRoleCode)

	authService := auth.NewService(db, enforcer, testAdminSecret, 24*time.Hour)
	adminClaims := jwt.RegisteredClaims{
		Issuer:    "mall-admin-core",
		Subject:   strconv.FormatInt(admin.ID, 10),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
	}
	adminToken, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, adminClaims).SignedString([]byte(testAdminSecret))

	// Members
	mobile1 := "13800138000"
	member1 := model.Member{Nickname: "会员1", Mobile: &mobile1, Status: 1}
	_ = db.Create(&member1)

	mobile2 := "13900139000"
	member2 := model.Member{Nickname: "会员2", Mobile: &mobile2, Status: 1}
	_ = db.Create(&member2)

	bGen, err := bizno.NewGenerator(0)
	if err != nil {
		t.Fatalf("new bizno gen: %v", err)
	}
	mockRefProvider := refund.NewMockProvider()
	asService := orderaftersale.NewService(db, bGen, mockRefProvider)

	bConfig := bootstrap.Config{
		JWTSecret:     "miniapp_secret",
		JWTExpireHour: 24,
		BizNoWorkerID: 0,
	}
	router := routers.NewRouter(bConfig, db, authService)

	return &testAfterSaleEnv{
		db:             db,
		handler:        router,
		admin:          admin,
		adminToken:     adminToken,
		memberID:       member1.ID,
		otherMemberID:  member2.ID,
		asService:      asService,
		refundProvider: mockRefProvider,
		bizNoGen:       bGen,
	}
}

// helper to create an order with inventory for testing
func createTestOrder(t *testing.T, db *gorm.DB, memberID int64, status int8, initialStock int) (int64, int64, int64) {
	t.Helper()
	now := time.Now()

	prod := model.Product{
		Name:       "测试商品",
		Status:     1,
		SalesCount: 0,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	_ = db.Create(&prod)

	code := "SKU-TEST-001"
	sku := model.ProductSKU{
		ProductID:   prod.ID,
		SKUCode:     &code,
		SpecValues:  "[]",
		Price:       "99.00",
		CostPrice:   "50.00",
		MarketPrice: "120.00",
		Status:      1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	_ = db.Create(&sku)

	inv := model.ProductSkuInventory{
		ProductID: prod.ID,
		SKUID:     sku.ID,
		Stock:     initialStock,
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_ = db.Create(&inv)

	orderNo := fmt.Sprintf("D%d", time.Now().UnixNano())
	reqID := fmt.Sprintf("req-%d", time.Now().UnixNano())

	var paidAt, shippedAt, completedAt, cancelledAt *time.Time
	if status >= model.OrderStatusPendingShipment && status != model.OrderStatusCancelled {
		paidAt = &now
	}
	if status >= model.OrderStatusPendingReceipt && status != model.OrderStatusCancelled {
		shippedAt = &now
	}
	if status == model.OrderStatusCompleted {
		completedAt = &now
	}
	if status == model.OrderStatusCancelled {
		cancelledAt = &now
	}

	order := model.OrderMain{
		OrderNo:       orderNo,
		MemberID:      memberID,
		RequestID:     reqID,
		Status:        status,
		TotalAmount:   "99.00",
		PayAmount:     "99.00",
		FreightAmount: "0.00",
		PaidAt:        paidAt,
		ShippedAt:     shippedAt,
		CompletedAt:   completedAt,
		CancelledAt:   cancelledAt,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	_ = db.Create(&order)

	item := model.OrderItem{
		OrderID:     order.ID,
		ProductID:   prod.ID,
		SKUID:       sku.ID,
		ProductName: prod.Name,
		SKUCode:     *sku.SKUCode,
		UnitPrice:   sku.Price,
		Quantity:    1,
		TotalAmount: "99.00",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	_ = db.Create(&item)

	if paidAt != nil {
		payNo := fmt.Sprintf("P%d", time.Now().UnixNano())
		txnNo := fmt.Sprintf("TXN_%s", payNo)
		payment := model.OrderPayment{
			PaymentNo:     payNo,
			OrderID:       order.ID,
			Channel:       "mock",
			Amount:        "99.00",
			Status:        model.PaymentStatusSuccess,
			TransactionNo: &txnNo,
			PaidAt:        paidAt,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		_ = db.Create(&payment)
	}

	return order.ID, prod.ID, sku.ID
}

func TestAfterSaleEligibility(t *testing.T) {
	env := setupAfterSaleTestEnv(t)
	ctx := context.Background()

	// 1. Pending payment (10) cannot apply
	o10, _, _ := createTestOrder(t, env.db, env.memberID, model.OrderStatusPendingPayment, 10)
	_, err := env.asService.Apply(ctx, env.memberID, orderaftersale.MiniappApplyRequest{
		OrderID:   o10,
		Type:      model.AfterSaleTypeRefundOnly,
		Reason:    "不想要了",
		RequestID: "req-10-1",
	})
	if err == nil {
		t.Fatal("expected pending payment order to fail after-sale apply")
	}

	// 2. Pending shipment (20): can apply RefundOnly (1), cannot apply ReturnRefund (2)
	o20, _, _ := createTestOrder(t, env.db, env.memberID, model.OrderStatusPendingShipment, 10)
	_, err = env.asService.Apply(ctx, env.memberID, orderaftersale.MiniappApplyRequest{
		OrderID:   o20,
		Type:      model.AfterSaleTypeReturnRefund,
		Reason:    "退货退款",
		RequestID: "req-20-return",
	})
	if err == nil {
		t.Fatal("expected pending shipment order to reject ReturnRefund")
	}

	as20, err := env.asService.Apply(ctx, env.memberID, orderaftersale.MiniappApplyRequest{
		OrderID:   o20,
		Type:      model.AfterSaleTypeRefundOnly,
		Reason:    "仅退款",
		RequestID: "req-20-refund",
	})
	if err != nil {
		t.Fatalf("expected pending shipment order to accept RefundOnly, got err: %v", err)
	}
	if as20.Status != model.AfterSaleStatusPendingReview {
		t.Fatalf("expected status 10, got %d", as20.Status)
	}

	// 3. Pending receipt (30): can apply RefundOnly (1) AND ReturnRefund (2)
	o30a, _, _ := createTestOrder(t, env.db, env.memberID, model.OrderStatusPendingReceipt, 10)
	as30a, err := env.asService.Apply(ctx, env.memberID, orderaftersale.MiniappApplyRequest{
		OrderID:   o30a,
		Type:      model.AfterSaleTypeRefundOnly,
		Reason:    "仅退款",
		RequestID: "req-30-refund",
	})
	if err != nil {
		t.Fatalf("expected pending receipt to accept RefundOnly: %v", err)
	}
	if as30a.Type != model.AfterSaleTypeRefundOnly {
		t.Fatalf("expected type 1, got %d", as30a.Type)
	}

	o30b, _, _ := createTestOrder(t, env.db, env.memberID, model.OrderStatusPendingReceipt, 10)
	as30b, err := env.asService.Apply(ctx, env.memberID, orderaftersale.MiniappApplyRequest{
		OrderID:   o30b,
		Type:      model.AfterSaleTypeReturnRefund,
		Reason:    "退货退款",
		RequestID: "req-30-return",
	})
	if err != nil {
		t.Fatalf("expected pending receipt to accept ReturnRefund: %v", err)
	}
	if as30b.Type != model.AfterSaleTypeReturnRefund {
		t.Fatalf("expected type 2, got %d", as30b.Type)
	}

	// 4. Completed order (40): within 7 days allows ReturnRefund (2), rejects RefundOnly (1)
	o40, _, _ := createTestOrder(t, env.db, env.memberID, model.OrderStatusCompleted, 10)
	_, err = env.asService.Apply(ctx, env.memberID, orderaftersale.MiniappApplyRequest{
		OrderID:   o40,
		Type:      model.AfterSaleTypeRefundOnly,
		Reason:    "仅退款",
		RequestID: "req-40-refund",
	})
	if err == nil {
		t.Fatal("expected completed order to reject RefundOnly")
	}

	as40, err := env.asService.Apply(ctx, env.memberID, orderaftersale.MiniappApplyRequest{
		OrderID:   o40,
		Type:      model.AfterSaleTypeReturnRefund,
		Reason:    "退货退款",
		RequestID: "req-40-return",
	})
	if err != nil {
		t.Fatalf("expected completed order to accept ReturnRefund: %v", err)
	}
	if as40.Type != model.AfterSaleTypeReturnRefund {
		t.Fatalf("expected type 2, got %d", as40.Type)
	}

	// Completed order expired (> 7 days) cannot apply
	o40Exp, _, _ := createTestOrder(t, env.db, env.memberID, model.OrderStatusCompleted, 10)
	eightDaysAgo := time.Now().Add(-8 * 24 * time.Hour)
	env.db.Model(&model.OrderMain{}).Where("id = ?", o40Exp).Update("completed_at", &eightDaysAgo)
	_, err = env.asService.Apply(ctx, env.memberID, orderaftersale.MiniappApplyRequest{
		OrderID:   o40Exp,
		Type:      model.AfterSaleTypeReturnRefund,
		Reason:    "退货退款",
		RequestID: "req-40-expired",
	})
	if err == nil {
		t.Fatal("expected expired completed order to be rejected")
	}

	// 5. Cancelled (50) and Refunded (60) cannot apply
	o50, _, _ := createTestOrder(t, env.db, env.memberID, model.OrderStatusCancelled, 10)
	_, err = env.asService.Apply(ctx, env.memberID, orderaftersale.MiniappApplyRequest{
		OrderID:   o50,
		Type:      model.AfterSaleTypeRefundOnly,
		Reason:    "仅退款",
		RequestID: "req-50",
	})
	if err == nil {
		t.Fatal("expected cancelled order to be rejected")
	}

	o60, _, _ := createTestOrder(t, env.db, env.memberID, model.OrderStatusRefunded, 10)
	_, err = env.asService.Apply(ctx, env.memberID, orderaftersale.MiniappApplyRequest{
		OrderID:   o60,
		Type:      model.AfterSaleTypeRefundOnly,
		Reason:    "仅退款",
		RequestID: "req-60",
	})
	if err == nil {
		t.Fatal("expected refunded order to be rejected")
	}

	// 6. Non-owner cannot apply
	oOther, _, _ := createTestOrder(t, env.db, env.memberID, model.OrderStatusPendingShipment, 10)
	_, err = env.asService.Apply(ctx, env.otherMemberID, orderaftersale.MiniappApplyRequest{
		OrderID:   oOther,
		Type:      model.AfterSaleTypeRefundOnly,
		Reason:    "仅退款",
		RequestID: "req-other",
	})
	if err == nil {
		t.Fatal("expected non-owner to be rejected")
	}

	// 7. Same order cannot apply when active after-sale exists
	_, err = env.asService.Apply(ctx, env.memberID, orderaftersale.MiniappApplyRequest{
		OrderID:   o20,
		Type:      model.AfterSaleTypeRefundOnly,
		Reason:    "重复申请",
		RequestID: "req-20-repeat",
	})
	if err == nil {
		t.Fatal("expected active after-sale to block new apply")
	}

	// 8. request_id idempotency
	sameAS, err := env.asService.Apply(ctx, env.memberID, orderaftersale.MiniappApplyRequest{
		OrderID:   o20,
		Type:      model.AfterSaleTypeRefundOnly,
		Reason:    "仅退款",
		RequestID: "req-20-refund", // same request_id
	})
	if err != nil {
		t.Fatalf("expected idempotent success: %v", err)
	}
	if sameAS.ID != as20.ID {
		t.Fatalf("expected same after-sale ID %d, got %d", as20.ID, sameAS.ID)
	}
}

func TestAfterSaleLifecycleRefundOnly(t *testing.T) {
	env := setupAfterSaleTestEnv(t)
	ctx := context.Background()

	// Initial stock is 10. Order 20 was created and paid (deducted from stock).
	orderID, _, skuID := createTestOrder(t, env.db, env.memberID, model.OrderStatusPendingShipment, 10)

	// 1. User applies RefundOnly (1)
	as, err := env.asService.Apply(ctx, env.memberID, orderaftersale.MiniappApplyRequest{
		OrderID:   orderID,
		Type:      model.AfterSaleTypeRefundOnly,
		Reason:    "多买了",
		RequestID: "req-refund-only-flow",
	})
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	if as.Status != model.AfterSaleStatusPendingReview {
		t.Fatalf("want status 10, got %d", as.Status)
	}

	// 2. Admin approves RefundOnly: 10 -> 40 -> 50 (success)
	err = env.asService.AdminApprove(ctx, env.admin, orderaftersale.AdminApproveRequest{
		ID: as.ID,
	})
	if err != nil {
		t.Fatalf("admin approve failed: %v", err)
	}

	// Verify AfterSale completed (50)
	var asDb model.AfterSale
	_ = env.db.Where("id = ?", as.ID).First(&asDb)
	if asDb.Status != model.AfterSaleStatusCompleted {
		t.Fatalf("expected after_sale status 50, got %d", asDb.Status)
	}
	if asDb.CompletedAt == nil {
		t.Fatal("expected completed_at to be set")
	}

	// Verify OrderMain status 60 (Refunded)
	var orderDb model.OrderMain
	_ = env.db.Where("id = ?", orderID).First(&orderDb)
	if orderDb.Status != model.OrderStatusRefunded {
		t.Fatalf("expected order status 60, got %d", orderDb.Status)
	}

	// Verify OrderRefund status 30 (Success)
	var refDb model.OrderRefund
	_ = env.db.Where("after_sale_id = ?", as.ID).First(&refDb)
	if refDb.Status != model.RefundStatusSuccess {
		t.Fatalf("expected refund status 30, got %d", refDb.Status)
	}
	if refDb.ProviderRefundNo == nil || *refDb.ProviderRefundNo == "" {
		t.Fatal("expected provider_refund_no to be set")
	}

	// Verify inventory: Pending Shipment + Refund Only MUST restore stock: 10 + 1 = 11
	var inv model.ProductSkuInventory
	_ = env.db.Where("sku_id = ?", skuID).First(&inv)
	if inv.Stock != 11 {
		t.Fatalf("expected stock 11, got %d", inv.Stock)
	}

	// Verify idempotency: repeated FinalizeRefundSuccess does not duplicate stock or change
	err = env.asService.FinalizeRefundSuccess(ctx, as.ID, "DUPLICATE_TXN")
	if err != nil {
		t.Fatalf("repeat finalize failed: %v", err)
	}
	_ = env.db.Where("sku_id = ?", skuID).First(&inv)
	if inv.Stock != 11 {
		t.Fatalf("stock was duplicated on repeat finalize! got %d", inv.Stock)
	}
}

func TestAfterSaleLifecycleReturnRefund(t *testing.T) {
	env := setupAfterSaleTestEnv(t)
	ctx := context.Background()

	// Completed order with stock 10
	orderID, _, skuID := createTestOrder(t, env.db, env.memberID, model.OrderStatusCompleted, 10)

	// 1. User applies ReturnRefund (2)
	as, err := env.asService.Apply(ctx, env.memberID, orderaftersale.MiniappApplyRequest{
		OrderID:   orderID,
		Type:      model.AfterSaleTypeReturnRefund,
		Reason:    "尺码不合适",
		RequestID: "req-return-flow",
	})
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}

	// 2. Admin approves ReturnRefund: 10 -> 20 (WaitingBuyerReturn)
	// Return address is mandatory
	err = env.asService.AdminApprove(ctx, env.admin, orderaftersale.AdminApproveRequest{
		ID: as.ID,
	})
	if err == nil {
		t.Fatal("expected approve to fail without return address")
	}

	err = env.asService.AdminApprove(ctx, env.admin, orderaftersale.AdminApproveRequest{
		ID:             as.ID,
		ReturnReceiver: "售后仓库",
		ReturnPhone:    "13800000000",
		ReturnAddress:  "浙江省杭州市西湖区电商产业园A座101",
	})
	if err != nil {
		t.Fatalf("approve failed: %v", err)
	}

	var asDb model.AfterSale
	_ = env.db.Where("id = ?", as.ID).First(&asDb)
	if asDb.Status != model.AfterSaleStatusWaitingBuyerReturn {
		t.Fatalf("expected status 20, got %d", asDb.Status)
	}
	if asDb.ReturnReceiver != "售后仓库" || asDb.ReturnPhone != "13800000000" {
		t.Fatal("return address snapshot not saved")
	}

	// 3. Buyer submits tracking info: 20 -> 30
	// Non-buyer cannot submit
	err = env.asService.SubmitReturnLogistics(ctx, env.otherMemberID, as.ID, "顺丰速运", "SF123456789")
	if err == nil {
		t.Fatal("expected non-owner submit to fail")
	}
	err = env.asService.SubmitReturnLogistics(ctx, env.memberID, as.ID, "顺丰速运", "SF123456789")
	if err != nil {
		t.Fatalf("submit logistics failed: %v", err)
	}
	_ = env.db.Where("id = ?", as.ID).First(&asDb)
	if asDb.Status != model.AfterSaleStatusWaitingMerchantReceive {
		t.Fatalf("expected status 30, got %d", asDb.Status)
	}
	if asDb.BuyerReturnCompany != "顺丰速运" || asDb.BuyerReturnNo != "SF123456789" {
		t.Fatal("tracking info not saved")
	}

	// 4. Admin confirms return receipt: 30 -> 40 -> 50
	var asItem model.AfterSaleItem
	_ = env.db.Where("after_sale_id = ?", as.ID).First(&asItem)

	// Restock quantity cannot exceed item.Quantity (1)
	err = env.asService.AdminConfirmReturnReceived(ctx, env.admin, orderaftersale.AdminConfirmReceiveRequest{
		ID: as.ID,
		Items: []orderaftersale.AdminConfirmReceiveItem{
			{AfterSaleItemID: asItem.ID, RestockQuantity: 5},
		},
	})
	if err == nil {
		t.Fatal("expected invalid restock quantity to fail")
	}

	// Restock 1 item
	err = env.asService.AdminConfirmReturnReceived(ctx, env.admin, orderaftersale.AdminConfirmReceiveRequest{
		ID: as.ID,
		Items: []orderaftersale.AdminConfirmReceiveItem{
			{AfterSaleItemID: asItem.ID, RestockQuantity: 1},
		},
	})
	if err != nil {
		t.Fatalf("confirm return received failed: %v", err)
	}

	// Verify inventory restocked: 10 + 1 = 11
	var inv model.ProductSkuInventory
	_ = env.db.Where("sku_id = ?", skuID).First(&inv)
	if inv.Stock != 11 {
		t.Fatalf("expected stock 11, got %d", inv.Stock)
	}

	// Verify after_sale 50 and order 60
	_ = env.db.Where("id = ?", as.ID).First(&asDb)
	if asDb.Status != model.AfterSaleStatusCompleted {
		t.Fatalf("expected status 50, got %d", asDb.Status)
	}
	var orderDb model.OrderMain
	_ = env.db.Where("id = ?", orderID).First(&orderDb)
	if orderDb.Status != model.OrderStatusRefunded {
		t.Fatalf("expected order status 60, got %d", orderDb.Status)
	}

	// Verify restock_quantity snapshot saved
	_ = env.db.Where("id = ?", asItem.ID).First(&asItem)
	if asItem.RestockQuantity != 1 {
		t.Fatalf("expected restock_quantity 1, got %d", asItem.RestockQuantity)
	}
}

func TestPendingReceiptRefundOnlyDoesNotRestoreStock(t *testing.T) {
	env := setupAfterSaleTestEnv(t)
	ctx := context.Background()

	// Pending receipt (30): shipped goods, customer requested refund only
	orderID, _, skuID := createTestOrder(t, env.db, env.memberID, model.OrderStatusPendingReceipt, 10)

	as, err := env.asService.Apply(ctx, env.memberID, orderaftersale.MiniappApplyRequest{
		OrderID:   orderID,
		Type:      model.AfterSaleTypeRefundOnly,
		Reason:    "未收到货协商仅退款",
		RequestID: "req-p-receipt-refund",
	})
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}

	err = env.asService.AdminApprove(ctx, env.admin, orderaftersale.AdminApproveRequest{ID: as.ID})
	if err != nil {
		t.Fatalf("approve failed: %v", err)
	}

	// Verify status completed
	var asDb model.AfterSale
	_ = env.db.Where("id = ?", as.ID).First(&asDb)
	if asDb.Status != model.AfterSaleStatusCompleted {
		t.Fatalf("expected status 50, got %d", asDb.Status)
	}

	// Verify inventory: Pending Receipt + Refund Only MUST NOT restore stock: stock remains 10!
	var inv model.ProductSkuInventory
	_ = env.db.Where("sku_id = ?", skuID).First(&inv)
	if inv.Stock != 10 {
		t.Fatalf("pending receipt refund only MUST NOT restore stock! got %d, want 10", inv.Stock)
	}
}

func TestRefundFailureAndRetry(t *testing.T) {
	env := setupAfterSaleTestEnv(t)
	ctx := context.Background()

	orderID, _, _ := createTestOrder(t, env.db, env.memberID, model.OrderStatusPendingShipment, 10)

	as, err := env.asService.Apply(ctx, env.memberID, orderaftersale.MiniappApplyRequest{
		OrderID:   orderID,
		Type:      model.AfterSaleTypeRefundOnly,
		Reason:    "退款重试测试",
		RequestID: "req-refund-fail-retry",
	})
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}

	// Configure mock refund provider to fail
	env.refundProvider.SetFailure(true, "银行通道维护中")

	err = env.asService.AdminApprove(ctx, env.admin, orderaftersale.AdminApproveRequest{ID: as.ID})
	if err != nil {
		t.Fatalf("admin approve error: %v", err)
	}

	// Verify AfterSale is 80 (RefundFailed)
	var asDb model.AfterSale
	_ = env.db.Where("id = ?", as.ID).First(&asDb)
	if asDb.Status != model.AfterSaleStatusRefundFailed {
		t.Fatalf("expected status 80, got %d", asDb.Status)
	}

	// Verify OrderRefund is 40 (Failed)
	var refDb model.OrderRefund
	_ = env.db.Where("after_sale_id = ?", as.ID).First(&refDb)
	if refDb.Status != model.RefundStatusFailed {
		t.Fatalf("expected refund status 40, got %d", refDb.Status)
	}
	if refDb.LastError != "银行通道维护中" {
		t.Fatalf("expected error message recorded, got: %s", refDb.LastError)
	}
	originalRefundNo := refDb.RefundNo

	// Admin retries with provider still failing -> still 80/40, retry_count incremented
	_ = env.asService.AdminRetryRefund(ctx, env.admin, orderaftersale.AdminRetryRefundRequest{ID: as.ID})
	_ = env.db.Where("after_sale_id = ?", as.ID).First(&refDb)
	if refDb.RetryCount != 2 {
		t.Fatalf("expected retry_count 2, got %d", refDb.RetryCount)
	}

	// Configure mock provider to succeed
	env.refundProvider.SetFailure(false, "")

	// Retry again -> succeeds
	err = env.asService.AdminRetryRefund(ctx, env.admin, orderaftersale.AdminRetryRefundRequest{ID: as.ID})
	if err != nil {
		t.Fatalf("retry refund failed: %v", err)
	}

	_ = env.db.Where("id = ?", as.ID).First(&asDb)
	if asDb.Status != model.AfterSaleStatusCompleted {
		t.Fatalf("expected status 50, got %d", asDb.Status)
	}

	_ = env.db.Where("after_sale_id = ?", as.ID).First(&refDb)
	if refDb.Status != model.RefundStatusSuccess {
		t.Fatalf("expected refund status 30, got %d", refDb.Status)
	}
	// Verify SAME refund_no was used
	if refDb.RefundNo != originalRefundNo {
		t.Fatalf("refund_no changed during retry! original: %s, new: %s", originalRefundNo, refDb.RefundNo)
	}
}

func TestUserCancelAndAdminReject(t *testing.T) {
	env := setupAfterSaleTestEnv(t)
	ctx := context.Background()

	// 1. User cancels after-sale (10 -> 70)
	orderID1, _, _ := createTestOrder(t, env.db, env.memberID, model.OrderStatusPendingShipment, 10)
	as1, _ := env.asService.Apply(ctx, env.memberID, orderaftersale.MiniappApplyRequest{
		OrderID:   orderID1,
		Type:      model.AfterSaleTypeRefundOnly,
		Reason:    "取消测试",
		RequestID: "req-cancel-1",
	})
	err := env.asService.Cancel(ctx, env.memberID, as1.ID)
	if err != nil {
		t.Fatalf("cancel failed: %v", err)
	}
	var asDb1 model.AfterSale
	_ = env.db.Where("id = ?", as1.ID).First(&asDb1)
	if asDb1.Status != model.AfterSaleStatusCancelled {
		t.Fatalf("expected status 70, got %d", asDb1.Status)
	}

	// After cancellation, user CAN re-apply
	as1Re, err := env.asService.Apply(ctx, env.memberID, orderaftersale.MiniappApplyRequest{
		OrderID:   orderID1,
		Type:      model.AfterSaleTypeRefundOnly,
		Reason:    "重新申请",
		RequestID: "req-cancel-reapply",
	})
	if err != nil {
		t.Fatalf("re-apply after cancel failed: %v", err)
	}
	if as1Re.ID == as1.ID {
		t.Fatal("re-apply should create a new after-sale record")
	}

	// 2. Admin rejects after-sale (10 -> 60)
	orderID2, _, _ := createTestOrder(t, env.db, env.memberID, model.OrderStatusPendingShipment, 10)
	as2, _ := env.asService.Apply(ctx, env.memberID, orderaftersale.MiniappApplyRequest{
		OrderID:   orderID2,
		Type:      model.AfterSaleTypeRefundOnly,
		Reason:    "拒绝测试",
		RequestID: "req-reject-1",
	})
	err = env.asService.AdminReject(ctx, env.admin, orderaftersale.AdminRejectRequest{
		ID:           as2.ID,
		RejectReason: "商品已打包发货",
	})
	if err != nil {
		t.Fatalf("reject failed: %v", err)
	}
	var asDb2 model.AfterSale
	_ = env.db.Where("id = ?", as2.ID).First(&asDb2)
	if asDb2.Status != model.AfterSaleStatusRejected {
		t.Fatalf("expected status 60, got %d", asDb2.Status)
	}
	if asDb2.RejectReason != "商品已打包发货" {
		t.Fatalf("unexpected reject reason: %s", asDb2.RejectReason)
	}

	// After rejection, user CAN re-apply
	_, err = env.asService.Apply(ctx, env.memberID, orderaftersale.MiniappApplyRequest{
		OrderID:   orderID2,
		Type:      model.AfterSaleTypeRefundOnly,
		Reason:    "重新申请仅退款",
		RequestID: "req-reject-reapply",
	})
	if err != nil {
		t.Fatalf("re-apply after reject failed: %v", err)
	}
}

func TestActiveAfterSaleBlocksShipAndConfirm(t *testing.T) {
	env := setupAfterSaleTestEnv(t)
	ctx := context.Background()

	// 1. Active after-sale blocks Ship
	orderID1, _, _ := createTestOrder(t, env.db, env.memberID, model.OrderStatusPendingShipment, 10)
	_, _ = env.asService.Apply(ctx, env.memberID, orderaftersale.MiniappApplyRequest{
		OrderID:   orderID1,
		Type:      model.AfterSaleTypeRefundOnly,
		Reason:    "阻止发货测试",
		RequestID: "req-block-ship",
	})

	orderSvc := orderorder.NewService(env.db)
	err := orderSvc.Ship(ctx, env.admin, orderorder.OrderShipRequest{
		ID:             orderID1,
		ExpressCompany: "顺丰速运",
		ExpressNo:      "SF99999",
	})
	if err == nil {
		t.Fatal("expected Ship to be blocked when active after-sale exists")
	}

	// 2. Active after-sale blocks ConfirmReceipt
	orderID2, _, _ := createTestOrder(t, env.db, env.memberID, model.OrderStatusPendingReceipt, 10)
	_, _ = env.asService.Apply(ctx, env.memberID, orderaftersale.MiniappApplyRequest{
		OrderID:   orderID2,
		Type:      model.AfterSaleTypeRefundOnly,
		Reason:    "阻止收货测试",
		RequestID: "req-block-confirm",
	})

	miniappOrderSvc := miniapporder.NewService(env.db, env.bizNoGen)
	err = miniappOrderSvc.Confirm(ctx, model.Member{ID: env.memberID}, orderID2)
	if err == nil {
		t.Fatal("expected Confirm to be blocked when active after-sale exists")
	}
}

func TestProofFileIDsValidation(t *testing.T) {
	env := setupAfterSaleTestEnv(t)
	ctx := context.Background()

	// Insert valid files in system_file
	now := time.Now()
	f1 := model.SystemFile{Storage: "local", Name: "1.jpg", Path: "1.jpg", Ext: "jpg", FileType: model.FileTypeImage, Scene: model.SceneAfterSaleProof, OwnerType: model.OwnerTypeMember, OwnerID: env.memberID, Visibility: model.VisibilityPrivate, Size: 100, CreatedAt: now}
	f2 := model.SystemFile{Storage: "local", Name: "2.jpg", Path: "2.jpg", Ext: "jpg", FileType: model.FileTypeImage, Scene: model.SceneAfterSaleProof, OwnerType: model.OwnerTypeMember, OwnerID: env.memberID, Visibility: model.VisibilityPrivate, Size: 100, CreatedAt: now}
	_ = env.db.Create(&f1)
	_ = env.db.Create(&f2)

	orderID, _, _ := createTestOrder(t, env.db, env.memberID, model.OrderStatusPendingShipment, 10)

	// Valid proof files with duplicate ID -> deduplicates and passes
	as, err := env.asService.Apply(ctx, env.memberID, orderaftersale.MiniappApplyRequest{
		OrderID:      orderID,
		Type:         model.AfterSaleTypeRefundOnly,
		Reason:       "凭证测试",
		ProofFileIDs: []int64{f1.ID, f2.ID, f1.ID}, // duplicate
		RequestID:    "req-proof-valid",
	})
	if err != nil {
		t.Fatalf("apply with valid proofs failed: %v", err)
	}
	if as.ProofFileIDs == nil || *as.ProofFileIDs != fmt.Sprintf("[%d,%d]", f1.ID, f2.ID) {
		t.Fatalf("expected proof_file_ids to be [%d,%d], got: %v", f1.ID, f2.ID, as.ProofFileIDs)
	}

	// Invalid file ID (not in system_file)
	orderID2, _, _ := createTestOrder(t, env.db, env.memberID, model.OrderStatusPendingShipment, 10)
	_, err = env.asService.Apply(ctx, env.memberID, orderaftersale.MiniappApplyRequest{
		OrderID:      orderID2,
		Type:         model.AfterSaleTypeRefundOnly,
		Reason:       "凭证测试非法",
		ProofFileIDs: []int64{999999},
		RequestID:    "req-proof-invalid",
	})
	if err == nil {
		t.Fatal("expected nonexistent proof_file_id to be rejected")
	}

	// Test cross-user file ownership theft (User A attempting to use User B's file)
	otherMemberFile := model.SystemFile{Storage: "local", Name: "other.jpg", Path: "other.jpg", Ext: "jpg", FileType: model.FileTypeImage, Scene: model.SceneAfterSaleProof, OwnerType: model.OwnerTypeMember, OwnerID: 99999, Visibility: model.VisibilityPrivate, Size: 100, CreatedAt: now}
	_ = env.db.Create(&otherMemberFile)
	orderIDTheft, _, _ := createTestOrder(t, env.db, env.memberID, model.OrderStatusPendingShipment, 10)
	_, err = env.asService.Apply(ctx, env.memberID, orderaftersale.MiniappApplyRequest{
		OrderID:      orderIDTheft,
		Type:         model.AfterSaleTypeRefundOnly,
		Reason:       "凭证越权测试",
		ProofFileIDs: []int64{otherMemberFile.ID},
		RequestID:    "req-proof-theft",
	})
	if err == nil {
		t.Fatal("expected cross-user proof file to be rejected")
	}

	// Test scene mismatch (cannot use admin_material as proof)
	adminMaterialFile := model.SystemFile{Storage: "local", Name: "mat.jpg", Path: "mat.jpg", Ext: "jpg", FileType: model.FileTypeImage, Scene: model.SceneAdminMaterial, OwnerType: model.OwnerTypeAdmin, OwnerID: 1, Visibility: model.VisibilityPublic, Size: 100, CreatedAt: now}
	_ = env.db.Create(&adminMaterialFile)
	orderIDScene, _, _ := createTestOrder(t, env.db, env.memberID, model.OrderStatusPendingShipment, 10)
	_, err = env.asService.Apply(ctx, env.memberID, orderaftersale.MiniappApplyRequest{
		OrderID:      orderIDScene,
		Type:         model.AfterSaleTypeRefundOnly,
		Reason:       "场景不匹配测试",
		ProofFileIDs: []int64{adminMaterialFile.ID},
		RequestID:    "req-proof-scene",
	})
	if err == nil {
		t.Fatal("expected admin_material file to be rejected as proof")
	}

	// More than 10 files rejected
	orderID3, _, _ := createTestOrder(t, env.db, env.memberID, model.OrderStatusPendingShipment, 10)
	_, err = env.asService.Apply(ctx, env.memberID, orderaftersale.MiniappApplyRequest{
		OrderID:      orderID3,
		Type:         model.AfterSaleTypeRefundOnly,
		Reason:       "凭证测试超限",
		ProofFileIDs: []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
		RequestID:    "req-proof-overflow",
	})
	if err == nil {
		t.Fatal("expected >10 proof files to be rejected")
	}
}

func TestAdminAfterSaleHTTPAPI(t *testing.T) {
	env := setupAfterSaleTestEnv(t)
	ctx := context.Background()

	orderID, _, _ := createTestOrder(t, env.db, env.memberID, model.OrderStatusPendingShipment, 10)
	as, _ := env.asService.Apply(ctx, env.memberID, orderaftersale.MiniappApplyRequest{
		OrderID:   orderID,
		Type:      model.AfterSaleTypeRefundOnly,
		Reason:    "接口测试",
		RequestID: "req-http-api",
	})

	// 1. List
	listBody, _ := json.Marshal(orderaftersale.AdminAfterSaleListRequest{Page: 1, PageSize: 10})
	req := httptest.NewRequest(http.MethodPost, "/api/order/after-sales/list", bytes.NewReader(listBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+env.adminToken)
	w := httptest.NewRecorder()
	env.handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list HTTP status: %d", w.Code)
	}

	var listResp response.Response[[]orderaftersale.AdminAfterSaleListRow]
	_ = json.Unmarshal(w.Body.Bytes(), &listResp)
	if listResp.Code != 1 || len(listResp.Data) == 0 {
		t.Fatalf("list response failed: %+v", listResp)
	}

	// 2. Detail
	detailBody, _ := json.Marshal(orderaftersale.AdminAfterSaleDetailRequest{ID: as.ID})
	req = httptest.NewRequest(http.MethodPost, "/api/order/after-sales/detail", bytes.NewReader(detailBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+env.adminToken)
	w = httptest.NewRecorder()
	env.handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("detail HTTP status: %d", w.Code)
	}
	var detailResp response.Response[orderaftersale.AdminAfterSaleDetailResponse]
	_ = json.Unmarshal(w.Body.Bytes(), &detailResp)
	if detailResp.Code != 1 || detailResp.Data.ID != as.ID {
		t.Fatalf("detail response failed: %+v", detailResp)
	}

	// 3. TabCounts
	tabReq := httptest.NewRequest(http.MethodPost, "/api/order/after-sales/tab-counts", bytes.NewReader([]byte("{}")))
	tabReq.Header.Set("Content-Type", "application/json")
	tabReq.Header.Set("Authorization", "Bearer "+env.adminToken)
	wTab := httptest.NewRecorder()
	env.handler.ServeHTTP(wTab, tabReq)
	if wTab.Code != http.StatusOK {
		t.Fatalf("tab-counts HTTP status: %d", wTab.Code)
	}
	var tabResp response.Response[orderaftersale.AdminAfterSaleTabCountsResponse]
	_ = json.Unmarshal(wTab.Body.Bytes(), &tabResp)
	if tabResp.Code != 1 || tabResp.Data.PendingReview < 1 || tabResp.Data.All < 1 {
		t.Fatalf("unexpected tab-counts: %+v", tabResp)
	}

	// 4. List with Tab
	listTabBody, _ := json.Marshal(orderaftersale.AdminAfterSaleListRequest{
		Page:     1,
		PageSize: 10,
		Tab:      "pending_review",
	})
	reqTabList := httptest.NewRequest(http.MethodPost, "/api/order/after-sales/list", bytes.NewReader(listTabBody))
	reqTabList.Header.Set("Content-Type", "application/json")
	reqTabList.Header.Set("Authorization", "Bearer "+env.adminToken)
	wTabList := httptest.NewRecorder()
	env.handler.ServeHTTP(wTabList, reqTabList)
	if wTabList.Code != http.StatusOK {
		t.Fatalf("list with tab HTTP status: %d", wTabList.Code)
	}
	var listTabResp response.Response[[]orderaftersale.AdminAfterSaleListRow]
	_ = json.Unmarshal(wTabList.Body.Bytes(), &listTabResp)
	if listTabResp.Code != 1 || len(listTabResp.Data) == 0 {
		t.Fatalf("list with tab failed: %+v", listTabResp)
	}
	for _, row := range listTabResp.Data {
		if row.Status != 10 {
			t.Fatalf("expected status 10, got %d", row.Status)
		}
	}
}

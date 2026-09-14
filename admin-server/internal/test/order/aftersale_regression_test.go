package order_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/gorm"

	miniapporder "mall/admin_core/internal/miniapp/order"
	"mall/admin_core/internal/model"
	orderaftersale "mall/admin_core/internal/order/aftersale"
	"mall/admin_core/internal/order/refund"
	productinventory "mall/admin_core/internal/product/inventory"
)

func TestAfterSaleConcurrentApplySameRequestID(t *testing.T) {
	env := setupAfterSaleTestEnv(t)
	ctx := context.Background()

	orderID, _, _ := createTestOrder(t, env.db, env.memberID, model.OrderStatusPendingShipment, 10)

	const callers = 10
	var wg sync.WaitGroup
	start := make(chan struct{})
	var successCount int32
	var asIDs sync.Map

	reqID := fmt.Sprintf("req-concurrent-%d", time.Now().UnixNano())

	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			as, err := env.asService.Apply(ctx, env.memberID, orderaftersale.MiniappApplyRequest{
				OrderID:   orderID,
				Type:      model.AfterSaleTypeRefundOnly,
				Reason:    "并发幂等申请",
				RequestID: reqID,
			})
			if err == nil && as != nil {
				atomic.AddInt32(&successCount, 1)
				asIDs.Store(as.ID, true)
			}
		}()
	}

	close(start)
	wg.Wait()

	if successCount == 0 {
		t.Fatal("expected at least one apply to succeed")
	}

	// All successes must return the exact same after-sale ID
	var distinctIDs []int64
	asIDs.Range(func(key, value any) bool {
		distinctIDs = append(distinctIDs, key.(int64))
		return true
	})
	if len(distinctIDs) != 1 {
		t.Fatalf("expected exactly 1 distinct after-sale created, got %d: %v", len(distinctIDs), distinctIDs)
	}

	var count int64
	env.db.Model(&model.AfterSale{}).Where("order_id = ?", orderID).Count(&count)
	if count != 1 {
		t.Fatalf("expected 1 record in database, got %d", count)
	}
}

func TestAfterSaleConcurrentApplyDifferentRequestID(t *testing.T) {
	env := setupAfterSaleTestEnv(t)
	ctx := context.Background()

	orderID, _, _ := createTestOrder(t, env.db, env.memberID, model.OrderStatusPendingShipment, 10)

	const callers = 8
	var wg sync.WaitGroup
	start := make(chan struct{})
	var successCount int32

	for i := 0; i < callers; i++ {
		workerIndex := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := env.asService.Apply(ctx, env.memberID, orderaftersale.MiniappApplyRequest{
				OrderID:   orderID,
				Type:      model.AfterSaleTypeRefundOnly,
				Reason:    "并发不同请求号申请",
				RequestID: fmt.Sprintf("req-diff-%d", workerIndex),
			})
			if err == nil {
				atomic.AddInt32(&successCount, 1)
			}
		}()
	}

	close(start)
	wg.Wait()

	// Only 1 concurrent application should succeed because row lock on order checks active after-sales
	if successCount != 1 {
		t.Fatalf("expected exactly 1 active after-sale created, got %d", successCount)
	}

	var count int64
	env.db.Model(&model.AfterSale{}).Where("order_id = ?", orderID).Count(&count)
	if count != 1 {
		t.Fatalf("expected 1 record in database, got %d", count)
	}
}

func TestConcurrentAdminApprove(t *testing.T) {
	env := setupAfterSaleTestEnv(t)
	ctx := context.Background()

	orderID, _, _ := createTestOrder(t, env.db, env.memberID, model.OrderStatusPendingShipment, 10)
	as, _ := env.asService.Apply(ctx, env.memberID, orderaftersale.MiniappApplyRequest{
		OrderID:   orderID,
		Type:      model.AfterSaleTypeRefundOnly,
		Reason:    "并发审核测试",
		RequestID: "req-approve-race",
	})

	const callers = 5
	var wg sync.WaitGroup
	start := make(chan struct{})
	var successCount int32

	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			err := env.asService.AdminApprove(ctx, env.admin, orderaftersale.AdminApproveRequest{ID: as.ID})
			if err == nil {
				atomic.AddInt32(&successCount, 1)
			}
		}()
	}

	close(start)
	wg.Wait()

	// Only 1 approve can succeed
	if successCount != 1 {
		t.Fatalf("expected exactly 1 approve to succeed, got %d", successCount)
	}
}

func TestConcurrentAdminConfirmReceive(t *testing.T) {
	env := setupAfterSaleTestEnv(t)
	ctx := context.Background()

	orderID, _, skuID := createTestOrder(t, env.db, env.memberID, model.OrderStatusCompleted, 10)
	as, _ := env.asService.Apply(ctx, env.memberID, orderaftersale.MiniappApplyRequest{
		OrderID:   orderID,
		Type:      model.AfterSaleTypeReturnRefund,
		Reason:    "并发收货测试",
		RequestID: "req-receive-race",
	})

	_ = env.asService.AdminApprove(ctx, env.admin, orderaftersale.AdminApproveRequest{
		ID:             as.ID,
		ReturnReceiver: "售后仓库",
		ReturnPhone:    "13800000000",
		ReturnAddress:  "杭州仓库",
	})

	_ = env.asService.SubmitReturnLogistics(ctx, env.memberID, as.ID, "顺丰速运", "SF123")

	var item model.AfterSaleItem
	_ = env.db.Where("after_sale_id = ?", as.ID).First(&item)

	const callers = 5
	var wg sync.WaitGroup
	start := make(chan struct{})
	var successCount int32

	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			err := env.asService.AdminConfirmReturnReceived(ctx, env.admin, orderaftersale.AdminConfirmReceiveRequest{
				ID: as.ID,
				Items: []orderaftersale.AdminConfirmReceiveItem{
					{AfterSaleItemID: item.ID, RestockQuantity: 1},
				},
			})
			if err == nil {
				atomic.AddInt32(&successCount, 1)
			}
		}()
	}

	close(start)
	wg.Wait()

	if successCount != 1 {
		t.Fatalf("expected exactly 1 confirm receive to succeed, got %d", successCount)
	}

	// Stock must only be restored ONCE: 10 + 1 = 11
	var inv model.ProductSkuInventory
	_ = env.db.Where("sku_id = ?", skuID).First(&inv)
	if inv.Stock != 11 {
		t.Fatalf("stock was restocked more than once! got: %d", inv.Stock)
	}
}

func TestConcurrentFinalizeRefundSuccessIdempotency(t *testing.T) {
	env := setupAfterSaleTestEnv(t)
	ctx := context.Background()

	orderID, _, skuID := createTestOrder(t, env.db, env.memberID, model.OrderStatusPendingShipment, 10)
	as, _ := env.asService.Apply(ctx, env.memberID, orderaftersale.MiniappApplyRequest{
		OrderID:   orderID,
		Type:      model.AfterSaleTypeRefundOnly,
		Reason:    "并发完成测试",
		RequestID: "req-finalize-race",
	})

	// Pre-create refund record in processing state
	now := time.Now()
	ref := model.OrderRefund{
		RefundNo:     "RF_TEST_CONC",
		AfterSaleID:  as.ID,
		OrderID:      orderID,
		MemberID:     env.memberID,
		RefundAmount: "99.00",
		Status:       model.RefundStatusProcessing,
		Provider:     "mock",
		RequestedAt:  &now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	_ = env.db.Create(&ref)
	_ = env.db.Model(&model.AfterSale{}).Where("id = ?", as.ID).Update("status", model.AfterSaleStatusRefunding)

	const callers = 10
	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := 0; i < callers; i++ {
		workerIndex := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_ = env.asService.FinalizeRefundSuccess(ctx, as.ID, fmt.Sprintf("TXN_%d", workerIndex))
		}()
	}

	close(start)
	wg.Wait()

	// Stock must only be restored ONCE: 10 + 1 = 11
	var inv model.ProductSkuInventory
	_ = env.db.Where("sku_id = ?", skuID).First(&inv)
	if inv.Stock != 11 {
		t.Fatalf("stock restored multiple times! got: %d", inv.Stock)
	}

	// Order must be refunded (60)
	var order model.OrderMain
	_ = env.db.Where("id = ?", orderID).First(&order)
	if order.Status != model.OrderStatusRefunded {
		t.Fatalf("expected order status 60, got %d", order.Status)
	}

	// Refund must be success (30)
	var orderRef model.OrderRefund
	_ = env.db.Where("after_sale_id = ?", as.ID).First(&orderRef)
	if orderRef.Status != model.RefundStatusSuccess {
		t.Fatalf("expected refund status 30, got %d", orderRef.Status)
	}
}

func TestOrderCancel_PendingPaymentVsPendingShipment(t *testing.T) {
	env := setupAfterSaleTestEnv(t)
	ctx := context.Background()

	// 1. PendingPayment (10) can be cancelled
	orderID1, _, skuID1 := createTestOrder(t, env.db, env.memberID, model.OrderStatusPendingPayment, 10)
	// Add locked stock for pending payment
	_ = env.db.Model(&model.ProductSkuInventory{}).Where("sku_id = ?", skuID1).Update("locked_stock", 1)

	orderSvc := miniapporder.NewService(env.db)
	err := orderSvc.Cancel(ctx, model.Member{ID: env.memberID}, orderID1)
	if err != nil {
		t.Fatalf("expected cancel on status 10 to succeed, got %v", err)
	}

	var o1 model.OrderMain
	if err := env.db.First(&o1, orderID1).Error; err != nil || o1.Status != model.OrderStatusCancelled {
		t.Fatalf("expected order status 50, got %d, err: %v", o1.Status, err)
	}
	var inv1 model.ProductSkuInventory
	_ = env.db.Where("sku_id = ?", skuID1).First(&inv1)
	if inv1.LockedStock != 0 {
		t.Fatalf("expected locked stock released, got %d", inv1.LockedStock)
	}

	// 2. PendingShipment (20) CANNOT be cancelled via normal cancel
	orderID2, _, _ := createTestOrder(t, env.db, env.memberID, model.OrderStatusPendingShipment, 10)
	err = orderSvc.Cancel(ctx, model.Member{ID: env.memberID}, orderID2)
	if err == nil {
		t.Fatalf("expected cancel on status 20 to fail, got nil")
	}

	var o2 model.OrderMain
	if err := env.db.First(&o2, orderID2).Error; err != nil || o2.Status != model.OrderStatusPendingShipment {
		t.Fatalf("expected order status to remain 20, got %d, err: %v", o2.Status, err)
	}
}

func TestAfterSale_RequireValidPayment(t *testing.T) {
	env := setupAfterSaleTestEnv(t)
	ctx := context.Background()

	orderID, _, _ := createTestOrder(t, env.db, env.memberID, model.OrderStatusPendingShipment, 10)

	// Remove all payments to simulate corrupted or missing payment record
	env.db.Where("order_id = ?", orderID).Delete(&model.OrderPayment{})

	as, err := env.asService.Apply(ctx, env.memberID, orderaftersale.MiniappApplyRequest{
		OrderID:   orderID,
		Type:      model.AfterSaleTypeRefundOnly,
		Reason:    "缺失支付记录测试",
		RequestID: "req-no-payment-apply",
	})
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}

	// AdminApprove must reject refund with ErrNoEligiblePayment
	err = env.asService.AdminApprove(ctx, env.admin, orderaftersale.AdminApproveRequest{
		ID: as.ID,
	})
	if !errors.Is(err, orderaftersale.ErrNoEligiblePayment) {
		t.Fatalf("expected ErrNoEligiblePayment when payment record is missing, got %v", err)
	}
}

func TestAfterSale_ProviderSuccessWithCancelledContextStillFinalizes(t *testing.T) {
	env := setupAfterSaleTestEnv(t)

	orderID, _, _ := createTestOrder(t, env.db, env.memberID, model.OrderStatusPendingShipment, 10)

	as, err := env.asService.Apply(context.Background(), env.memberID, orderaftersale.MiniappApplyRequest{
		OrderID:   orderID,
		Type:      model.AfterSaleTypeRefundOnly,
		Reason:    "上下文取消测试",
		RequestID: "req-ctx-cancel-test",
	})
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}

	// Create a context for the caller that gets cancelled once the initial transaction commits
	callerCtx, cancel := context.WithCancel(context.Background())
	go func() {
		for i := 0; i < 100; i++ {
			var check model.AfterSale
			if err := env.db.First(&check, as.ID).Error; err == nil && check.Status == model.AfterSaleStatusRefunding {
				cancel()
				return
			}
			time.Sleep(2 * time.Millisecond)
		}
		cancel()
	}()

	// AdminApprove is invoked.
	// The subsequent finalize must use background timeout context and succeed,
	// avoiding marking the refund as RefundFailed even though callerCtx was cancelled!
	_ = env.asService.AdminApprove(callerCtx, env.admin, orderaftersale.AdminApproveRequest{
		ID: as.ID,
	})

	var updatedAS model.AfterSale
	if err := env.db.First(&updatedAS, as.ID).Error; err != nil {
		t.Fatal(err)
	}

	// Must NOT be stuck in RefundFailed
	if updatedAS.Status == model.AfterSaleStatusRefundFailed {
		t.Fatalf("aftersale was wrongly marked as RefundFailed when provider succeeded!")
	}
	if updatedAS.Status != model.AfterSaleStatusCompleted {
		t.Fatalf("expected completed aftersale, got status %d", updatedAS.Status)
	}

	var orderRef model.OrderRefund
	if err := env.db.Where("after_sale_id = ?", as.ID).First(&orderRef).Error; err != nil {
		t.Fatal(err)
	}
	if orderRef.Status != model.RefundStatusSuccess {
		t.Fatalf("expected refund status success (30), got %d", orderRef.Status)
	}
}

func TestAfterSale_ReconcileRefundSuccessAndFailed(t *testing.T) {
	env := setupAfterSaleTestEnv(t)
	ctx := context.Background()

	// Case 1: Reconcile provider success
	orderID1, _, _ := createTestOrder(t, env.db, env.memberID, model.OrderStatusPendingShipment, 10)
	as1, err := env.asService.Apply(ctx, env.memberID, orderaftersale.MiniappApplyRequest{
		OrderID:   orderID1,
		Type:      model.AfterSaleTypeRefundOnly,
		Reason:    "对账成功测试",
		RequestID: "req-reconcile-succ-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	refundNo1 := "RF_RECON_SUCC_001"
	now := time.Now()
	ref1 := model.OrderRefund{
		RefundNo:     refundNo1,
		AfterSaleID:  as1.ID,
		OrderID:      orderID1,
		MemberID:     env.memberID,
		RefundAmount: "99.00",
		Status:       model.RefundStatusProcessing,
		Provider:     "mock",
		RequestedAt:  &now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	_ = env.db.Create(&ref1)
	_ = env.db.Model(&model.AfterSale{}).Where("id = ?", as1.ID).Update("status", model.AfterSaleStatusRefunding)

	// Set query override to Success
	env.refundProvider.SetQueryOverride(refundNo1, &refund.RefundQueryResult{
		Status:           refund.RefundQueryStatusSuccess,
		ProviderRefundNo: "MOCK_PROVIDER_NO_001",
	})

	err = env.asService.ReconcileRefund(ctx, as1.ID)
	if err != nil {
		t.Fatalf("ReconcileRefund failed: %v", err)
	}

	var asCheck1 model.AfterSale
	_ = env.db.First(&asCheck1, as1.ID)
	if asCheck1.Status != model.AfterSaleStatusCompleted {
		t.Fatalf("expected aftersale status completed (50), got %d", asCheck1.Status)
	}
	var refCheck1 model.OrderRefund
	_ = env.db.Where("after_sale_id = ?", as1.ID).First(&refCheck1)
	if refCheck1.Status != model.RefundStatusSuccess {
		t.Fatalf("expected refund status success (30), got %d", refCheck1.Status)
	}

	// Repeated reconcile call should be idempotent no-op
	err = env.asService.ReconcileRefund(ctx, as1.ID)
	if err != nil {
		t.Fatalf("idempotent ReconcileRefund failed: %v", err)
	}

	// Case 2: Reconcile provider failed
	orderID2, _, _ := createTestOrder(t, env.db, env.memberID, model.OrderStatusPendingShipment, 10)
	as2, err := env.asService.Apply(ctx, env.memberID, orderaftersale.MiniappApplyRequest{
		OrderID:   orderID2,
		Type:      model.AfterSaleTypeRefundOnly,
		Reason:    "对账失败测试",
		RequestID: "req-reconcile-fail-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	refundNo2 := "RF_RECON_FAIL_002"
	ref2 := model.OrderRefund{
		RefundNo:     refundNo2,
		AfterSaleID:  as2.ID,
		OrderID:      orderID2,
		MemberID:     env.memberID,
		RefundAmount: "99.00",
		Status:       model.RefundStatusProcessing,
		Provider:     "mock",
		RequestedAt:  &now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	_ = env.db.Create(&ref2)
	_ = env.db.Model(&model.AfterSale{}).Where("id = ?", as2.ID).Update("status", model.AfterSaleStatusRefunding)

	// Set query override to Failed
	env.refundProvider.SetQueryOverride(refundNo2, &refund.RefundQueryResult{
		Status:       refund.RefundQueryStatusFailed,
		ErrorMessage: "银行卡账户冻结",
	})

	err = env.asService.ReconcileRefund(ctx, as2.ID)
	if err != nil {
		t.Fatalf("ReconcileRefund with failure result should not error, got %v", err)
	}

	var asCheck2 model.AfterSale
	_ = env.db.First(&asCheck2, as2.ID)
	if asCheck2.Status != model.AfterSaleStatusRefundFailed {
		t.Fatalf("expected aftersale status refund_failed (60), got %d", asCheck2.Status)
	}
	var refCheck2 model.OrderRefund
	_ = env.db.Where("after_sale_id = ?", as2.ID).First(&refCheck2)
	if refCheck2.Status != model.RefundStatusFailed {
		t.Fatalf("expected refund status failed (40), got %d", refCheck2.Status)
	}
}

func TestAfterSale_RestockSoftDeletedProductAndSKU(t *testing.T) {
	env := setupAfterSaleTestEnv(t)
	ctx := context.Background()

	_, prodID, skuID := createTestOrder(t, env.db, env.memberID, model.OrderStatusPendingShipment, 10)

	// Soft delete the product and sku
	if err := env.db.Delete(&model.Product{}, prodID).Error; err != nil {
		t.Fatal(err)
	}
	if err := env.db.Delete(&model.ProductSKU{}, skuID).Error; err != nil {
		t.Fatal(err)
	}

	// Verify standard queries cannot find them
	if err := env.db.First(&model.Product{}, prodID).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected product to be soft deleted, got err: %v", err)
	}
	if err := env.db.First(&model.ProductSKU{}, skuID).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected SKU to be soft deleted, got err: %v", err)
	}

	// Restock via AfterSaleRestoreStockTx must still succeed due to Unscoped()
	invSvc := productinventory.NewService(env.db)
	reqID := "req-soft-del-restock-001"
	err := env.db.Transaction(func(tx *gorm.DB) error {
		return invSvc.AfterSaleRestoreStockTx(ctx, tx, prodID, skuID, 1, 100, "AS_TEST_001", reqID, 1, "admin", "售后退货入库")
	})
	if err != nil {
		t.Fatalf("AfterSaleRestoreStockTx failed on soft-deleted product/sku: %v", err)
	}

	// Check stock restored: 10 + 1 = 11
	var inv model.ProductSkuInventory
	if err := env.db.Where("sku_id = ?", skuID).First(&inv).Error; err != nil {
		t.Fatal(err)
	}
	if inv.Stock != 11 {
		t.Fatalf("expected stock 11, got %d", inv.Stock)
	}

	// Idempotency: second call with same reqID returns nil without re-incrementing stock
	err = env.db.Transaction(func(tx *gorm.DB) error {
		return invSvc.AfterSaleRestoreStockTx(ctx, tx, prodID, skuID, 1, 100, "AS_TEST_001", reqID, 1, "admin", "售后退货入库")
	})
	if err != nil {
		t.Fatalf("second AfterSaleRestoreStockTx call failed: %v", err)
	}
	_ = env.db.Where("sku_id = ?", skuID).First(&inv)
	if inv.Stock != 11 {
		t.Fatalf("stock double-restored on repeated requestID! expected 11, got %d", inv.Stock)
	}
}

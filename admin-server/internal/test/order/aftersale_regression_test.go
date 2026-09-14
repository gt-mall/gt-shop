package order_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"mall/admin_core/internal/model"
	orderaftersale "mall/admin_core/internal/order/aftersale"
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

package miniapp_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	miniapporder "mall/admin_core/internal/miniapp/order"
	miniapppayment "mall/admin_core/internal/miniapp/payment"
	"mall/admin_core/internal/model"

	"gorm.io/gorm"
)

func createPaymentRegressionOrder(t *testing.T, db *gorm.DB, memberID int64) (int64, int64) {
	t.Helper()
	_, skuID, addrID := createOrderTestData(t, db, memberID)
	order, err := miniapporder.NewService(db).Create(context.Background(), model.Member{ID: memberID}, miniapporder.OrderCreateRequest{
		RequestID: "payment-regression",
		AddressID: addrID,
		Items:     []miniapporder.CheckoutItemRequest{{SKUID: skuID, Quantity: 2}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return order.ID, skuID
}

func assertPaymentStock(t *testing.T, db *gorm.DB, skuID int64, stock, locked int) {
	t.Helper()
	var inv model.ProductSkuInventory
	if err := db.Where("sku_id = ?", skuID).First(&inv).Error; err != nil {
		t.Fatal(err)
	}
	if inv.Stock != stock || inv.LockedStock != locked {
		t.Fatalf("stock=%d locked=%d, want stock=%d locked=%d", inv.Stock, inv.LockedStock, stock, locked)
	}
}

func TestPaymentRegressionOwnershipAndIdentifiers(t *testing.T) {
	db, _, _, ownerID, _, otherID := setupOrderTestApp(t)
	orderID, skuID := createPaymentRegressionOrder(t, db, ownerID)
	svc := miniapppayment.NewService(db)
	ctx := context.Background()
	p, err := svc.CreatePayment(ctx, ownerID, miniapppayment.CreatePaymentRequest{OrderID: orderID})
	if err != nil {
		t.Fatal(err)
	}
	if p.Channel != "wechat" {
		t.Fatalf("new payment channel = %q, want wechat", p.Channel)
	}
	for _, memberID := range []int64{otherID, 0} {
		if _, err := svc.CreatePayment(ctx, memberID, miniapppayment.CreatePaymentRequest{OrderID: orderID}); !errors.Is(err, miniapppayment.ErrOrderNotBelong) {
			t.Fatalf("member %d created another member's payment: %v", memberID, err)
		}
		if _, err := svc.Query(ctx, memberID, p.PaymentNo); !errors.Is(err, miniapppayment.ErrOrderNotBelong) {
			t.Fatalf("member %d queried another member's payment: %v", memberID, err)
		}
	}
	if _, err := svc.Query(ctx, ownerID, "missing"); !errors.Is(err, miniapppayment.ErrPaymentNotFound) {
		t.Fatalf("unknown payment: %v", err)
	}
	replay, err := svc.CreatePayment(ctx, ownerID, miniapppayment.CreatePaymentRequest{OrderID: orderID})
	if err != nil || replay.ID != p.ID {
		t.Fatalf("pending payment replay failed: %v", err)
	}
	assertPaymentStock(t, db, skuID, 50, 2)
}

func TestPaymentRegressionConcurrentCreation(t *testing.T) {
	db, _, _, memberID, _, _ := setupOrderTestApp(t)
	orderID, skuID := createPaymentRegressionOrder(t, db, memberID)
	testConcurrentPaymentCreation(t, db, memberID, orderID, skuID)
}

// Shared with the opt-in MySQL test, where row locks and parallel connections are real.
func testConcurrentPaymentCreation(t *testing.T, db *gorm.DB, memberID, orderID, skuID int64) {
	t.Helper()
	svc := miniapppayment.NewService(db)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	const callers = 12
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if _, err := svc.CreatePayment(ctx, memberID, miniapppayment.CreatePaymentRequest{OrderID: orderID}); err != nil {
				t.Errorf("concurrent payment creation: %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()
	var count int64
	if err := db.Model(&model.OrderPayment{}).Where("order_id = ?", orderID).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("payment count=%d, err=%v", count, err)
	}
	assertPaymentStock(t, db, skuID, 50, 2)
}

func TestPaymentRegressionLegacyDuplicatePending(t *testing.T) {
	db, _, _, memberID, _, _ := setupOrderTestApp(t)
	orderID, skuID := createPaymentRegressionOrder(t, db, memberID)
	fixture := newWechatPaymentFixture(t, db, memberID)
	p, err := fixture.service.CreatePayment(context.Background(), memberID, miniapppayment.CreatePaymentRequest{OrderID: orderID})
	if err != nil {
		t.Fatal(err)
	}
	duplicate := *p
	duplicate.ID = 0
	duplicate.PaymentNo = "legacy-duplicate"
	if err := db.Create(&duplicate).Error; err != nil {
		t.Fatal(err)
	}
	paid, txn := fixture.pay(t, orderID)
	if paid.ID != p.ID {
		t.Fatal("duplicate pending payment was selected")
	}
	if err := db.First(&duplicate, duplicate.ID).Error; err != nil || duplicate.Status != model.PaymentStatusClosed {
		t.Fatalf("duplicate was not closed: status=%d err=%v", duplicate.Status, err)
	}
	if _, err := fixture.service.ConfirmWechat(context.Background(), txn); err != nil {
		t.Fatalf("repeated callback failed: %v", err)
	}
	assertPaymentStock(t, db, skuID, 48, 0)
}

func TestPaymentRegressionExpiryWorker(t *testing.T) {
	db, _, _, memberID, _, _ := setupOrderTestApp(t)
	orderID, skuID := createPaymentRegressionOrder(t, db, memberID)
	svc := miniapppayment.NewService(db)
	ctx := context.Background()
	if n, err := svc.CloseExpiredOrders(ctx, time.Minute); err != nil || n != 0 {
		t.Fatalf("fresh order closed: n=%d err=%v", n, err)
	}
	if err := db.Model(&model.OrderMain{}).Where("id = ?", orderID).
		Updates(map[string]any{"created_at": time.Now().Add(-time.Hour), "expire_at": time.Now().Add(-time.Minute)}).Error; err != nil {
		t.Fatal(err)
	}
	workerCtx, stop := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		svc.RunTimeoutWorker(workerCtx, time.Minute, 10*time.Millisecond)
	}()
	t.Cleanup(func() { stop(); <-done })
	deadline := time.NewTimer(3 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-deadline.C:
			t.Fatal("timeout worker did not close expired order")
		case <-ticker.C:
			var order model.OrderMain
			if err := db.First(&order, orderID).Error; err != nil {
				t.Fatal(err)
			}
			if order.Status == model.OrderStatusCancelled {
				stop()
				<-done
				assertPaymentStock(t, db, skuID, 50, 0)
				if n, err := svc.CloseExpiredOrders(ctx, time.Minute); err != nil || n != 0 {
					t.Fatalf("closed order counted again: n=%d err=%v", n, err)
				}
				return
			}
		}
	}
}

func TestPaymentConfiguredExpirySnapshot(t *testing.T) {
	db, _, _, memberID, _, _ := setupOrderTestApp(t)
	orderID, _ := createPaymentRegressionOrder(t, db, memberID)
	createdAt := time.Now().Add(-31 * time.Minute)
	expiresAt := createdAt.Add(60 * time.Minute)
	if err := db.Model(&model.OrderMain{}).Where("id = ?", orderID).
		Updates(map[string]any{"created_at": createdAt, "expire_at": expiresAt}).Error; err != nil {
		t.Fatal(err)
	}
	svc := miniapppayment.NewService(db)
	if n, err := svc.CloseExpiredOrders(context.Background(), 30*time.Minute); err != nil || n != 0 {
		t.Fatalf("60-minute order closed at 30 minutes: n=%d err=%v", n, err)
	}
	if _, err := svc.CreatePayment(context.Background(), memberID, miniapppayment.CreatePaymentRequest{OrderID: orderID}); err != nil {
		t.Fatalf("payment rejected before saved expiry: %v", err)
	}
	// This test has no WeChat merchant fixture; retire the pending prepay before sweeping.
	if err := db.Model(&model.OrderPayment{}).Where("order_id = ?", orderID).
		Update("status", model.PaymentStatusClosed).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.OrderMain{}).Where("id = ?", orderID).
		Update("expire_at", time.Now().Add(-time.Minute)).Error; err != nil {
		t.Fatal(err)
	}
	if n, err := svc.CloseExpiredOrders(context.Background(), 30*time.Minute); err != nil || n != 1 {
		t.Fatalf("expired order not closed: n=%d err=%v", n, err)
	}
}

func TestPaymentRegressionExpiryBatchesAndRetry(t *testing.T) {
	db, _, _, memberID, _, _ := setupOrderTestApp(t)
	orderID, skuID := createPaymentRegressionOrder(t, db, memberID)
	old := time.Now().Add(-time.Hour)
	if err := db.Model(&model.OrderMain{}).Where("id = ?", orderID).
		Updates(map[string]any{"created_at": old, "expire_at": old.Add(30 * time.Minute)}).Error; err != nil {
		t.Fatal(err)
	}
	var orders []model.OrderMain
	for i := 0; i < 101; i++ {
		orders = append(orders, model.OrderMain{
			OrderNo: fmt.Sprintf("batch-%d", i), MemberID: memberID, RequestID: fmt.Sprintf("batch-%d", i),
			Status: model.OrderStatusPendingPayment, CreatedAt: old,
		})
	}
	orders = append(orders,
		model.OrderMain{OrderNo: "fresh", MemberID: memberID, RequestID: "fresh", Status: model.OrderStatusPendingPayment, CreatedAt: time.Now()},
		model.OrderMain{OrderNo: "paid", MemberID: memberID, RequestID: "paid", Status: model.OrderStatusPendingShipment, CreatedAt: old},
	)
	if err := db.Create(&orders).Error; err != nil {
		t.Fatal(err)
	}
	legacy := model.OrderPayment{PaymentNo: "legacy-close-retry", OrderID: orderID, Channel: "mock", Amount: "1.00", Status: model.PaymentStatusPending}
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatal(err)
	}
	injected := errors.New("temporary payment update failure")
	failed := false
	if err := db.Callback().Update().Before("gorm:update").Register("test:fail_first_close", func(tx *gorm.DB) {
		if tx.Statement.Table == "order_payment" && !failed {
			failed = true
			tx.AddError(injected)
		}
	}); err != nil {
		t.Fatal(err)
	}
	svc := miniapppayment.NewService(db)
	n, err := svc.CloseExpiredOrders(context.Background(), 30*time.Minute)
	if n != 101 || !errors.Is(err, injected) {
		t.Fatalf("sweep must report failure and continue across batches: n=%d err=%v", n, err)
	}
	assertPaymentStock(t, db, skuID, 50, 2)
	n, err = svc.CloseExpiredOrders(context.Background(), 30*time.Minute)
	if n != 1 || err != nil {
		t.Fatalf("failed order was not retried: n=%d err=%v", n, err)
	}
	assertPaymentStock(t, db, skuID, 50, 0)
	for _, expected := range orders[101:] {
		var actual model.OrderMain
		if err := db.First(&actual, expected.ID).Error; err != nil || actual.Status != expected.Status {
			t.Fatalf("fresh/paid order changed: status=%d err=%v", actual.Status, err)
		}
	}
}

func TestPaymentExpiryDirectRejectionAndAutoClose(t *testing.T) {
	pastTime := time.Now().Add(-31 * time.Minute)

	t.Run("CreatePayment rejected on expired order and automatically closes order", func(t *testing.T) {
		db, _, _, memberID, _, _ := setupOrderTestApp(t)
		svc := miniapppayment.NewService(db)
		ctx := context.Background()

		orderID, skuID := createPaymentRegressionOrder(t, db, memberID)
		if err := db.Model(&model.OrderMain{}).Where("id = ?", orderID).
			Updates(map[string]any{"created_at": pastTime, "expire_at": pastTime.Add(30 * time.Minute)}).Error; err != nil {
			t.Fatal(err)
		}
		assertPaymentStock(t, db, skuID, 50, 2)

		_, err := svc.CreatePayment(ctx, memberID, miniapppayment.CreatePaymentRequest{OrderID: orderID})
		if !errors.Is(err, miniapppayment.ErrOrderExpired) {
			t.Fatalf("expected ErrOrderExpired, got %v", err)
		}

		var order model.OrderMain
		if err := db.First(&order, orderID).Error; err != nil || order.Status != model.OrderStatusCancelled {
			t.Fatalf("order not cancelled after expired payment attempt: status=%d, err=%v", order.Status, err)
		}
		assertPaymentStock(t, db, skuID, 50, 0)
	})

}

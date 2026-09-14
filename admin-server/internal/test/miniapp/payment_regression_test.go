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
	paid, err := svc.MockPay(ctx, ownerID, miniapppayment.MockPayRequest{OrderID: orderID})
	if err != nil {
		t.Fatal(err)
	}
	for _, memberID := range []int64{otherID, 0} {
		for _, req := range []miniapppayment.MockPayRequest{{OrderID: orderID}, {PaymentNo: paid.PaymentNo}} {
			if _, err := svc.MockPay(ctx, memberID, req); !errors.Is(err, miniapppayment.ErrOrderNotBelong) {
				t.Fatalf("member %d accessed another member's payment: %v", memberID, err)
			}
		}
		if _, err := svc.CreatePayment(ctx, memberID, miniapppayment.CreatePaymentRequest{OrderID: orderID}); !errors.Is(err, miniapppayment.ErrOrderNotBelong) {
			t.Fatalf("create payment ownership error: %v", err)
		}
	}
	if _, err := svc.MockPay(ctx, ownerID, miniapppayment.MockPayRequest{OrderID: orderID + 1, PaymentNo: paid.PaymentNo}); err == nil {
		t.Fatal("mismatched payment and order accepted")
	}
	if _, err := svc.MockPay(ctx, ownerID, miniapppayment.MockPayRequest{OrderID: orderID, PaymentNo: "missing"}); !errors.Is(err, miniapppayment.ErrPaymentNotFound) {
		t.Fatalf("unknown payment should not fall back to order ID: %v", err)
	}
	replay, err := svc.MockPay(ctx, ownerID, miniapppayment.MockPayRequest{OrderID: orderID})
	if err != nil || replay.ID != paid.ID {
		t.Fatalf("owner replay failed: %v", err)
	}
	assertPaymentStock(t, db, skuID, 48, 0)
}

func TestPaymentRegressionConcurrentCreationAndPay(t *testing.T) {
	db, _, _, memberID, _, _ := setupOrderTestApp(t)
	orderID, skuID := createPaymentRegressionOrder(t, db, memberID)
	testConcurrentPayment(t, db, memberID, orderID, skuID)
}

// Shared with the opt-in MySQL test, where row locks and parallel connections are real.
func testConcurrentPayment(t *testing.T, db *gorm.DB, memberID, orderID, skuID int64) {
	t.Helper()
	svc := miniapppayment.NewService(db)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	const callers = 12
	for _, pay := range []bool{false, true} {
		var wg sync.WaitGroup
		start := make(chan struct{})
		for i := 0; i < callers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				var err error
				if pay {
					_, err = svc.MockPay(ctx, memberID, miniapppayment.MockPayRequest{OrderID: orderID})
				} else {
					_, err = svc.CreatePayment(ctx, memberID, miniapppayment.CreatePaymentRequest{OrderID: orderID})
				}
				if err != nil {
					t.Errorf("concurrent payment (pay=%v): %v", pay, err)
				}
			}()
		}
		close(start)
		wg.Wait()
		var count int64
		if err := db.Model(&model.OrderPayment{}).Where("order_id = ?", orderID).Count(&count).Error; err != nil || count != 1 {
			t.Fatalf("payment count=%d, err=%v", count, err)
		}
	}
	assertPaymentStock(t, db, skuID, 48, 0)
	var count int64
	if err := db.Model(&model.OrderLog{}).Where("order_id = ? AND action = ?", orderID, "pay").Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("pay log count=%d, err=%v", count, err)
	}
}

func TestPaymentRegressionLegacyDuplicatePending(t *testing.T) {
	db, _, _, memberID, _, _ := setupOrderTestApp(t)
	orderID, skuID := createPaymentRegressionOrder(t, db, memberID)
	svc := miniapppayment.NewService(db)
	ctx := context.Background()
	p, err := svc.CreatePayment(ctx, memberID, miniapppayment.CreatePaymentRequest{OrderID: orderID})
	if err != nil {
		t.Fatal(err)
	}
	duplicate := *p
	duplicate.ID = 0
	duplicate.PaymentNo = "legacy-duplicate"
	if err := db.Create(&duplicate).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.MockPay(ctx, memberID, miniapppayment.MockPayRequest{PaymentNo: p.PaymentNo}); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&duplicate, duplicate.ID).Error; err != nil || duplicate.Status != model.PaymentStatusClosed {
		t.Fatalf("duplicate was not closed: status=%d err=%v", duplicate.Status, err)
	}
	if _, err := svc.MockPay(ctx, memberID, miniapppayment.MockPayRequest{PaymentNo: duplicate.PaymentNo}); err == nil {
		t.Fatal("duplicate payment marked successful")
	}
	replay, err := svc.MockPay(ctx, memberID, miniapppayment.MockPayRequest{OrderID: orderID})
	if err != nil || replay.ID != p.ID {
		t.Fatalf("replay must return original successful payment: %v", err)
	}
	assertPaymentStock(t, db, skuID, 48, 0)
}

func TestPaymentRegressionCancelAndPay(t *testing.T) {
	db, _, _, memberID, _, _ := setupOrderTestApp(t)
	orderID, skuID := createPaymentRegressionOrder(t, db, memberID)
	testCancelAndPay(t, db, memberID, orderID, skuID)
}

func testCancelAndPay(t *testing.T, db *gorm.DB, memberID, orderID, skuID int64) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	paySvc := miniapppayment.NewService(db)
	p, err := paySvc.CreatePayment(ctx, memberID, miniapppayment.CreatePaymentRequest{OrderID: orderID})
	if err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	payResult := make(chan error, 1)
	go func() {
		<-start
		_, err := paySvc.MockPay(ctx, memberID, miniapppayment.MockPayRequest{PaymentNo: p.PaymentNo})
		payResult <- err
	}()
	close(start)
	cancelErr := miniapporder.NewService(db).Cancel(ctx, model.Member{ID: memberID}, orderID)
	payErr := <-payResult

	var order model.OrderMain
	if err := db.First(&order, orderID).Error; err != nil {
		t.Fatal(err)
	}

	if cancelErr == nil {
		// Cancel won the race
		if payErr != nil && !errors.Is(payErr, miniapppayment.ErrOrderCancelled) {
			t.Fatalf("payment/cancel race unexpected pay error: %v", payErr)
		}
		if order.Status != model.OrderStatusCancelled {
			t.Fatalf("expected cancelled order, got %d", order.Status)
		}
		assertPaymentStock(t, db, skuID, 50, 0)
	} else {
		// Pay won the race, Cancel failed because order is now paid (status 20 is non-cancellable)
		if payErr != nil {
			t.Fatalf("expected pay success when cancel failed: payErr=%v, cancelErr=%v", payErr, cancelErr)
		}
		if order.Status != model.OrderStatusPendingShipment {
			t.Fatalf("expected pending shipment order, got %d", order.Status)
		}
		assertPaymentStock(t, db, skuID, 48, 0)
	}
}

func TestPaymentRegressionCloseFailureRollsBack(t *testing.T) {
	for _, timeout := range []bool{false, true} {
		t.Run(map[bool]string{false: "cancel", true: "timeout"}[timeout], func(t *testing.T) {
			db, _, _, memberID, _, _ := setupOrderTestApp(t)
			orderID, skuID := createPaymentRegressionOrder(t, db, memberID)
			svc := miniapppayment.NewService(db)
			ctx := context.Background()
			if _, err := svc.CreatePayment(ctx, memberID, miniapppayment.CreatePaymentRequest{OrderID: orderID}); err != nil {
				t.Fatal(err)
			}
			injected := errors.New("payment update failed")
			if err := db.Callback().Update().Before("gorm:update").Register("test:fail_payment_update", func(tx *gorm.DB) {
				if tx.Statement.Table == "order_payment" {
					tx.AddError(injected)
				}
			}); err != nil {
				t.Fatal(err)
			}
			var err error
			if timeout {
				err = svc.CloseTimeoutOrder(ctx, orderID)
			} else {
				err = miniapporder.NewService(db).Cancel(ctx, model.Member{ID: memberID}, orderID)
			}
			if !errors.Is(err, injected) {
				t.Fatalf("payment update error swallowed: %v", err)
			}
			assertPaymentStock(t, db, skuID, 50, 2)
			var order model.OrderMain
			if err := db.First(&order, orderID).Error; err != nil || order.Status != model.OrderStatusPendingPayment {
				t.Fatalf("order not rolled back: status=%d err=%v", order.Status, err)
			}
		})
	}
}

func TestPaymentRegressionExpiryWorker(t *testing.T) {
	db, _, _, memberID, _, _ := setupOrderTestApp(t)
	orderID, skuID := createPaymentRegressionOrder(t, db, memberID)
	svc := miniapppayment.NewService(db)
	ctx := context.Background()
	if _, err := svc.CreatePayment(ctx, memberID, miniapppayment.CreatePaymentRequest{OrderID: orderID}); err != nil {
		t.Fatal(err)
	}
	if n, err := svc.CloseExpiredOrders(ctx, time.Minute); err != nil || n != 0 {
		t.Fatalf("fresh order closed: n=%d err=%v", n, err)
	}
	if err := db.Model(&model.OrderMain{}).Where("id = ?", orderID).Update("created_at", time.Now().Add(-time.Hour)).Error; err != nil {
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

func TestPaymentRegressionExpiryBatchesAndRetry(t *testing.T) {
	db, _, _, memberID, _, _ := setupOrderTestApp(t)
	orderID, skuID := createPaymentRegressionOrder(t, db, memberID)
	old := time.Now().Add(-time.Hour)
	if err := db.Model(&model.OrderMain{}).Where("id = ?", orderID).Update("created_at", old).Error; err != nil {
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

func TestPaymentRegressionTimeoutAndPay(t *testing.T) {
	db, _, _, memberID, _, _ := setupOrderTestApp(t)
	orderID, skuID := createPaymentRegressionOrder(t, db, memberID)
	testTimeoutAndPay(t, db, memberID, orderID, skuID)
}

func testTimeoutAndPay(t *testing.T, db *gorm.DB, memberID, orderID, skuID int64) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	svc := miniapppayment.NewService(db)
	if _, err := svc.CreatePayment(ctx, memberID, miniapppayment.CreatePaymentRequest{OrderID: orderID}); err != nil {
		t.Fatal(err)
	}
	payResult := make(chan error, 1)
	go func() {
		_, err := svc.MockPay(ctx, memberID, miniapppayment.MockPayRequest{OrderID: orderID})
		payResult <- err
	}()
	if err := svc.CloseTimeoutOrder(ctx, orderID); err != nil {
		t.Fatal(err)
	}
	payErr := <-payResult
	var order model.OrderMain
	if err := db.First(&order, orderID).Error; err != nil {
		t.Fatal(err)
	}
	if payErr == nil {
		if order.Status != model.OrderStatusPendingShipment {
			t.Fatalf("paid order was closed: status=%d", order.Status)
		}
		assertPaymentStock(t, db, skuID, 48, 0)
	} else {
		if !errors.Is(payErr, miniapppayment.ErrOrderCancelled) || order.Status != model.OrderStatusCancelled {
			t.Fatalf("timeout/payment race: status=%d err=%v", order.Status, payErr)
		}
		assertPaymentStock(t, db, skuID, 50, 0)
	}
}

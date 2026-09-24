package miniapp_test

import (
	"context"
	"testing"
	"time"

	miniapporder "mall/admin_core/internal/miniapp/order"
	miniapppayment "mall/admin_core/internal/miniapp/payment"
	"mall/admin_core/internal/model"
	orderaftersale "mall/admin_core/internal/order/aftersale"
	systemsetting "mall/admin_core/internal/system/setting"
)

func TestAutoReceiveRespectsActiveAfterSaleAndZeroDeadline(t *testing.T) {
	db, _, _, memberID, _, _ := setupOrderTestApp(t)
	orderID, _ := createPaymentRegressionOrder(t, db, memberID)
	ctx := context.Background()
	cfg := systemsetting.DefaultTradeConfig()
	cfg.AfterSaleDays = 0
	if err := systemsetting.NewService(db).SaveTrade(ctx, cfg); err != nil {
		t.Fatal(err)
	}

	shippedAt := time.Now().Add(-8 * 24 * time.Hour)
	autoReceiveAt := time.Now().Add(-time.Minute)
	if err := db.Model(&model.OrderMain{}).Where("id = ?", orderID).
		Updates(map[string]any{
			"status":          model.OrderStatusPendingReceipt,
			"paid_at":         shippedAt,
			"shipped_at":      shippedAt,
			"auto_receive_at": autoReceiveAt,
		}).Error; err != nil {
		t.Fatal(err)
	}
	active := model.AfterSale{
		AfterSaleNo: "AS-ACTIVE-RECEIVE", RequestID: "active-receive",
		OrderID: orderID, MemberID: memberID, Type: model.AfterSaleTypeRefundOnly,
		Status: model.AfterSaleStatusPendingReview, OrderStatusAtApply: model.OrderStatusPendingReceipt,
		Reason: "待处理退款", RefundAmount: "399.80", AppliedAt: time.Now(),
	}
	if err := db.Create(&active).Error; err != nil {
		t.Fatal(err)
	}

	svc := miniapppayment.NewService(db)
	if n, err := svc.AutoReceiveShippedOrders(ctx); err != nil || n != 0 {
		t.Fatalf("active after-sale order auto received: n=%d err=%v", n, err)
	}
	var order model.OrderMain
	if err := db.First(&order, orderID).Error; err != nil || order.Status != model.OrderStatusPendingReceipt {
		t.Fatalf("order status changed with active after-sale: status=%d err=%v", order.Status, err)
	}

	if err := db.Model(&active).Update("status", model.AfterSaleStatusCancelled).Error; err != nil {
		t.Fatal(err)
	}
	if n, err := svc.AutoReceiveShippedOrders(ctx); err != nil || n != 1 {
		t.Fatalf("order was not auto received: n=%d err=%v", n, err)
	}
	if err := db.First(&order, orderID).Error; err != nil ||
		order.Status != model.OrderStatusCompleted || order.AfterSaleDeadline == nil {
		t.Fatalf("completed order missing deadline: status=%d deadline=%v err=%v",
			order.Status, order.AfterSaleDeadline, err)
	}
	if _, err := orderaftersale.NewService(db, nil, nil).Apply(ctx, memberID, orderaftersale.MiniappApplyRequest{
		OrderID: orderID, Type: model.AfterSaleTypeReturnRefund,
		Reason: "已过零天售后期限", RequestID: "zero-deadline-apply",
	}); err == nil {
		t.Fatal("after-sale application accepted although configured deadline is zero days")
	}

}

func TestManualConfirmZeroAfterSaleDeadline(t *testing.T) {
	db, _, _, memberID, _, _ := setupOrderTestApp(t)
	orderID, _ := createPaymentRegressionOrder(t, db, memberID)
	ctx := context.Background()
	cfg := systemsetting.DefaultTradeConfig()
	cfg.AfterSaleDays = 0
	if err := systemsetting.NewService(db).SaveTrade(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := db.Model(&model.OrderMain{}).Where("id = ?", orderID).
		Updates(map[string]any{
			"status":     model.OrderStatusPendingReceipt,
			"paid_at":    now,
			"shipped_at": now,
		}).Error; err != nil {
		t.Fatal(err)
	}
	if err := miniapporder.NewService(db).Confirm(ctx, model.Member{ID: memberID}, orderID); err != nil {
		t.Fatal(err)
	}
	var order model.OrderMain
	if err := db.First(&order, orderID).Error; err != nil ||
		order.AfterSaleDeadline == nil || order.CompletedAt == nil ||
		order.AfterSaleDeadline.After(order.CompletedAt.Add(time.Second)) {
		t.Fatalf("manual receipt did not preserve zero-day deadline: deadline=%v completed=%v err=%v",
			order.AfterSaleDeadline, order.CompletedAt, err)
	}
	if _, err := orderaftersale.NewService(db, nil, nil).Apply(ctx, memberID, orderaftersale.MiniappApplyRequest{
		OrderID: orderID, Type: model.AfterSaleTypeReturnRefund,
		Reason: "已过零天售后期限", RequestID: "manual-zero-deadline-apply",
	}); err == nil {
		t.Fatal("after-sale application accepted after zero-day manual confirmation")
	}
}

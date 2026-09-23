package order_test

import (
	"context"
	"testing"

	"mall/admin_core/internal/model"
	"mall/admin_core/internal/order/aftersale"
	"mall/admin_core/internal/order/refund"
	"mall/admin_core/internal/order/wechatpay"
	"mall/admin_core/pkg/util"
)

type acceptedRefundProvider struct {
	status  refund.RefundQueryStatus
	request refund.RefundRequest
}

func (p *acceptedRefundProvider) Refund(_ context.Context, req refund.RefundRequest) (*refund.RefundResult, error) {
	p.request = req
	return &refund.RefundResult{Accepted: true, ProviderRefundNo: "wx-refund-1"}, nil
}
func (p *acceptedRefundProvider) QueryRefund(context.Context, string, string) (*refund.RefundQueryResult, error) {
	return &refund.RefundQueryResult{Status: p.status, ProviderRefundNo: "wx-refund-1"}, nil
}

func TestAcceptedWechatRefundWaitsForConfirmedSuccess(t *testing.T) {
	env := setupAfterSaleTestEnv(t)
	provider := &acceptedRefundProvider{status: refund.RefundQueryStatusProcessing}
	svc := aftersale.NewService(env.db, env.bizNoGen, provider)
	orderID, _, skuID := createTestOrder(t, env.db, env.memberID, model.OrderStatusPendingShipment, 10)
	as, err := svc.Apply(context.Background(), env.memberID, aftersale.MiniappApplyRequest{OrderID: orderID, Type: model.AfterSaleTypeRefundOnly, Reason: "不需要了", RequestID: "wechat-refund-accepted"})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.AdminApprove(context.Background(), env.admin, aftersale.AdminApproveRequest{ID: as.ID}); err != nil {
		t.Fatal(err)
	}
	totalCents, parseErr := util.ParseDecimalYuanToCent(provider.request.TotalAmount)
	if parseErr != nil || totalCents != 9900 {
		t.Fatalf("wrong original payment amount %s: %v", provider.request.TotalAmount, parseErr)
	}
	var saved model.AfterSale
	if err := env.db.First(&saved, as.ID).Error; err != nil || saved.Status != model.AfterSaleStatusRefunding {
		t.Fatalf("accepted refund was finalized: %v, status %d", err, saved.Status)
	}
	var inv model.ProductSkuInventory
	if err := env.db.Where("sku_id = ?", skuID).First(&inv).Error; err != nil || inv.Stock != 10 {
		t.Fatalf("stock changed before refund success: %v, %d", err, inv.Stock)
	}
	if err := svc.ReconcileRefund(context.Background(), as.ID); err != nil {
		t.Fatal(err)
	}
	if err := env.db.First(&saved, as.ID).Error; err != nil || saved.Status != model.AfterSaleStatusRefunding {
		t.Fatal("processing refund was finalized")
	}
	provider.status = refund.RefundQueryStatusSuccess
	if err := svc.ReconcileRefund(context.Background(), as.ID); err != nil {
		t.Fatal(err)
	}
	if err := env.db.First(&saved, as.ID).Error; err != nil || saved.Status != model.AfterSaleStatusCompleted {
		t.Fatalf("confirmed refund not finalized: %v, %d", err, saved.Status)
	}
}

func TestLegacyMockOrderCannotEnterWechatRefund(t *testing.T) {
	env := setupAfterSaleTestEnv(t)
	svc := aftersale.NewService(env.db, env.bizNoGen, wechatpay.RefundProvider{Store: wechatpay.Store{DB: env.db}})
	orderID, _, _ := createTestOrder(t, env.db, env.memberID, model.OrderStatusPendingShipment, 10)
	as, err := svc.Apply(context.Background(), env.memberID, aftersale.MiniappApplyRequest{OrderID: orderID, Type: model.AfterSaleTypeRefundOnly, Reason: "历史订单", RequestID: "legacy-mock-refund"})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.AdminApprove(context.Background(), env.admin, aftersale.AdminApproveRequest{ID: as.ID}); err == nil {
		t.Fatal("mock payment entered real WeChat refund")
	}
	var saved model.AfterSale
	if err := env.db.First(&saved, as.ID).Error; err != nil || saved.Status != model.AfterSaleStatusPendingReview {
		t.Fatalf("approval changed status after rejection: %v, %d", err, saved.Status)
	}
}

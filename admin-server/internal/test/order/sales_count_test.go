package order_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"mall/admin_core/internal/model"
	orderaftersale "mall/admin_core/internal/order/aftersale"
)

func createUnpaidTestOrder(t *testing.T, env *testAfterSaleEnv, prodID, skuID int64, qty int) int64 {
	t.Helper()
	now := time.Now()

	// Ensure inventory has locked_stock for this order to be deducted upon payment
	var inv model.ProductSkuInventory
	if err := env.db.Where("sku_id = ?", skuID).First(&inv).Error; err == nil {
		env.db.Model(&inv).Update("locked_stock", inv.LockedStock+qty)
	}

	order := model.OrderMain{
		OrderNo:     fmt.Sprintf("ORD_PAY_%d", time.Now().UnixNano()),
		MemberID:    env.memberID,
		RequestID:   fmt.Sprintf("req_pay_%d", time.Now().UnixNano()),
		Status:      model.OrderStatusPendingPayment,
		TotalAmount: fmt.Sprintf("%d0.00", qty),
		PayAmount:   fmt.Sprintf("%d0.00", qty),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := env.db.Create(&order).Error; err != nil {
		t.Fatalf("create unpaid order: %v", err)
	}

	specs := `[{"spec_id":1,"spec_name":"颜色","value_id":101,"value_name":"黑色"}]`
	item := model.OrderItem{
		OrderID:       order.ID,
		ProductID:     prodID,
		SKUID:         skuID,
		ProductName:   "销量商品",
		SKUCode:       fmt.Sprintf("SKU_PAY_%d", skuID),
		SKUSpecValues: &specs,
		UnitPrice:     "10.00",
		Quantity:      qty,
		TotalAmount:   order.PayAmount,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := env.db.Create(&item).Error; err != nil {
		t.Fatalf("create unpaid order item: %v", err)
	}

	return order.ID
}

func TestSalesCount_Lifecycle(t *testing.T) {
	env := setupAfterSaleTestEnv(t)
	ctx := context.Background()
	payFixture := newOrderWechatPaymentFixture(t, env)

	now := time.Now()
	prod := model.Product{
		Name:       "销量联动测试商品",
		Status:     1,
		SalesCount: 0,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := env.db.Create(&prod).Error; err != nil {
		t.Fatal(err)
	}

	skuCode := "SKU_SALES_001"
	sku := model.ProductSKU{
		ProductID:   prod.ID,
		SKUCode:     &skuCode,
		SpecValues:  "[]",
		Price:       "10.00",
		CostPrice:   "5.00",
		MarketPrice: "20.00",
		Status:      1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := env.db.Create(&sku).Error; err != nil {
		t.Fatal(err)
	}

	inv := model.ProductSkuInventory{
		ProductID: prod.ID,
		SKUID:     sku.ID,
		Stock:     100,
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := env.db.Create(&inv).Error; err != nil {
		t.Fatal(err)
	}

	// Case 1: Initial sales_count = 0. Pay order with quantity = 2 -> sales_count = 2
	orderID1 := createUnpaidTestOrder(t, env, prod.ID, sku.ID, 2)
	payRes, firstTxn := payFixture.pay(t, orderID1)
	if payRes.Status != model.PaymentStatusSuccess {
		t.Fatalf("expected payment success, got %d", payRes.Status)
	}

	var prodCheck model.Product
	_ = env.db.First(&prodCheck, prod.ID)
	if prodCheck.SalesCount != 2 {
		t.Fatalf("Case 1: expected sales_count = 2, got %d", prodCheck.SalesCount)
	}

	// Case 2: Duplicate payment on order 1 -> still = 2 (idempotent)
	_, err := payFixture.service.ConfirmWechat(ctx, firstTxn)
	if err != nil {
		t.Fatalf("duplicate WeChat callback failed: %v", err)
	}
	_ = env.db.First(&prodCheck, prod.ID)
	if prodCheck.SalesCount != 2 {
		t.Fatalf("Case 2: duplicate payment changed sales_count! expected 2, got %d", prodCheck.SalesCount)
	}

	// Case 3: Second order with quantity = 3 -> sales_count becomes 2 + 3 = 5
	orderID2 := createUnpaidTestOrder(t, env, prod.ID, sku.ID, 3)
	payFixture.pay(t, orderID2)
	_ = env.db.First(&prodCheck, prod.ID)
	if prodCheck.SalesCount != 5 {
		t.Fatalf("Case 3: expected sales_count = 5, got %d", prodCheck.SalesCount)
	}

	// Case 4: Full refund on order 1 (quantity 2) -> sales_count becomes 5 - 2 = 3
	as1, err := env.asService.Apply(ctx, env.memberID, orderaftersale.MiniappApplyRequest{
		OrderID:   orderID1,
		Type:      model.AfterSaleTypeRefundOnly,
		Reason:    "退款回退销量测试",
		RequestID: fmt.Sprintf("req_as_sales_%d", time.Now().UnixNano()),
	})
	if err != nil {
		t.Fatalf("apply aftersale on order 1 failed: %v", err)
	}

	// Approve will trigger refund and FinalizeRefundSuccess for pending shipment
	err = env.asService.AdminApprove(ctx, env.admin, orderaftersale.AdminApproveRequest{ID: as1.ID})
	if err != nil {
		t.Fatalf("approve aftersale failed: %v", err)
	}

	_ = env.db.First(&prodCheck, prod.ID)
	if prodCheck.SalesCount != 3 {
		t.Fatalf("Case 4: expected sales_count = 3 after refunding 2 items, got %d", prodCheck.SalesCount)
	}

	// Case 5: Duplicate FinalizeRefundSuccess on as1 -> still = 3 (idempotent)
	err = env.asService.FinalizeRefundSuccess(ctx, as1.ID, "MOCK_DUP_REF_NO")
	if err != nil {
		t.Fatalf("duplicate FinalizeRefundSuccess failed: %v", err)
	}
	_ = env.db.First(&prodCheck, prod.ID)
	if prodCheck.SalesCount != 3 {
		t.Fatalf("Case 5: duplicate finalize changed sales_count! expected 3, got %d", prodCheck.SalesCount)
	}

	// Case 6: Refund quantity larger than current sales_count -> capped at 0, never negative
	// Currently sales_count = 3. We refund order 2 which has quantity = 3 -> sales_count = 0.
	as2, err := env.asService.Apply(ctx, env.memberID, orderaftersale.MiniappApplyRequest{
		OrderID:   orderID2,
		Type:      model.AfterSaleTypeRefundOnly,
		Reason:    "退款归零测试",
		RequestID: fmt.Sprintf("req_as_sales_%d", time.Now().UnixNano()),
	})
	if err != nil {
		t.Fatalf("apply aftersale on order 2 failed: %v", err)
	}
	err = env.asService.AdminApprove(ctx, env.admin, orderaftersale.AdminApproveRequest{ID: as2.ID})
	if err != nil {
		t.Fatalf("approve aftersale on order 2 failed: %v", err)
	}
	_ = env.db.First(&prodCheck, prod.ID)
	if prodCheck.SalesCount != 0 {
		t.Fatalf("Case 6: expected sales_count = 0, got %d", prodCheck.SalesCount)
	}

	// Now try another finalize directly simulating excess refund -> remains 0, never negative
	// Modify as2 status temporarily to test FinalizeRefundSuccess directly with sales_count at 0
	_ = env.db.Model(&model.AfterSale{}).Where("id = ?", as2.ID).Update("status", model.AfterSaleStatusRefunding)
	_ = env.db.Model(&model.OrderRefund{}).Where("after_sale_id = ?", as2.ID).Update("status", model.RefundStatusProcessing)
	err = env.asService.FinalizeRefundSuccess(ctx, as2.ID, "MOCK_EXCESS_REF_NO")
	if err != nil {
		t.Fatalf("excess finalize failed: %v", err)
	}
	_ = env.db.First(&prodCheck, prod.ID)
	if prodCheck.SalesCount < 0 {
		t.Fatalf("Case 6: sales_count became negative: %d", prodCheck.SalesCount)
	}
	if prodCheck.SalesCount != 0 {
		t.Fatalf("Case 6: expected sales_count = 0, got %d", prodCheck.SalesCount)
	}
}

func TestSalesCount_SoftDeletedAndPartialRestock(t *testing.T) {
	env := setupAfterSaleTestEnv(t)
	ctx := context.Background()
	payFixture := newOrderWechatPaymentFixture(t, env)

	now := time.Now()
	prod := model.Product{
		Name:       "软删除与部分入库测试商品",
		Status:     1,
		SalesCount: 0,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := env.db.Create(&prod).Error; err != nil {
		t.Fatal(err)
	}

	skuCode := "SKU_SALES_002"
	sku := model.ProductSKU{
		ProductID:   prod.ID,
		SKUCode:     &skuCode,
		SpecValues:  "[]",
		Price:       "10.00",
		CostPrice:   "5.00",
		MarketPrice: "20.00",
		Status:      1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := env.db.Create(&sku).Error; err != nil {
		t.Fatal(err)
	}

	inv := model.ProductSkuInventory{
		ProductID: prod.ID,
		SKUID:     sku.ID,
		Stock:     50,
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := env.db.Create(&inv).Error; err != nil {
		t.Fatal(err)
	}

	// 1. Create order for 2 items and pay -> sales_count = 2
	orderID := createUnpaidTestOrder(t, env, prod.ID, sku.ID, 2)
	payFixture.pay(t, orderID)

	var pCheck model.Product
	_ = env.db.First(&pCheck, prod.ID)
	if pCheck.SalesCount != 2 {
		t.Fatalf("expected sales_count 2, got %d", pCheck.SalesCount)
	}

	// 2. Soft-delete the product
	if err := env.db.Delete(&model.Product{}, prod.ID).Error; err != nil {
		t.Fatal(err)
	}

	// 3. Apply after-sale on the soft-deleted product
	as, err := env.asService.Apply(ctx, env.memberID, orderaftersale.MiniappApplyRequest{
		OrderID:   orderID,
		Type:      model.AfterSaleTypeRefundOnly,
		Reason:    "已删除商品退款",
		RequestID: fmt.Sprintf("req_as_del_%d", time.Now().UnixNano()),
	})
	if err != nil {
		t.Fatalf("apply aftersale on soft-deleted product failed: %v", err)
	}

	// 4. Approve and finalize refund
	err = env.asService.AdminApprove(ctx, env.admin, orderaftersale.AdminApproveRequest{ID: as.ID})
	if err != nil {
		t.Fatalf("approve aftersale failed: %v", err)
	}

	// Check soft-deleted product sales_count rolled back: 2 -> 0
	var pDelCheck model.Product
	if err := env.db.Unscoped().First(&pDelCheck, prod.ID).Error; err != nil {
		t.Fatalf("find unscoped product: %v", err)
	}
	if pDelCheck.SalesCount != 0 {
		t.Fatalf("expected sales_count rolled back to 0 on soft-deleted product, got %d", pDelCheck.SalesCount)
	}
	// Verify deleted_at is STILL not null (not erroneously restored)
	if !pDelCheck.DeletedAt.Valid {
		t.Fatal("soft-deleted product should remain deleted after sales_count rollback")
	}

	// 5. Test Return & Refund: restock_quantity = 1, refund quantity = 2
	// sales_count must decrement by 2, while inventory stock must only increment by 1
	prod2 := model.Product{Name: "退货入库分离商品", Status: 1, SalesCount: 0, CreatedAt: now, UpdatedAt: now}
	env.db.Create(&prod2)
	skuCode2 := "SKU_SALES_003"
	sku2 := model.ProductSKU{ProductID: prod2.ID, SKUCode: &skuCode2, Price: "10.00", Status: 1, CreatedAt: now, UpdatedAt: now}
	env.db.Create(&sku2)
	inv2 := model.ProductSkuInventory{ProductID: prod2.ID, SKUID: sku2.ID, Stock: 50, Version: 1, CreatedAt: now, UpdatedAt: now}
	env.db.Create(&inv2)

	orderID2 := createUnpaidTestOrder(t, env, prod2.ID, sku2.ID, 2)
	payFixture.pay(t, orderID2)

	// Move order to PendingReceipt (shipped)
	shippedTime := time.Now()
	env.db.Model(&model.OrderMain{}).Where("id = ?", orderID2).Updates(map[string]any{
		"status":     model.OrderStatusPendingReceipt,
		"shipped_at": &shippedTime,
	})

	// Apply return refund
	asReturn, err := env.asService.Apply(ctx, env.memberID, orderaftersale.MiniappApplyRequest{
		OrderID:   orderID2,
		Type:      model.AfterSaleTypeReturnRefund,
		Reason:    "部分损坏退货",
		RequestID: fmt.Sprintf("req_as_ret_%d", time.Now().UnixNano()),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Merchant approves with return address
	err = env.asService.AdminApprove(ctx, env.admin, orderaftersale.AdminApproveRequest{
		ID:             asReturn.ID,
		ReturnReceiver: "售后仓库",
		ReturnPhone:    "13800000000",
		ReturnAddress:  "上海市浦东新区张江高科",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Buyer fills logistics
	err = env.asService.SubmitReturnLogistics(ctx, env.memberID, asReturn.ID, "顺丰速运", "SF1234567890")
	if err != nil {
		t.Fatal(err)
	}

	// Merchant confirms receipt: out of 2 items returned, only 1 is restocked (1 is damaged)
	var asItem model.AfterSaleItem
	env.db.Where("after_sale_id = ?", asReturn.ID).First(&asItem)

	err = env.asService.AdminConfirmReturnReceived(ctx, env.admin, orderaftersale.AdminConfirmReceiveRequest{
		ID: asReturn.ID,
		Items: []orderaftersale.AdminConfirmReceiveItem{
			{AfterSaleItemID: asItem.ID, RestockQuantity: 1}, // only 1 restocked!
		},
	})
	if err != nil {
		t.Fatalf("receive and restock failed: %v", err)
	}

	// Verify inventory stock: was 50 - 2 (locked deduct) = 48 + 1 (restocked) = 49
	var invCheck model.ProductSkuInventory
	env.db.Where("sku_id = ?", sku2.ID).First(&invCheck)
	if invCheck.Stock != 49 {
		t.Fatalf("expected inventory stock 49 (only 1 restocked), got %d", invCheck.Stock)
	}

	// Verify sales_count: was 2 - 2 (refunded) = 0!
	// It must decrement by full refund quantity 2, NOT by restock_quantity 1!
	var prod2Check model.Product
	env.db.First(&prod2Check, prod2.ID)
	if prod2Check.SalesCount != 0 {
		t.Fatalf("expected sales_count = 0 (full refund quantity 2 subtracted), got %d", prod2Check.SalesCount)
	}
}

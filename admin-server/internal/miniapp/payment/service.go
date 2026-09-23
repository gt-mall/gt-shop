package payment

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"mall/admin_core/internal/model"
	productinventory "mall/admin_core/internal/product/inventory"
	"mall/admin_core/pkg/bizno"
	"mall/admin_core/pkg/cache"
	"mall/admin_core/pkg/ordertimeout"
)

type Service struct {
	db       *gorm.DB
	bizNoGen *bizno.Generator
	cache    cache.Cache
}

func (s *Service) WithCache(c cache.Cache) *Service {
	s.cache = c
	return s
}

func (s *Service) invalidateProductSalesCache(productIDs []int64) {
	if s.cache == nil || s.cache.IsDisabled() {
		return
	}
	bgCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	for _, pid := range productIDs {
		if _, err := cache.BumpProductDetailGeneration(bgCtx, s.cache, pid); err != nil {
			log.Printf("[Cache] product sales_count cache invalidation failed product_id=%d: %v", pid, err)
		}
	}
	if _, err := cache.BumpProductListGeneration(bgCtx, s.cache); err != nil {
		log.Printf("[Cache] product list sales_count cache invalidation failed: %v", err)
	}
}

func NewService(db *gorm.DB, bizNoGen ...*bizno.Generator) *Service {
	var gen *bizno.Generator
	if len(bizNoGen) > 0 && bizNoGen[0] != nil {
		gen = bizNoGen[0]
	} else {
		var err error
		gen, err = bizno.NewGenerator(0)
		if err != nil {
			panic(fmt.Sprintf("failed to initialize default bizno generator in payment: %v", err))
		}
	}
	return &Service{
		db:       db,
		bizNoGen: gen,
	}
}

// CreatePayment creates a new payment record for an order or returns an existing pending payment.
func (s *Service) CreatePayment(ctx context.Context, memberID int64, req CreatePaymentRequest) (*model.OrderPayment, error) {
	if s.db == nil {
		return nil, errors.New("数据库服务不可用")
	}

	if memberID <= 0 {
		return nil, ErrOrderNotBelong
	}

	var payment *model.OrderPayment
	var isTimeout bool
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order model.OrderMain
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND deleted_at IS NULL", req.OrderID).First(&order).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrOrderNotFound
			}
			return err
		}
		if order.MemberID != memberID {
			return ErrOrderNotBelong
		}
		if err := pendingOrderError(order.Status); err != nil {
			return err
		}
		now := time.Now()
		if ordertimeout.IsExpired(order.CreatedAt, now) {
			if err := s.closeTimeoutOrderTx(ctx, tx, &order, now); err != nil {
				return err
			}
			isTimeout = true
			return nil
		}
		var err error
		payment, err = s.pendingPaymentTx(tx, order, req.Channel)
		return err
	})
	if err != nil {
		return nil, err
	}
	if isTimeout {
		return nil, ErrOrderExpired
	}
	return payment, nil
}

func pendingOrderError(status int8) error {
	switch status {
	case model.OrderStatusPendingPayment:
		return nil
	case model.OrderStatusCancelled:
		return ErrOrderCancelled
	case model.OrderStatusPendingShipment, model.OrderStatusPendingReceipt, model.OrderStatusCompleted:
		return ErrOrderAlreadyPaid
	default:
		return ErrOrderInvalidStatus
	}
}

// The caller must hold the order row lock before this transaction's first snapshot read.
// That lock serializes payments for the order; locking an empty payment range would
// create MySQL gap-lock deadlocks between different orders inserting payments.
func (s *Service) pendingPaymentTx(tx *gorm.DB, order model.OrderMain, channel string) (*model.OrderPayment, error) {
	channel = strings.TrimSpace(channel)
	if channel == "" {
		channel = "mock"
	}
	var existing model.OrderPayment
	err := tx.
		Where("order_id = ? AND status IN ?", order.ID, []int8{model.PaymentStatusSuccess, model.PaymentStatusPending}).
		Order("status DESC, id ASC").First(&existing).Error
	if err == nil {
		if existing.Status == model.PaymentStatusSuccess {
			return nil, ErrOrderAlreadyPaid
		}
		if existing.Channel == channel {
			return &existing, nil
		}
		if existing.Channel != "mock" || channel != "wechat" {
			return nil, errors.New("已有其他支付渠道的待支付单")
		}
		// Legacy mock payment was never sent to WeChat. Retire it before real prepay.
		if err := tx.Model(&model.OrderPayment{}).Where("id = ? AND status = ?", existing.ID, model.PaymentStatusPending).Update("status", model.PaymentStatusClosed).Error; err != nil {
			return nil, err
		}
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	paymentNo, err := s.bizNoGen.Next(bizno.BizPrefixPayment)
	if err != nil {
		return nil, fmt.Errorf("生成支付单号失败: %w", err)
	}
	now := time.Now()
	payment := model.OrderPayment{
		PaymentNo: paymentNo,
		OrderID:   order.ID,
		Channel:   channel,
		Amount:    order.PayAmount,
		Status:    model.PaymentStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := tx.Create(&payment).Error; err != nil {
		return nil, fmt.Errorf("保存支付单失败: %w", err)
	}
	return &payment, nil
}

// MockPay processes a mock payment, deducting stock and transitioning order 10 -> 20.
// It is fully idempotent: repeated calls return success without deducting stock multiple times.
func (s *Service) MockPay(ctx context.Context, memberID int64, req MockPayRequest) (*model.OrderPayment, error) {
	if s.db == nil {
		return nil, errors.New("数据库服务不可用")
	}

	paymentNo := strings.TrimSpace(req.PaymentNo)
	orderID := req.OrderID

	if paymentNo == "" && orderID <= 0 {
		return nil, errors.New("必须提供 payment_no 或 order_id")
	}

	if memberID <= 0 {
		return nil, ErrOrderNotBelong
	}
	// Resolve the order without locking the payment. All mutations lock order first.
	if paymentNo != "" {
		var payment model.OrderPayment
		if err := s.db.WithContext(ctx).Where("payment_no = ?", paymentNo).First(&payment).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrPaymentNotFound
			}
			return nil, err
		}
		if orderID > 0 && orderID != payment.OrderID {
			return nil, errors.New("支付单与订单不匹配")
		}
		orderID = payment.OrderID
	}

	invSvc := productinventory.NewService(s.db)
	var resultPayment model.OrderPayment
	var affectedProdIDs []int64
	var isTimeout bool
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order model.OrderMain
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND deleted_at IS NULL", orderID).First(&order).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrOrderNotFound
			}
			return err
		}
		// Ownership must be checked before any idempotent success response.
		if order.MemberID != memberID {
			return ErrOrderNotBelong
		}
		if order.Status == model.OrderStatusCancelled {
			return ErrOrderCancelled
		}

		var payment model.OrderPayment
		if paymentNo != "" {
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("payment_no = ? AND order_id = ?", paymentNo, order.ID).First(&payment).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return ErrPaymentNotFound
				}
				return err
			}
		} else if order.Status == model.OrderStatusPendingPayment {
			p, err := s.pendingPaymentTx(tx, order, "mock")
			if err != nil {
				return err
			}
			payment = *p
		} else {
			// Retry by order ID returns the actual successful payment, never a new success.
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("order_id = ? AND status = ?", order.ID, model.PaymentStatusSuccess).
				Order("id ASC").First(&payment).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return pendingOrderError(order.Status)
				}
				return err
			}
		}
		if payment.Status == model.PaymentStatusSuccess {
			resultPayment = payment
			return nil
		}
		if err := pendingOrderError(order.Status); err != nil {
			return err
		}
		if payment.Status != model.PaymentStatusPending {
			return ErrPaymentClosed
		}

		now := time.Now()
		if ordertimeout.IsExpired(order.CreatedAt, now) {
			if err := s.closeTimeoutOrderTx(ctx, tx, &order, now); err != nil {
				return err
			}
			isTimeout = true
			return nil
		}

		// 4. Order is pending payment (status 10). Deduct stock and release locked_stock
		var items []model.OrderItem
		if err := tx.Where("order_id = ?", order.ID).Find(&items).Error; err != nil {
			return err
		}

		// Sort by sku_id ASC to prevent deadlocks
		sort.Slice(items, func(i, j int) bool {
			return items[i].SKUID < items[j].SKUID
		})

		operatorName := "member"
		for _, it := range items {
			if err := invSvc.OrderDeductLockedStockTx(ctx, tx, it.ProductID, it.SKUID, it.Quantity, order.OrderNo, memberID, operatorName); err != nil {
				return err
			}
		}

		// 5. Update OrderMain: 10 -> 20
		now = time.Now()
		res := tx.Model(&model.OrderMain{}).
			Where("id = ? AND status = ?", order.ID, model.OrderStatusPendingPayment).
			Updates(map[string]any{
				"status":     model.OrderStatusPendingShipment,
				"paid_at":    &now,
				"updated_at": now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return errors.New("订单状态变更失败或并发冲突")
		}

		// Increment sales_count for each product in this order
		prodSalesMap := make(map[int64]int)
		for _, it := range items {
			prodSalesMap[it.ProductID] += it.Quantity
		}
		for prodID, qty := range prodSalesMap {
			if err := tx.Model(&model.Product{}).Where("id = ?", prodID).
				Update("sales_count", gorm.Expr("sales_count + ?", qty)).Error; err != nil {
				return err
			}
			affectedProdIDs = append(affectedProdIDs, prodID)
		}

		// 6. Update OrderPayment: 10 -> 20
		txnNo := fmt.Sprintf("MOCK_TXN_%s", payment.PaymentNo)
		resPay := tx.Model(&model.OrderPayment{}).
			Where("id = ? AND status = ?", payment.ID, model.PaymentStatusPending).
			Updates(map[string]any{
				"status":         model.PaymentStatusSuccess,
				"transaction_no": &txnNo,
				"paid_at":        &now,
				"updated_at":     now,
			})
		if resPay.Error != nil {
			return resPay.Error
		}
		if resPay.RowsAffected != 1 {
			return errors.New("支付单状态变更失败或并发冲突")
		}

		// Close older duplicate pending payments in the same transaction.
		var duplicateIDs []int64
		if err := tx.Model(&model.OrderPayment{}).
			Where("order_id = ? AND id <> ? AND status = ?", order.ID, payment.ID, model.PaymentStatusPending).
			Pluck("id", &duplicateIDs).Error; err != nil {
			return err
		}
		if len(duplicateIDs) > 0 {
			if err := tx.Model(&model.OrderPayment{}).Where("id IN ?", duplicateIDs).
				Updates(map[string]any{"status": model.PaymentStatusClosed, "updated_at": now}).Error; err != nil {
				return err
			}
		}

		// 7. Record OrderLog
		orderLog := model.OrderLog{
			OrderID:      order.ID,
			FromStatus:   model.OrderStatusPendingPayment,
			ToStatus:     model.OrderStatusPendingShipment,
			Action:       "pay",
			OperatorType: "member",
			OperatorID:   memberID,
			OperatorName: operatorName,
			Remark:       "模拟支付成功",
			CreatedAt:    now,
		}
		if err := tx.Create(&orderLog).Error; err != nil {
			return err
		}

		payment.Status = model.PaymentStatusSuccess
		payment.PaidAt = &now
		payment.TransactionNo = &txnNo
		resultPayment = payment
		return nil
	})

	if err != nil {
		return nil, err
	}
	if isTimeout {
		return nil, ErrOrderExpired
	}
	if len(affectedProdIDs) > 0 {
		s.invalidateProductSalesCache(affectedProdIDs)
	}
	return &resultPayment, nil
}

// closeTimeoutOrderTx closes a timed out order and releases its locked stock within an existing transaction.
// The caller must hold an UPDATE row lock on order.
func (s *Service) closeTimeoutOrderTx(ctx context.Context, tx *gorm.DB, order *model.OrderMain, now time.Time) error {
	if err := s.EnsureClosableTx(ctx, tx, order.ID); err != nil {
		return err
	}
	invSvc := productinventory.NewService(s.db)

	var items []model.OrderItem
	if err := tx.Where("order_id = ?", order.ID).Find(&items).Error; err != nil {
		return err
	}

	// Sort by sku_id ASC to prevent deadlocks
	sort.Slice(items, func(i, j int) bool {
		return items[i].SKUID < items[j].SKUID
	})

	// Unlock locked stock
	for _, it := range items {
		if err := invSvc.OrderUnlockStockTx(ctx, tx, it.ProductID, it.SKUID, it.Quantity, order.OrderNo, 0, "system"); err != nil {
			return err
		}
	}

	// Update order: 10 -> 50
	res := tx.Model(&model.OrderMain{}).
		Where("id = ? AND status = ?", order.ID, model.OrderStatusPendingPayment).
		Updates(map[string]any{
			"status":       model.OrderStatusCancelled,
			"cancelled_at": &now,
			"updated_at":   now,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 1 {
		return errors.New("订单状态变更失败或并发冲突")
	}

	// Close pending payments
	if err := tx.Model(&model.OrderPayment{}).
		Where("order_id = ? AND status = ?", order.ID, model.PaymentStatusPending).
		Updates(map[string]any{
			"status":     model.PaymentStatusClosed,
			"updated_at": now,
		}).Error; err != nil {
		return err
	}

	// Record order log
	orderLog := model.OrderLog{
		OrderID:      order.ID,
		FromStatus:   model.OrderStatusPendingPayment,
		ToStatus:     model.OrderStatusCancelled,
		Action:       "timeout_close",
		OperatorType: "system",
		OperatorID:   0,
		OperatorName: "system",
		Remark:       "支付超时自动关闭",
		CreatedAt:    now,
	}
	if err := tx.Create(&orderLog).Error; err != nil {
		return err
	}

	order.Status = model.OrderStatusCancelled
	return nil
}

// CloseTimeoutOrder closes an order that has timed out without payment and releases its locked stock.
func (s *Service) CloseTimeoutOrder(ctx context.Context, orderID int64) error {
	_, err := s.closeTimeoutOrder(ctx, orderID, nil)
	return err
}

func (s *Service) closeTimeoutOrder(ctx context.Context, orderID int64, threshold *time.Time) (bool, error) {
	if s.db == nil {
		return false, errors.New("数据库服务不可用")
	}

	closed := false
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order model.OrderMain
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND deleted_at IS NULL", orderID).
			First(&order).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrOrderNotFound
			}
			return err
		}

		now := time.Now()
		// If not pending payment, do nothing (idempotent)
		if order.Status != model.OrderStatusPendingPayment || (threshold != nil && order.CreatedAt.After(*threshold)) {
			return nil
		}

		if err := s.closeTimeoutOrderTx(ctx, tx, &order, now); err != nil {
			return err
		}
		closed = true
		return nil
	})
	return closed && err == nil, err
}

// CloseExpiredOrders closes expired orders in bounded batches, reporting failures for retry.
func (s *Service) CloseExpiredOrders(ctx context.Context, expireDuration time.Duration) (int64, error) {
	if s.db == nil {
		return 0, errors.New("数据库服务不可用")
	}
	if expireDuration <= 0 {
		return 0, errors.New("支付超时时间必须大于0")
	}
	threshold := time.Now().Add(-expireDuration)
	var closedCount, lastID int64
	var firstErr error
	for {
		var orderIDs []int64
		if err := s.db.WithContext(ctx).Model(&model.OrderMain{}).
			Where("id > ? AND status = ? AND created_at <= ? AND deleted_at IS NULL", lastID, model.OrderStatusPendingPayment, threshold).
			Order("id ASC").Limit(100).Pluck("id", &orderIDs).Error; err != nil {
			return closedCount, err
		}
		if len(orderIDs) == 0 {
			return closedCount, firstErr
		}
		for _, id := range orderIDs {
			if err := ctx.Err(); err != nil {
				return closedCount, err
			}
			closed, err := s.closeTimeoutOrder(ctx, id, &threshold)
			if err != nil && firstErr == nil {
				firstErr = fmt.Errorf("关闭超时订单 %d 失败: %w", id, err)
			}
			if closed {
				closedCount++
			}
			lastID = id
		}
	}
}

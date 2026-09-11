package payment

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"mall/admin_core/internal/model"
	productinventory "mall/admin_core/internal/product/inventory"
	"mall/admin_core/pkg/bizno"
)

type Service struct {
	db       *gorm.DB
	bizNoGen *bizno.Generator
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

	var order model.OrderMain
	if err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", req.OrderID).
		First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}

	if memberID > 0 && order.MemberID != memberID {
		return nil, ErrOrderNotBelong
	}

	if order.Status == model.OrderStatusCancelled {
		return nil, ErrOrderCancelled
	}
	if order.Status == model.OrderStatusPendingShipment ||
		order.Status == model.OrderStatusPendingReceipt ||
		order.Status == model.OrderStatusCompleted {
		return nil, ErrOrderAlreadyPaid
	}
	if order.Status != model.OrderStatusPendingPayment {
		return nil, ErrOrderInvalidStatus
	}

	// Check existing payment records for idempotency
	var existing model.OrderPayment
	if err := s.db.WithContext(ctx).
		Where("order_id = ?", order.ID).
		Order("id DESC").
		First(&existing).Error; err == nil {
		if existing.Status == model.PaymentStatusSuccess {
			return nil, ErrOrderAlreadyPaid
		}
		if existing.Status == model.PaymentStatusPending {
			return &existing, nil
		}
	}

	// Generate P-prefixed payment business number
	paymentNo, err := s.bizNoGen.Next(bizno.BizPrefixPayment)
	if err != nil {
		return nil, fmt.Errorf("生成支付单号失败: %w", err)
	}

	channel := strings.TrimSpace(req.Channel)
	if channel == "" {
		channel = "mock"
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

	if err := s.db.WithContext(ctx).Create(&payment).Error; err != nil {
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

	invSvc := productinventory.NewService(s.db)
	var resultPayment model.OrderPayment

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var payment model.OrderPayment
		var err error

		// 1. Locate and lock payment
		if paymentNo != "" {
			err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("payment_no = ?", paymentNo).
				First(&payment).Error
		} else {
			err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("order_id = ? AND status != ?", orderID, model.PaymentStatusClosed).
				Order("id DESC").
				First(&payment).Error
		}

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// If payment was not created yet but orderID was provided and order is pending payment, create on the fly
				if orderID > 0 {
					var ord model.OrderMain
					if ordErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
						Where("id = ? AND deleted_at IS NULL", orderID).
						First(&ord).Error; ordErr == nil {
						if ord.Status == model.OrderStatusPendingPayment {
							pNo, genErr := s.bizNoGen.Next(bizno.BizPrefixPayment)
							if genErr != nil {
								return fmt.Errorf("生成支付单号失败: %w", genErr)
							}
							now := time.Now()
							payment = model.OrderPayment{
								PaymentNo: pNo,
								OrderID:   ord.ID,
								Channel:   "mock",
								Amount:    ord.PayAmount,
								Status:    model.PaymentStatusPending,
								CreatedAt: now,
								UpdatedAt: now,
							}
							if createErr := tx.Create(&payment).Error; createErr != nil {
								return createErr
							}
						}
					}
				}
				if payment.ID == 0 {
					return ErrPaymentNotFound
				}
			} else {
				return err
			}
		}

		// 2. Check payment status idempotency
		if payment.Status == model.PaymentStatusSuccess {
			// Idempotent: already paid, do not deduct inventory again!
			resultPayment = payment
			return nil
		}
		if payment.Status == model.PaymentStatusClosed || payment.Status == model.PaymentStatusFailed {
			return ErrPaymentClosed
		}

		// 3. Locate and lock OrderMain
		var order model.OrderMain
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND deleted_at IS NULL", payment.OrderID).
			First(&order).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrOrderNotFound
			}
			return err
		}

		if memberID > 0 && order.MemberID != memberID {
			return ErrOrderNotBelong
		}

		// Check order status
		if order.Status == model.OrderStatusCancelled {
			_ = tx.Model(&payment).Updates(map[string]any{
				"status":     model.PaymentStatusClosed,
				"updated_at": time.Now(),
			}).Error
			return ErrOrderCancelled
		}
		if order.Status == model.OrderStatusPendingShipment ||
			order.Status == model.OrderStatusPendingReceipt ||
			order.Status == model.OrderStatusCompleted {
			// Order is already paid in another transaction; update payment to success without deducting stock
			now := time.Now()
			txnNo := fmt.Sprintf("MOCK_TXN_%s", payment.PaymentNo)
			_ = tx.Model(&payment).Updates(map[string]any{
				"status":         model.PaymentStatusSuccess,
				"transaction_no": &txnNo,
				"paid_at":        order.PaidAt,
				"updated_at":     now,
			}).Error
			payment.Status = model.PaymentStatusSuccess
			payment.PaidAt = order.PaidAt
			payment.TransactionNo = &txnNo
			resultPayment = payment
			return nil
		}
		if order.Status != model.OrderStatusPendingPayment {
			return ErrOrderInvalidStatus
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
		now := time.Now()
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
	return &resultPayment, nil
}

// CloseTimeoutOrder closes an order that has timed out without payment and releases its locked stock.
func (s *Service) CloseTimeoutOrder(ctx context.Context, orderID int64) error {
	if s.db == nil {
		return errors.New("数据库服务不可用")
	}

	invSvc := productinventory.NewService(s.db)

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order model.OrderMain
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND deleted_at IS NULL", orderID).
			First(&order).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrOrderNotFound
			}
			return err
		}

		// If not pending payment, do nothing (idempotent)
		if order.Status != model.OrderStatusPendingPayment {
			return nil
		}

		var items []model.OrderItem
		if err := tx.Where("order_id = ?", order.ID).Find(&items).Error; err != nil {
			return err
		}

		// Sort by sku_id ASC
		sort.Slice(items, func(i, j int) bool {
			return items[i].SKUID < items[j].SKUID
		})

		// Unlock locked stock
		for _, it := range items {
			if err := invSvc.OrderUnlockStockTx(ctx, tx, it.ProductID, it.SKUID, it.Quantity, order.OrderNo, 0, "system"); err != nil {
				return err
			}
		}

		now := time.Now()
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
		_ = tx.Model(&model.OrderPayment{}).
			Where("order_id = ? AND status = ?", order.ID, model.PaymentStatusPending).
			Updates(map[string]any{
				"status":     model.PaymentStatusClosed,
				"updated_at": now,
			}).Error

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
		return tx.Create(&orderLog).Error
	})
}

// CloseExpiredOrders finds and closes all orders pending payment older than expireDuration.
func (s *Service) CloseExpiredOrders(ctx context.Context, expireDuration time.Duration) (int64, error) {
	if s.db == nil {
		return 0, errors.New("数据库服务不可用")
	}

	threshold := time.Now().Add(-expireDuration)
	var orderIDs []int64
	if err := s.db.WithContext(ctx).Model(&model.OrderMain{}).
		Where("status = ? AND created_at <= ? AND deleted_at IS NULL", model.OrderStatusPendingPayment, threshold).
		Pluck("id", &orderIDs).Error; err != nil {
		return 0, err
	}

	var closedCount int64
	for _, id := range orderIDs {
		if err := s.CloseTimeoutOrder(ctx, id); err == nil {
			closedCount++
		}
	}
	return closedCount, nil
}

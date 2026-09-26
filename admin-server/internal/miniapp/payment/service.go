package payment

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"mall/admin_core/internal/model"
	productinventory "mall/admin_core/internal/product/inventory"
	"mall/admin_core/internal/system/job"
	systemsetting "mall/admin_core/internal/system/setting"
	"mall/admin_core/pkg/bizno"
	"mall/admin_core/pkg/cache"
	"mall/admin_core/pkg/logger"
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

	paymentCfg, err := systemsetting.NewService(s.db).GetPayment(ctx)
	if err != nil {
		return nil, fmt.Errorf("读取支付设置失败: %w", err)
	}
	if !paymentCfg.WechatEnabled {
		return nil, errors.New("当前支付方式已关闭")
	}

	var payment *model.OrderPayment
	var isTimeout bool
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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
		if ordertimeout.IsOrderExpired(order.ExpireAt, order.CreatedAt, now) {
			if err := s.closeTimeoutOrderTx(ctx, tx, &order, now); err != nil {
				return err
			}
			isTimeout = true
			return nil
		}
		var err error
		payment, err = s.pendingPaymentTx(tx, order)
		return err
	})
	if err != nil {
		logger.Warn("create order payment failed",
			logger.Int64("member_id", memberID),
			logger.Int64("order_id", req.OrderID),
			logger.Err(err),
		)
		return nil, err
	}
	if isTimeout {
		logger.Warn("create order payment failed: order expired",
			logger.Int64("member_id", memberID),
			logger.Int64("order_id", req.OrderID),
		)
		return nil, ErrOrderExpired
	}
	logger.Info("order payment created",
		logger.Int64("member_id", memberID),
		logger.Int64("order_id", req.OrderID),
		logger.Int64("payment_id", payment.ID),
		logger.String("payment_no", payment.PaymentNo),
		logger.String("amount", payment.Amount),
		logger.Int8("status", payment.Status),
	)
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
func (s *Service) pendingPaymentTx(tx *gorm.DB, order model.OrderMain) (*model.OrderPayment, error) {
	var existing model.OrderPayment
	err := tx.
		Where("order_id = ? AND status IN ?", order.ID, []int8{model.PaymentStatusSuccess, model.PaymentStatusPending}).
		Order("status DESC, id ASC").First(&existing).Error
	if err == nil {
		if existing.Status == model.PaymentStatusSuccess {
			return nil, ErrOrderAlreadyPaid
		}
		if existing.Channel == "wechat" {
			return &existing, nil
		}
		if existing.Channel != "mock" {
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
		Channel:   "wechat",
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
	logger.Info("order closed due to payment timeout",
		logger.Int64("order_id", order.ID),
		logger.String("order_no", order.OrderNo),
		logger.Int64("member_id", order.MemberID),
	)
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
		if order.Status != model.OrderStatusPendingPayment {
			return nil
		}
		if threshold != nil {
			isExpired := false
			if order.ExpireAt != nil && !now.Before(*order.ExpireAt) {
				isExpired = true
			}
			if order.ExpireAt == nil && !order.CreatedAt.After(*threshold) {
				isExpired = true
			}
			if !isExpired {
				return nil
			}
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
	result, err := s.CloseExpiredOrdersWithResult(ctx, expireDuration)
	return int64(result.SuccessCount), err
}

func (s *Service) CloseExpiredOrdersWithResult(ctx context.Context, expireDuration time.Duration) (job.RunResult, error) {
	var result job.RunResult
	if s.db == nil {
		return result, errors.New("数据库服务不可用")
	}
	if expireDuration <= 0 {
		return result, errors.New("支付超时时间必须大于0")
	}
	now := time.Now()
	threshold := now.Add(-expireDuration)
	var lastID int64
	var firstErr error
	for {
		var orderIDs []int64
		if err := s.db.WithContext(ctx).Model(&model.OrderMain{}).
			Where("id > ? AND status = ? AND ((expire_at IS NOT NULL AND expire_at <= ?) OR (expire_at IS NULL AND created_at <= ?)) AND deleted_at IS NULL",
				lastID, model.OrderStatusPendingPayment, now, threshold).
			Order("id ASC").Limit(100).Pluck("id", &orderIDs).Error; err != nil {
			return result, err
		}
		if len(orderIDs) == 0 {
			return result, firstErr
		}
		for _, id := range orderIDs {
			if err := ctx.Err(); err != nil {
				return result, err
			}
			result.ProcessedCount++
			closed, err := s.closeTimeoutOrder(ctx, id, &threshold)
			if err != nil {
				result.FailedCount++
				if firstErr == nil {
					firstErr = fmt.Errorf("关闭超时订单 %d 失败: %w", id, err)
				}
			}
			if closed {
				result.SuccessCount++
			}
			lastID = id
		}
	}
}

// AutoReceiveShippedOrders scans orders in PendingReceipt where auto_receive_at <= NOW() and completes them.
func (s *Service) AutoReceiveShippedOrders(ctx context.Context) (int64, error) {
	result, err := s.AutoReceiveShippedOrdersWithResult(ctx)
	return int64(result.SuccessCount), err
}

func (s *Service) AutoReceiveShippedOrdersWithResult(ctx context.Context) (job.RunResult, error) {
	var result job.RunResult
	if s.db == nil {
		return result, errors.New("数据库服务不可用")
	}
	tradeCfg, err := systemsetting.NewService(s.db).GetTrade(ctx)
	if err != nil {
		return result, fmt.Errorf("读取交易设置失败: %w", err)
	}
	autoReceiveDays := tradeCfg.AutoReceiveDays
	if autoReceiveDays <= 0 {
		autoReceiveDays = 7
	}
	afterSaleDays := tradeCfg.AfterSaleDays

	now := time.Now()
	fallbackThreshold := now.Add(-time.Duration(autoReceiveDays) * 24 * time.Hour)
	var lastID int64
	var firstErr error

	for {
		var orders []model.OrderMain
		if err := s.db.WithContext(ctx).Model(&model.OrderMain{}).
			Where("id > ? AND status = ? AND ((auto_receive_at IS NOT NULL AND auto_receive_at <= ?) OR (auto_receive_at IS NULL AND shipped_at IS NOT NULL AND shipped_at <= ?)) AND deleted_at IS NULL",
				lastID, model.OrderStatusPendingReceipt, now, fallbackThreshold).
			Order("id ASC").Limit(100).Find(&orders).Error; err != nil {
			return result, err
		}
		if len(orders) == 0 {
			return result, firstErr
		}

		for _, o := range orders {
			lastID = o.ID
			if err := ctx.Err(); err != nil {
				return result, err
			}
			result.ProcessedCount++

			completed := false
			err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
				var current model.OrderMain
				if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND deleted_at IS NULL", o.ID).First(&current).Error; err != nil {
					return err
				}
				if current.Status != model.OrderStatusPendingReceipt {
					return nil
				}
				var activeCount int64
				if err := tx.Model(&model.AfterSale{}).
					Where("order_id = ? AND status IN ?", o.ID, []int8{
						model.AfterSaleStatusPendingReview,
						model.AfterSaleStatusWaitingBuyerReturn,
						model.AfterSaleStatusWaitingMerchantReceive,
						model.AfterSaleStatusRefunding,
						model.AfterSaleStatusRefundFailed,
					}).Count(&activeCount).Error; err != nil {
					return err
				}
				if activeCount > 0 {
					return nil
				}
				afterSaleDeadline := now.Add(time.Duration(afterSaleDays) * 24 * time.Hour)
				res := tx.Model(&model.OrderMain{}).
					Where("id = ? AND status = ?", o.ID, model.OrderStatusPendingReceipt).
					Updates(map[string]any{
						"status":              model.OrderStatusCompleted,
						"completed_at":        &now,
						"after_sale_deadline": &afterSaleDeadline,
						"updated_at":          now,
					})
				if res.Error != nil {
					return res.Error
				}
				if res.RowsAffected != 1 {
					return nil
				}

				completed = true
				orderLog := model.OrderLog{
					OrderID:      o.ID,
					FromStatus:   model.OrderStatusPendingReceipt,
					ToStatus:     model.OrderStatusCompleted,
					Action:       "auto_receive",
					OperatorType: "system",
					OperatorID:   0,
					OperatorName: "system",
					Remark:       "系统自动确认收货",
					CreatedAt:    now,
				}
				return tx.Create(&orderLog).Error
			})

			if err != nil {
				result.FailedCount++
				if firstErr == nil {
					firstErr = fmt.Errorf("自动确认收货订单 %d 失败: %w", o.ID, err)
				}
			} else if completed {
				result.SuccessCount++
			}
		}
	}
}

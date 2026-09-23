package payment

import (
	"context"
	"log"
	"time"

	"mall/admin_core/internal/model"
	"mall/admin_core/internal/order/wechatpay"
	"mall/admin_core/pkg/ordertimeout"
)

const (
	DefaultOrderExpiry   = ordertimeout.DefaultOrderExpiry
	DefaultSweepInterval = time.Minute
)

// RunTimeoutWorker sweeps on startup and on each tick until ctx is cancelled.
// Start it once from main, not during router construction or in request handlers.
func (s *Service) RunTimeoutWorker(ctx context.Context, expiry, interval time.Duration) {
	if expiry <= 0 || interval <= 0 {
		log.Print("order timeout worker requires positive expiry and interval")
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		if err := s.ReconcilePendingWechat(ctx); err != nil && ctx.Err() == nil {
			log.Printf("payment reconcile failed: %v", err)
		}
		closed, err := s.CloseExpiredOrders(ctx, expiry)
		if err != nil && ctx.Err() == nil {
			log.Printf("order timeout sweep failed (closed %d): %v", closed, err)
		} else if closed > 0 {
			log.Printf("order timeout sweep closed %d orders", closed)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// ReconcilePendingWechat repairs missed callbacks before the expiry sweep.
func (s *Service) ReconcilePendingWechat(ctx context.Context) error {
	var rows []model.OrderPayment
	if err := s.db.WithContext(ctx).Where("channel = ? AND status = ?", "wechat", model.PaymentStatusPending).Order("id ASC").Limit(100).Find(&rows).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	cfg, err := (wechatpay.Store{DB: s.db}).Load(ctx)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if err := ctx.Err(); err != nil {
			return err
		}
		txn, err := cfg.QueryPayment(ctx, row.PaymentNo)
		if err != nil {
			log.Printf("payment reconcile query failed payment_no=%s: %v", row.PaymentNo, err)
			continue
		}
		if txn != nil && txn.TradeState != nil && *txn.TradeState == "SUCCESS" {
			if _, err = s.ConfirmWechat(ctx, txn); err != nil {
				log.Printf("payment reconcile confirm failed payment_no=%s: %v", row.PaymentNo, err)
			}
		}
	}
	return nil
}

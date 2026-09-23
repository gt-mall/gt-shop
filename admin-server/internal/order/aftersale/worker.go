package aftersale

import (
	"context"
	"log"
	"mall/admin_core/internal/model"
	"time"
)

// RunRefundWorker reconciles accepted refunds when callbacks are delayed or lost.
func (s *Service) RunRefundWorker(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		if err := s.ReconcilePendingRefunds(ctx); err != nil && ctx.Err() == nil {
			log.Printf("refund reconcile failed: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Service) ReconcilePendingRefunds(ctx context.Context) error {
	var rows []model.OrderRefund
	if err := s.db.WithContext(ctx).Where("provider = ? AND status = ?", "wechat", model.RefundStatusProcessing).Order("id ASC").Limit(100).Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := s.ReconcileRefund(ctx, row.AfterSaleID); err != nil {
			log.Printf("refund reconcile query failed refund_no=%s: %v", row.RefundNo, err)
		}
	}
	return nil
}

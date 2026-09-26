package aftersale

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"mall/admin_core/internal/model"
	"mall/admin_core/internal/system/job"
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
	_, err := s.ReconcilePendingRefundsWithResult(ctx)
	return err
}

// ReconcilePendingRefundsWithResult keeps processing the batch and reports
// partial failures to the shared job manager.
func (s *Service) ReconcilePendingRefundsWithResult(ctx context.Context) (job.RunResult, error) {
	var result job.RunResult
	var rows []model.OrderRefund
	if err := s.db.WithContext(ctx).Where("provider = ? AND status = ?", "wechat", model.RefundStatusProcessing).Order("id ASC").Limit(100).Find(&rows).Error; err != nil {
		return result, err
	}
	result.ProcessedCount = len(rows)
	permanentFailure := false
	firstFailure := ""
	for _, row := range rows {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		if err := s.ReconcileRefund(ctx, row.AfterSaleID); err != nil {
			if ctx.Err() != nil {
				return result, ctx.Err()
			}
			reason := "退款状态处理失败"
			if strings.HasPrefix(err.Error(), "查询退款状态失败:") {
				reason = "微信退款状态查询失败"
			}
			log.Printf("refund reconcile failed refund_no=%s reason=%s", row.RefundNo, reason)
			result.FailedCount++
			permanentFailure = permanentFailure || job.IsPermanentError(err)
			if firstFailure == "" {
				firstFailure = fmt.Sprintf("；首条退款单 %s：%s", row.RefundNo, reason)
			}
		} else {
			result.SuccessCount++
		}
	}
	if result.FailedCount > 0 {
		err := fmt.Errorf("微信退款对账有 %d 条记录失败%s", result.FailedCount, firstFailure)
		if permanentFailure {
			return result, job.NoRetry(err)
		}
		return result, err
	}
	return result, nil
}

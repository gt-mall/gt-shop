package payment

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"mall/admin_core/internal/model"
	"mall/admin_core/internal/order/wechatpay"
	"mall/admin_core/internal/system/job"
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
		received, err := s.AutoReceiveShippedOrders(ctx)
		if err != nil && ctx.Err() == nil {
			log.Printf("auto receive sweep failed (received %d): %v", received, err)
		} else if received > 0 {
			log.Printf("auto receive sweep completed %d orders", received)
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
	_, err := s.ReconcilePendingWechatWithResult(ctx)
	return err
}

// ReconcilePendingWechatWithResult reports partial provider failures while
// continuing to inspect all eligible payment records in the current batch.
func (s *Service) ReconcilePendingWechatWithResult(ctx context.Context) (job.RunResult, error) {
	var result job.RunResult
	var rows []model.OrderPayment
	if err := s.db.WithContext(ctx).Where("channel = ? AND status = ?", "wechat", model.PaymentStatusPending).Order("id ASC").Limit(100).Find(&rows).Error; err != nil {
		return result, err
	}
	if len(rows) == 0 {
		return result, nil
	}
	result.ProcessedCount = len(rows)
	cfg, err := (wechatpay.Store{DB: s.db}).Load(ctx)
	if err != nil {
		result.FailedCount = len(rows)
		if job.IsPermanentError(err) {
			err = job.NoRetry(err)
		}
		return result, err
	}
	if err := cfg.Validate(); err != nil {
		result.FailedCount = len(rows)
		return result, job.NoRetry(err)
	}
	return ReconcileWechatPaymentRows(ctx, rows, func(ctx context.Context, row model.OrderPayment) error {
		txn, err := cfg.QueryPayment(ctx, row.PaymentNo)
		if err != nil {
			log.Printf("payment reconcile query failed payment_no=%s reason=微信支付状态查询失败", row.PaymentNo)
			return reconcileFailure{reason: "微信支付状态查询失败", cause: err}
		}
		if txn != nil && txn.TradeState != nil && *txn.TradeState == "SUCCESS" {
			if _, err := s.ConfirmWechat(ctx, txn); err != nil {
				reason := "微信支付确认失败"
				if errors.Is(err, ErrWechatPaidOrderClosed) {
					reason = "微信交易成功但订单已关闭，需人工退款"
				}
				log.Printf("payment reconcile confirm failed payment_no=%s reason=%s", row.PaymentNo, reason)
				return reconcileFailure{reason: reason, cause: err}
			}
		}
		return nil
	})
}

type reconcileFailure struct {
	reason string
	cause  error
}

func (e reconcileFailure) Error() string { return e.reason }
func (e reconcileFailure) Unwrap() error { return e.cause }

// ReconcileWechatPaymentRows processes the complete input batch and returns
// aggregate failure counts without exposing provider response bodies.
func ReconcileWechatPaymentRows(ctx context.Context, rows []model.OrderPayment, reconcile func(context.Context, model.OrderPayment) error) (job.RunResult, error) {
	result := job.RunResult{ProcessedCount: len(rows)}
	permanentFailure := false
	firstFailure := ""
	for _, row := range rows {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		if err := reconcile(ctx, row); err != nil {
			if ctx.Err() != nil {
				return result, ctx.Err()
			}
			result.FailedCount++
			permanentFailure = permanentFailure || job.IsPermanentError(err)
			if firstFailure == "" {
				reason := "处理失败"
				var safeFailure reconcileFailure
				if errors.As(err, &safeFailure) {
					reason = safeFailure.reason
				}
				firstFailure = fmt.Sprintf("；首条支付单 %s：%s", row.PaymentNo, reason)
			}
			continue
		}
		result.SuccessCount++
	}
	if result.FailedCount > 0 {
		err := fmt.Errorf("微信支付对账有 %d 条记录失败%s", result.FailedCount, firstFailure)
		if permanentFailure {
			return result, job.NoRetry(err)
		}
		return result, err
	}
	return result, nil
}

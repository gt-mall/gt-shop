package reconcile_pending_wechat_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"mall/admin_core/internal/miniapp/payment"
	"mall/admin_core/internal/model"
)

func TestSinglePaymentFailureDoesNotStopBatch(t *testing.T) {
	rows := []model.OrderPayment{{PaymentNo: "payment-fail"}, {PaymentNo: "payment-ok"}}
	result, err := payment.ReconcileWechatPaymentRows(context.Background(), rows, func(_ context.Context, row model.OrderPayment) error {
		if row.PaymentNo == "payment-fail" {
			return errors.New("provider response body must not be retained")
		}
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "微信支付对账有 1 条记录失败") || !strings.Contains(err.Error(), "payment-fail") {
		t.Fatalf("expected a safe aggregate error, got %v", err)
	}
	if result.ProcessedCount != 2 || result.SuccessCount != 1 || result.FailedCount != 1 {
		t.Fatalf("batch stopped early or reported incorrect counts: %+v", result)
	}
	if strings.Contains(fmt.Sprint(err), "provider response body must not be retained") {
		t.Fatalf("raw provider error escaped the aggregate: %v", err)
	}
}

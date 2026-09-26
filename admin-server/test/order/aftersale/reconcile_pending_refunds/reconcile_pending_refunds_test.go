package reconcile_pending_refunds_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"mall/admin_core/internal/model"
	"mall/admin_core/internal/order/aftersale"
	"mall/admin_core/internal/order/refund"
	"mall/admin_core/internal/system/job"
)

type partialQueryProvider struct{}

func (partialQueryProvider) Channel() string { return "wechat" }
func (partialQueryProvider) Refund(context.Context, refund.RefundRequest) (*refund.RefundResult, error) {
	return nil, errors.New("unused")
}
func (partialQueryProvider) QueryRefund(_ context.Context, refundNo, _ string) (*refund.RefundQueryResult, error) {
	if refundNo == "refund-fail" {
		return nil, errors.New("temporary provider failure")
	}
	return &refund.RefundQueryResult{Status: refund.RefundQueryStatusProcessing}, nil
}

func TestPartialRefundFailureDoesNotStopBatch(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?cache=shared", t.TempDir()+"/refunds.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.AfterSale{}, &model.OrderRefund{}); err != nil {
		t.Fatal(err)
	}
	for i, refundNo := range []string{"refund-fail", "refund-ok"} {
		afterSale := model.AfterSale{AfterSaleNo: fmt.Sprintf("after-sale-%d", i), RequestID: fmt.Sprintf("request-%d", i), OrderID: int64(i + 1), MemberID: int64(i + 1), Type: model.AfterSaleTypeRefundOnly, Status: model.AfterSaleStatusRefunding, OrderStatusAtApply: model.OrderStatusPendingReceipt, AppliedAt: time.Now()}
		if err := db.Create(&afterSale).Error; err != nil {
			t.Fatal(err)
		}
		orderRefund := model.OrderRefund{RefundNo: refundNo, AfterSaleID: afterSale.ID, OrderID: afterSale.OrderID, MemberID: afterSale.MemberID, Status: model.RefundStatusProcessing, Provider: "wechat", CreatedAt: time.Now(), UpdatedAt: time.Now()}
		if err := db.Create(&orderRefund).Error; err != nil {
			t.Fatal(err)
		}
	}

	result, err := aftersale.NewService(db, nil, partialQueryProvider{}).ReconcilePendingRefundsWithResult(context.Background())
	if err == nil || job.IsPermanentError(err) {
		t.Fatalf("expected one partial failure and an aggregate error, result=%+v err=%v", result, err)
	}
	if result.ProcessedCount != 2 || result.SuccessCount != 1 || result.FailedCount != 1 {
		t.Fatalf("batch stopped early or reported incorrect counts: %+v", result)
	}
	if !strings.Contains(err.Error(), "refund-fail") || !strings.Contains(err.Error(), "微信退款状态查询失败") || strings.Contains(err.Error(), "temporary provider failure") {
		t.Fatalf("refund error summary is not actionable or safe: %v", err)
	}
}

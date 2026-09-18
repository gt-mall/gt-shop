package flow

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"mall/admin_core/internal/analytics"
	"mall/admin_core/internal/model"
	"mall/admin_core/pkg/util"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

type rawFlowRow struct {
	ID            int64  `gorm:"column:id"`
	BizNo         string `gorm:"column:biz_no"`
	FlowType      string `gorm:"column:flow_type"`
	Direction     string `gorm:"column:direction"`
	OrderID       int64  `gorm:"column:order_id"`
	OrderNo       string `gorm:"column:order_no"`
	Channel       string `gorm:"column:channel"`
	Amount        string `gorm:"column:amount"`
	TransactionNo string `gorm:"column:transaction_no"`
	Status        int8   `gorm:"column:status"`
	OccurredAt    string `gorm:"column:occurred_at"`
	CreatedAt     string `gorm:"column:created_at"`
}

type flowSummaryRow struct {
	IncomeSum  string `gorm:"column:income_sum"`
	ExpenseSum string `gorm:"column:expense_sum"`
}

func (s *Service) List(ctx context.Context, req FlowListRequest) (FlowListResponse, error) {
	if s.db == nil {
		return FlowListResponse{}, errors.New("数据库服务不可用")
	}

	flowType := strings.ToLower(strings.TrimSpace(req.FlowType))
	if flowType == "income" {
		flowType = "payment"
	} else if flowType == "expense" {
		flowType = "refund"
	}

	// Build inner subquery
	var baseQuery string
	if flowType == "payment" {
		baseQuery = `
			SELECT 
				p.id AS id,
				p.payment_no AS biz_no,
				'payment' AS flow_type,
				'income' AS direction,
				p.order_id AS order_id,
				COALESCE(o.order_no, '') AS order_no,
				p.channel AS channel,
				p.amount AS amount,
				COALESCE(p.transaction_no, '') AS transaction_no,
				p.status AS status,
				COALESCE(p.paid_at, p.created_at) AS occurred_at,
				p.created_at AS created_at
			FROM order_payment p
			LEFT JOIN order_main o ON p.order_id = o.id
		`
	} else if flowType == "refund" {
		baseQuery = `
			SELECT 
				r.id AS id,
				r.refund_no AS biz_no,
				'refund' AS flow_type,
				'expense' AS direction,
				r.order_id AS order_id,
				COALESCE(o.order_no, '') AS order_no,
				r.provider AS channel,
				r.refund_amount AS amount,
				COALESCE(r.provider_refund_no, '') AS transaction_no,
				r.status AS status,
				COALESCE(r.refunded_at, r.created_at) AS occurred_at,
				r.created_at AS created_at
			FROM order_refund r
			LEFT JOIN order_main o ON r.order_id = o.id
		`
	} else {
		baseQuery = `
			SELECT 
				p.id AS id,
				p.payment_no AS biz_no,
				'payment' AS flow_type,
				'income' AS direction,
				p.order_id AS order_id,
				COALESCE(o.order_no, '') AS order_no,
				p.channel AS channel,
				p.amount AS amount,
				COALESCE(p.transaction_no, '') AS transaction_no,
				p.status AS status,
				COALESCE(p.paid_at, p.created_at) AS occurred_at,
				p.created_at AS created_at
			FROM order_payment p
			LEFT JOIN order_main o ON p.order_id = o.id
			UNION ALL
			SELECT 
				r.id AS id,
				r.refund_no AS biz_no,
				'refund' AS flow_type,
				'expense' AS direction,
				r.order_id AS order_id,
				COALESCE(o.order_no, '') AS order_no,
				r.provider AS channel,
				r.refund_amount AS amount,
				COALESCE(r.provider_refund_no, '') AS transaction_no,
				r.status AS status,
				COALESCE(r.refunded_at, r.created_at) AS occurred_at,
				r.created_at AS created_at
			FROM order_refund r
			LEFT JOIN order_main o ON r.order_id = o.id
		`
	}

	subQuery := fmt.Sprintf("(%s) AS f", baseQuery)
	tx := s.db.WithContext(ctx).Table(subQuery)

	if req.BizNo != "" {
		tx = tx.Where("f.biz_no LIKE ?", "%"+strings.TrimSpace(req.BizNo)+"%")
	}
	if req.OrderNo != "" {
		tx = tx.Where("f.order_no LIKE ?", "%"+strings.TrimSpace(req.OrderNo)+"%")
	}
	if req.Channel != "" {
		tx = tx.Where("f.channel = ?", strings.TrimSpace(req.Channel))
	}
	// Status filter: only effective when flowType is "payment" or "refund"
	// When flowType is "all" or empty, status is ignored to prevent status code collisions
	if req.Status != nil && (flowType == "payment" || flowType == "refund") {
		tx = tx.Where("f.status = ?", *req.Status)
	}
	if req.StartTime != "" {
		tx = tx.Where("f.occurred_at >= ?", strings.TrimSpace(req.StartTime))
	}
	if req.EndTime != "" {
		etStr := strings.TrimSpace(req.EndTime)
		if len(etStr) == 10 {
			if t, err := time.ParseInLocation("2006-01-02", etStr, time.Local); err == nil {
				etStr = t.AddDate(0, 0, 1).Format("2006-01-02 15:04:05")
			}
		} else if strings.HasSuffix(etStr, "23:59:59") {
			if t, err := time.ParseInLocation("2006-01-02 15:04:05", etStr, time.Local); err == nil {
				etStr = t.Add(time.Second).Format("2006-01-02 15:04:05")
			}
		}
		tx = tx.Where("f.occurred_at < ?", etStr)
	}

	// Count total
	var total int64
	if err := tx.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return FlowListResponse{}, err
	}

	// Compute summaries of matching query (only successfully completed flows)
	var sumRow flowSummaryRow
	err := tx.Session(&gorm.Session{}).Select(`
		COALESCE(SUM(CASE WHEN f.direction = 'income' AND f.status = ? THEN f.amount ELSE 0 END), 0) AS income_sum,
		COALESCE(SUM(CASE WHEN f.direction = 'expense' AND f.status = ? THEN f.amount ELSE 0 END), 0) AS expense_sum
	`, model.PaymentStatusSuccess, model.RefundStatusSuccess).Scan(&sumRow).Error
	if err != nil {
		return FlowListResponse{}, err
	}

	incomeCents, _ := util.ParseDecimalYuanToCent(sumRow.IncomeSum)
	expenseCents, _ := util.ParseDecimalYuanToCent(sumRow.ExpenseSum)
	netCents := incomeCents - expenseCents

	// Pagination
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	var rows []rawFlowRow
	err = tx.Session(&gorm.Session{}).
		Order("f.occurred_at DESC, f.id DESC").
		Limit(pageSize).
		Offset(offset).
		Scan(&rows).Error
	if err != nil {
		return FlowListResponse{}, err
	}

	items := make([]FinanceFlowRow, len(rows))
	for i, r := range rows {
		var statusText string
		if r.FlowType == "payment" {
			statusText = analytics.PaymentStatusText(r.Status)
		} else {
			statusText = analytics.RefundStatusText(r.Status)
		}
		cents, _ := util.ParseDecimalYuanToCent(r.Amount)
		items[i] = FinanceFlowRow{
			ID:            r.ID,
			BizNo:         r.BizNo,
			FlowType:      r.FlowType,
			Direction:     r.Direction,
			OrderID:       r.OrderID,
			OrderNo:       r.OrderNo,
			Channel:       r.Channel,
			Amount:        util.FormatCentToYuan(cents),
			TransactionNo: r.TransactionNo,
			Status:        r.Status,
			StatusText:    statusText,
			OccurredAt:    analytics.ParseDateTime(r.OccurredAt),
			CreatedAt:     analytics.ParseDateTime(r.CreatedAt),
		}
	}

	return FlowListResponse{
		Total:              total,
		Items:              items,
		TotalIncomeAmount:  util.FormatCentToYuan(incomeCents),
		TotalExpenseAmount: util.FormatCentToYuan(expenseCents),
		NetFlowAmount:      analytics.FormatSignedCentToYuan(netCents),
	}, nil
}

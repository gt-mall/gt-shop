package overview

import (
	"context"
	"errors"
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

type paymentAggRow struct {
	PayAmount  string `gorm:"column:pay_amount"`
	PaidOrders int64  `gorm:"column:paid_orders"`
}

type dateAmountRow struct {
	D   string `gorm:"column:d"`
	Amt string `gorm:"column:amt"`
}

type dateCountRow struct {
	D   string `gorm:"column:d"`
	Cnt int64  `gorm:"column:cnt"`
}

type topProductQueryRow struct {
	ProductID      int64  `gorm:"column:product_id"`
	ProductName    string `gorm:"column:product_name"`
	ImageURL       string `gorm:"column:image_url"`
	Quantity       int64  `gorm:"column:quantity"`
	SalesAmount    string `gorm:"column:sales_amount"`
	PaidOrderCount int64  `gorm:"column:paid_order_count"`
}

type flowQueryRow struct {
	ID            int64  `gorm:"column:id"`
	BizNo         string `gorm:"column:biz_no"`
	FlowType      string `gorm:"column:flow_type"`
	Direction     string `gorm:"column:direction"`
	OrderID       int64  `gorm:"column:order_id"`
	OrderNo       string `gorm:"column:order_no"`
	Channel       string `gorm:"column:channel"`
	Amount        string `gorm:"column:amount"`
	Status        int8   `gorm:"column:status"`
	TransactionNo string `gorm:"column:transaction_no"`
	OccurredAt    string `gorm:"column:occurred_at"`
}

func (s *Service) Overview(ctx context.Context, req OverviewQueryRequest) (OverviewResponse, error) {
	if s.db == nil {
		return OverviewResponse{}, errors.New("数据库服务不可用")
	}

	startTime, endTime, prevStartTime, prevEndTime, dates, err := analytics.ParseTimeRange(
		req.TimeRange,
		req.StartTime,
		req.EndTime,
	)
	if err != nil {
		return OverviewResponse{}, err
	}

	// 1. Current period metrics
	currPayCents, currPaidOrders, currPaidMembers, currRefundCents, err := s.calcPeriodFinancials(ctx, startTime, endTime)
	if err != nil {
		return OverviewResponse{}, err
	}
	currNetSalesCents := currPayCents - currRefundCents

	// 2. Previous cycle metrics (for growth rates)
	prevPayCents, prevPaidOrders, prevPaidMembers, prevRefundCents, err := s.calcPeriodFinancials(ctx, prevStartTime, prevEndTime)
	if err != nil {
		return OverviewResponse{}, err
	}
	prevNetSalesCents := prevPayCents - prevRefundCents

	var prevAovCents, currAovCents int64
	if currPaidOrders > 0 && currPayCents > 0 {
		currAovCents = currPayCents / currPaidOrders
	}
	if prevPaidOrders > 0 && prevPayCents > 0 {
		prevAovCents = prevPayCents / prevPaidOrders
	}

	metrics := OverviewMetric{
		PayAmount:                 util.FormatCentToYuan(currPayCents),
		RefundAmount:              util.FormatCentToYuan(currRefundCents),
		NetSalesAmount:            analytics.FormatSignedCentToYuan(currNetSalesCents),
		PaidOrderCount:            currPaidOrders,
		PaidMemberCount:           currPaidMembers,
		AverageOrderValue:         analytics.SafeAOV(currPayCents, currPaidOrders),
		PayAmountGrowthRate:       analytics.CalcGrowthRate(currPayCents, prevPayCents),
		RefundAmountGrowthRate:    analytics.CalcGrowthRate(currRefundCents, prevRefundCents),
		NetSalesGrowthRate:        analytics.CalcGrowthRate(currNetSalesCents, prevNetSalesCents),
		PaidOrderCountGrowthRate:  analytics.CalcCountGrowthRate(currPaidOrders, prevPaidOrders),
		PaidMemberCountGrowthRate: analytics.CalcCountGrowthRate(currPaidMembers, prevPaidMembers),
		AOVGrowthRate:             analytics.CalcGrowthRate(currAovCents, prevAovCents),
	}

	// 3. Sales trends (Pay amount, Refund amount, Net sales by date)
	salesTrends, err := s.querySalesTrends(ctx, startTime, endTime, dates)
	if err != nil {
		return OverviewResponse{}, err
	}

	// 4. Order trends (Created orders, Paid orders by date)
	orderTrends, err := s.queryOrderTrends(ctx, startTime, endTime, dates)
	if err != nil {
		return OverviewResponse{}, err
	}

	// 5. Top 10 Products
	topProducts, err := s.queryTopProducts(ctx, startTime, endTime)
	if err != nil {
		return OverviewResponse{}, err
	}

	// 6. Recent financial flows (Last 10)
	recentFlows, err := s.queryRecentFlows(ctx)
	if err != nil {
		return OverviewResponse{}, err
	}

	return OverviewResponse{
		Metrics:     metrics,
		SalesTrends: salesTrends,
		OrderTrends: orderTrends,
		TopProducts: topProducts,
		RecentFlows: recentFlows,
	}, nil
}

func (s *Service) calcPeriodFinancials(ctx context.Context, start, end time.Time) (
	payCents, paidOrders, paidMembers, refundCents int64, err error,
) {
	// Pay amount & paid orders
	var payRow paymentAggRow
	err = s.db.WithContext(ctx).Table("order_payment").
		Select("COALESCE(SUM(amount), 0) AS pay_amount, COUNT(DISTINCT order_id) AS paid_orders").
		Where("status = ? AND paid_at >= ? AND paid_at < ?", model.PaymentStatusSuccess, start, end).
		Scan(&payRow).Error
	if err != nil {
		return 0, 0, 0, 0, err
	}
	payCents, _ = util.ParseDecimalYuanToCent(payRow.PayAmount)
	paidOrders = payRow.PaidOrders

	// Paid members
	err = s.db.WithContext(ctx).Table("order_payment AS p").
		Joins("INNER JOIN order_main AS o ON p.order_id = o.id").
		Where("p.status = ? AND p.paid_at >= ? AND p.paid_at < ?", model.PaymentStatusSuccess, start, end).
		Select("COUNT(DISTINCT o.member_id)").
		Scan(&paidMembers).Error
	if err != nil {
		return 0, 0, 0, 0, err
	}

	// Refund amount
	var refundAmt string
	err = s.db.WithContext(ctx).Table("order_refund").
		Select("COALESCE(SUM(refund_amount), 0)").
		Where("status = ? AND refunded_at >= ? AND refunded_at < ?", model.RefundStatusSuccess, start, end).
		Scan(&refundAmt).Error
	if err != nil {
		return 0, 0, 0, 0, err
	}
	refundCents, _ = util.ParseDecimalYuanToCent(refundAmt)

	return payCents, paidOrders, paidMembers, refundCents, nil
}

func (s *Service) querySalesTrends(ctx context.Context, start, end time.Time, dates []string) ([]SalesTrendItem, error) {
	datePayMap := make(map[string]int64, len(dates))
	dateRefundMap := make(map[string]int64, len(dates))
	for _, d := range dates {
		datePayMap[d] = 0
		dateRefundMap[d] = 0
	}

	// Daily payments
	var payRows []dateAmountRow
	err := s.db.WithContext(ctx).Table("order_payment").
		Select("DATE(paid_at) AS d, COALESCE(SUM(amount), 0) AS amt").
		Where("status = ? AND paid_at >= ? AND paid_at < ?", model.PaymentStatusSuccess, start, end).
		Group("DATE(paid_at)").
		Scan(&payRows).Error
	if err != nil {
		return nil, err
	}
	for _, r := range payRows {
		d := strings.TrimSpace(r.D)
		if len(d) >= 10 {
			d = d[:10]
		}
		c, _ := util.ParseDecimalYuanToCent(r.Amt)
		datePayMap[d] = c
	}

	// Daily refunds
	var refundRows []dateAmountRow
	err = s.db.WithContext(ctx).Table("order_refund").
		Select("DATE(refunded_at) AS d, COALESCE(SUM(refund_amount), 0) AS amt").
		Where("status = ? AND refunded_at >= ? AND refunded_at < ?", model.RefundStatusSuccess, start, end).
		Group("DATE(refunded_at)").
		Scan(&refundRows).Error
	if err != nil {
		return nil, err
	}
	for _, r := range refundRows {
		d := strings.TrimSpace(r.D)
		if len(d) >= 10 {
			d = d[:10]
		}
		c, _ := util.ParseDecimalYuanToCent(r.Amt)
		dateRefundMap[d] = c
	}

	items := make([]SalesTrendItem, len(dates))
	for i, d := range dates {
		pCents := datePayMap[d]
		rCents := dateRefundMap[d]
		netCents := pCents - rCents
		items[i] = SalesTrendItem{
			Date:           d,
			PayAmount:      util.FormatCentToYuan(pCents),
			RefundAmount:   util.FormatCentToYuan(rCents),
			NetSalesAmount: analytics.FormatSignedCentToYuan(netCents),
		}
	}
	return items, nil
}

func (s *Service) queryOrderTrends(ctx context.Context, start, end time.Time, dates []string) ([]OrderTrendItem, error) {
	dateCreatedMap := make(map[string]int64, len(dates))
	datePaidMap := make(map[string]int64, len(dates))
	for _, d := range dates {
		dateCreatedMap[d] = 0
		datePaidMap[d] = 0
	}

	// Daily created orders
	var createdRows []dateCountRow
	err := s.db.WithContext(ctx).Table("order_main").
		Select("DATE(created_at) AS d, COUNT(*) AS cnt").
		Where("deleted_at IS NULL AND created_at >= ? AND created_at < ?", start, end).
		Group("DATE(created_at)").
		Scan(&createdRows).Error
	if err != nil {
		return nil, err
	}
	for _, r := range createdRows {
		d := strings.TrimSpace(r.D)
		if len(d) >= 10 {
			d = d[:10]
		}
		dateCreatedMap[d] = r.Cnt
	}

	// Daily paid orders
	var paidRows []dateCountRow
	err = s.db.WithContext(ctx).Table("order_payment").
		Select("DATE(paid_at) AS d, COUNT(DISTINCT order_id) AS cnt").
		Where("status = ? AND paid_at >= ? AND paid_at < ?", model.PaymentStatusSuccess, start, end).
		Group("DATE(paid_at)").
		Scan(&paidRows).Error
	if err != nil {
		return nil, err
	}
	for _, r := range paidRows {
		d := strings.TrimSpace(r.D)
		if len(d) >= 10 {
			d = d[:10]
		}
		datePaidMap[d] = r.Cnt
	}

	items := make([]OrderTrendItem, len(dates))
	for i, d := range dates {
		items[i] = OrderTrendItem{
			Date:           d,
			OrderCount:     dateCreatedMap[d],
			PaidOrderCount: datePaidMap[d],
		}
	}
	return items, nil
}

func (s *Service) queryTopProducts(ctx context.Context, start, end time.Time) ([]TopProductItem, error) {
	var rows []topProductQueryRow
	err := s.db.WithContext(ctx).Table("order_item AS i").
		Select(`
			i.product_id,
			MAX(i.product_name) AS product_name,
			COALESCE(MAX(i.sku_image_url), '') AS image_url,
			COALESCE(SUM(i.quantity), 0) AS quantity,
			COALESCE(SUM(i.total_amount), 0) AS sales_amount,
			COUNT(DISTINCT i.order_id) AS paid_order_count
		`).
		Where("i.order_id IN (SELECT DISTINCT order_id FROM order_payment WHERE status = ? AND paid_at >= ? AND paid_at < ?)", model.PaymentStatusSuccess, start, end).
		Group("i.product_id").
		Order("sales_amount DESC, quantity DESC").
		Limit(10).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	items := make([]TopProductItem, len(rows))
	for i, r := range rows {
		cents, _ := util.ParseDecimalYuanToCent(r.SalesAmount)
		items[i] = TopProductItem{
			ProductID:      r.ProductID,
			ProductName:    r.ProductName,
			ImageURL:       r.ImageURL,
			Quantity:       r.Quantity,
			SalesAmount:    util.FormatCentToYuan(cents),
			PaidOrderCount: r.PaidOrderCount,
		}
	}
	return items, nil
}

func (s *Service) queryRecentFlows(ctx context.Context) ([]RecentFlowItem, error) {
	// Query union of recent payment (success) and refund (success)
	var rows []flowQueryRow
	query := `
		SELECT id, biz_no, flow_type, direction, order_id, order_no, channel, amount, status, transaction_no, occurred_at
		FROM (
			SELECT 
				p.id AS id,
				p.payment_no AS biz_no,
				'payment' AS flow_type,
				'income' AS direction,
				p.order_id AS order_id,
				COALESCE(o.order_no, '') AS order_no,
				p.channel AS channel,
				p.amount AS amount,
				p.status AS status,
				COALESCE(p.transaction_no, '') AS transaction_no,
				COALESCE(p.paid_at, p.created_at) AS occurred_at
			FROM order_payment p
			LEFT JOIN order_main o ON p.order_id = o.id
			WHERE p.status = ?
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
				r.status AS status,
				COALESCE(r.provider_refund_no, '') AS transaction_no,
				COALESCE(r.refunded_at, r.created_at) AS occurred_at
			FROM order_refund r
			LEFT JOIN order_main o ON r.order_id = o.id
			WHERE r.status = ?
		) AS u
		ORDER BY occurred_at DESC, id DESC
		LIMIT 10
	`
	err := s.db.WithContext(ctx).Raw(query, model.PaymentStatusSuccess, model.RefundStatusSuccess).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	items := make([]RecentFlowItem, len(rows))
	for i, r := range rows {
		var statusText string
		if r.FlowType == "payment" {
			statusText = analytics.PaymentStatusText(r.Status)
		} else {
			statusText = analytics.RefundStatusText(r.Status)
		}
		cents, _ := util.ParseDecimalYuanToCent(r.Amount)
		items[i] = RecentFlowItem{
			ID:            r.ID,
			BizNo:         r.BizNo,
			FlowType:      r.FlowType,
			Direction:     r.Direction,
			OrderID:       r.OrderID,
			OrderNo:       r.OrderNo,
			Channel:       r.Channel,
			Amount:        util.FormatCentToYuan(cents),
			Status:        r.Status,
			StatusText:    statusText,
			TransactionNo: r.TransactionNo,
			OccurredAt:    analytics.ParseDateTime(r.OccurredAt),
		}
	}
	return items, nil
}

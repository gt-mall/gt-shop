package trade

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

type tradePaymentAggRow struct {
	PayAmount  string `gorm:"column:pay_amount"`
	PaidOrders int64  `gorm:"column:paid_orders"`
}

type tradeOrderDiscountFreightRow struct {
	DiscountAmount string `gorm:"column:discount_amount"`
	FreightAmount  string `gorm:"column:freight_amount"`
}

type dateAmountRow struct {
	D   string `gorm:"column:d"`
	Amt string `gorm:"column:amt"`
}

type dateCountRow struct {
	D   string `gorm:"column:d"`
	Cnt int64  `gorm:"column:cnt"`
}

type orderStatusCountRow struct {
	Status int8  `gorm:"column:status"`
	Cnt    int64 `gorm:"column:cnt"`
}

type channelAggRow struct {
	Channel   string `gorm:"column:channel"`
	PayCount  int64  `gorm:"column:pay_count"`
	PayAmount string `gorm:"column:pay_amount"`
}

func (s *Service) Trade(ctx context.Context, req TradeQueryRequest) (TradeResponse, error) {
	if s.db == nil {
		return TradeResponse{}, errors.New("数据库服务不可用")
	}

	startTime, endTime, _, _, dates, err := analytics.ParseTimeRange(
		req.TimeRange,
		req.StartTime,
		req.EndTime,
	)
	if err != nil {
		return TradeResponse{}, err
	}

	// 1. Metrics
	// 1.1 Total created orders & cancelled orders in period
	var orderCount int64
	err = s.db.WithContext(ctx).Table("order_main").
		Where("deleted_at IS NULL AND created_at >= ? AND created_at < ?", startTime, endTime).
		Count(&orderCount).Error
	if err != nil {
		return TradeResponse{}, err
	}

	// Cancelled orders: strictly attributes to actual cancellation occurrence time (cancelled_at).
	// Abnormal historical records with cancelled_at IS NULL are excluded from period cancellation metrics.
	var cancelledCount int64
	err = s.db.WithContext(ctx).Table("order_main").
		Where("deleted_at IS NULL AND status = ? AND cancelled_at >= ? AND cancelled_at < ?", model.OrderStatusCancelled, startTime, endTime).
		Count(&cancelledCount).Error
	if err != nil {
		return TradeResponse{}, err
	}

	// 1.2 Paid amount and paid orders
	var payRow tradePaymentAggRow
	err = s.db.WithContext(ctx).Table("order_payment").
		Select("COALESCE(SUM(amount), 0) AS pay_amount, COUNT(DISTINCT order_id) AS paid_orders").
		Where("status = ? AND paid_at >= ? AND paid_at < ?", model.PaymentStatusSuccess, startTime, endTime).
		Scan(&payRow).Error
	if err != nil {
		return TradeResponse{}, err
	}
	payCents, _ := util.ParseDecimalYuanToCent(payRow.PayAmount)
	paidOrders := payRow.PaidOrders

	// 1.3 Discount and freight amounts for paid orders
	var dfRow tradeOrderDiscountFreightRow
	err = s.db.WithContext(ctx).Table("order_main AS o").
		Select("COALESCE(SUM(o.discount_amount), 0) AS discount_amount, COALESCE(SUM(o.freight_amount), 0) AS freight_amount").
		Where("o.deleted_at IS NULL").
		Where("o.id IN (SELECT DISTINCT order_id FROM order_payment WHERE status = ? AND paid_at >= ? AND paid_at < ?)", model.PaymentStatusSuccess, startTime, endTime).
		Scan(&dfRow).Error
	if err != nil {
		return TradeResponse{}, err
	}
	discountCents, _ := util.ParseDecimalYuanToCent(dfRow.DiscountAmount)
	freightCents, _ := util.ParseDecimalYuanToCent(dfRow.FreightAmount)

	metrics := TradeMetric{
		OrderCount:          orderCount,
		PaidOrderCount:      paidOrders,
		CancelledOrderCount: cancelledCount,
		PayAmount:           util.FormatCentToYuan(payCents),
		DiscountAmount:      util.FormatCentToYuan(discountCents),
		FreightAmount:       util.FormatCentToYuan(freightCents),
		AverageOrderValue:   analytics.SafeAOV(payCents, paidOrders),
	}

	// 2. Order trends (Created, Paid, Cancelled)
	orderTrends, err := s.queryOrderTrends(ctx, startTime, endTime, dates)
	if err != nil {
		return TradeResponse{}, err
	}

	// 3. Amount trends (Pay, Refund, Net sales)
	amountTrends, err := s.queryAmountTrends(ctx, startTime, endTime, dates)
	if err != nil {
		return TradeResponse{}, err
	}

	// 4. Order status distribution
	statusDist, err := s.queryOrderStatusDistribution(ctx, startTime, endTime, orderCount)
	if err != nil {
		return TradeResponse{}, err
	}

	// 5. Payment channel distribution
	channelDist, err := s.queryChannelDistribution(ctx, startTime, endTime, payCents)
	if err != nil {
		return TradeResponse{}, err
	}

	return TradeResponse{
		Metrics:                  metrics,
		OrderTrends:              orderTrends,
		AmountTrends:             amountTrends,
		OrderStatusDistributions: statusDist,
		ChannelDistributions:     channelDist,
	}, nil
}

func (s *Service) queryOrderTrends(ctx context.Context, start, end time.Time, dates []string) ([]TradeOrderTrendItem, error) {
	dateCreatedMap := make(map[string]int64, len(dates))
	datePaidMap := make(map[string]int64, len(dates))
	dateCancelledMap := make(map[string]int64, len(dates))
	for _, d := range dates {
		dateCreatedMap[d] = 0
		datePaidMap[d] = 0
		dateCancelledMap[d] = 0
	}

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

	var cancelledRows []dateCountRow
	err = s.db.WithContext(ctx).Table("order_main").
		Select("DATE(cancelled_at) AS d, COUNT(*) AS cnt").
		Where("deleted_at IS NULL AND status = ? AND cancelled_at >= ? AND cancelled_at < ?", model.OrderStatusCancelled, start, end).
		Group("DATE(cancelled_at)").
		Scan(&cancelledRows).Error
	if err != nil {
		return nil, err
	}
	for _, r := range cancelledRows {
		d := strings.TrimSpace(r.D)
		if len(d) >= 10 {
			d = d[:10]
		}
		dateCancelledMap[d] = r.Cnt
	}

	items := make([]TradeOrderTrendItem, len(dates))
	for i, d := range dates {
		items[i] = TradeOrderTrendItem{
			Date:                d,
			OrderCount:          dateCreatedMap[d],
			PaidOrderCount:      datePaidMap[d],
			CancelledOrderCount: dateCancelledMap[d],
		}
	}
	return items, nil
}

func (s *Service) queryAmountTrends(ctx context.Context, start, end time.Time, dates []string) ([]TradeAmountTrendItem, error) {
	datePayMap := make(map[string]int64, len(dates))
	dateRefundMap := make(map[string]int64, len(dates))
	for _, d := range dates {
		datePayMap[d] = 0
		dateRefundMap[d] = 0
	}

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

	items := make([]TradeAmountTrendItem, len(dates))
	for i, d := range dates {
		pCents := datePayMap[d]
		rCents := dateRefundMap[d]
		netCents := pCents - rCents
		items[i] = TradeAmountTrendItem{
			Date:           d,
			PayAmount:      util.FormatCentToYuan(pCents),
			RefundAmount:   util.FormatCentToYuan(rCents),
			NetSalesAmount: analytics.FormatSignedCentToYuan(netCents),
		}
	}
	return items, nil
}

func (s *Service) queryOrderStatusDistribution(ctx context.Context, start, end time.Time, totalOrders int64) ([]OrderStatusDistributionItem, error) {
	statusList := []int8{
		model.OrderStatusPendingPayment,
		model.OrderStatusPendingShipment,
		model.OrderStatusPendingReceipt,
		model.OrderStatusCompleted,
		model.OrderStatusCancelled,
		model.OrderStatusRefunded,
	}

	var rows []orderStatusCountRow
	err := s.db.WithContext(ctx).Table("order_main").
		Select("status, COUNT(*) AS cnt").
		Where("deleted_at IS NULL AND created_at >= ? AND created_at < ?", start, end).
		Group("status").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	countMap := make(map[int8]int64, len(statusList))
	for _, st := range statusList {
		countMap[st] = 0
	}
	for _, r := range rows {
		countMap[r.Status] = r.Cnt
	}

	items := make([]OrderStatusDistributionItem, len(statusList))
	for i, st := range statusList {
		cnt := countMap[st]
		items[i] = OrderStatusDistributionItem{
			Status:     st,
			StatusText: model.OrderStatusText(st),
			Count:      cnt,
			Percentage: analytics.SafePercentage(cnt, totalOrders),
		}
	}
	return items, nil
}

func (s *Service) queryChannelDistribution(ctx context.Context, start, end time.Time, totalPayCents int64) ([]ChannelDistributionItem, error) {
	var rows []channelAggRow
	err := s.db.WithContext(ctx).Table("order_payment").
		Select("channel, COUNT(*) AS pay_count, COALESCE(SUM(amount), 0) AS pay_amount").
		Where("status = ? AND paid_at >= ? AND paid_at < ?", model.PaymentStatusSuccess, start, end).
		Group("channel").
		Order("pay_amount DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	formatChannelName := func(c string) string {
		switch strings.ToLower(c) {
		case "wechat":
			return "微信支付"
		case "mock":
			return "模拟支付"
		case "alipay":
			return "支付宝"
		default:
			if c == "" {
				return "其他支付"
			}
			return c
		}
	}

	items := make([]ChannelDistributionItem, len(rows))
	for i, r := range rows {
		cents, _ := util.ParseDecimalYuanToCent(r.PayAmount)
		items[i] = ChannelDistributionItem{
			Channel:     r.Channel,
			ChannelName: formatChannelName(r.Channel),
			PayCount:    r.PayCount,
			PayAmount:   util.FormatCentToYuan(cents),
			Percentage:  analytics.SafePercentage(cents, totalPayCents),
		}
	}
	return items, nil
}

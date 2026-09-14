package dashboard

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"mall/admin_core/internal/model"
	"mall/admin_core/pkg/util"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

type orderOverviewRow struct {
	TodayOrderCount      int64  `gorm:"column:today_order_count"`
	TodayPaidOrderCount  int64  `gorm:"column:today_paid_order_count"`
	TodaySalesAmount     string `gorm:"column:today_sales_amount"`
	PendingPaymentCount  int64  `gorm:"column:pending_payment_count"`
	PendingDeliveryCount int64  `gorm:"column:pending_delivery_count"`
}

type productOverviewRow struct {
	ProductCount       int64 `gorm:"column:product_count"`
	OnSaleProductCount int64 `gorm:"column:on_sale_product_count"`
}

type memberOverviewRow struct {
	MemberCount         int64 `gorm:"column:member_count"`
	TodayNewMemberCount int64 `gorm:"column:today_new_member_count"`
}

type trendQueryRow struct {
	StatDate    string `gorm:"column:stat_date"`
	OrderCount  int64  `gorm:"column:order_count"`
	SalesAmount string `gorm:"column:sales_amount"`
}

// Overview computes the aggregated data for the admin dashboard.
func (s *Service) Overview(ctx context.Context) (DashboardOverviewResponse, error) {
	if s.db == nil {
		return DashboardOverviewResponse{}, errors.New("数据库连接不可用")
	}

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayEnd := todayStart.AddDate(0, 0, 1)

	var resp DashboardOverviewResponse

	// 1. 订单维度统计（今日订单、今日已支付订单、今日支付销售额、待支付订单、待发货订单）
	// 统计口径说明：
	// - 今日订单：今日创建的所有非软删除订单
	// - 今日已支付订单：今日创建且状态为待发货(20)/待收货(30)/已完成(40)的订单
	// - 今日销售额：今日创建且已支付订单的实付金额总和（未支付10/已取消50/已退款60不计入）
	// - 待支付订单：当前系统中所有状态为待付款(10)的订单
	// - 待发货订单：当前系统中所有状态为待发货(20)的订单
	var orderRow orderOverviewRow
	err := s.db.WithContext(ctx).Table("order_main").
		Select(`
			COUNT(CASE WHEN created_at >= ? AND created_at < ? THEN 1 END) AS today_order_count,
			COUNT(CASE WHEN created_at >= ? AND created_at < ? AND status IN (?, ?, ?) THEN 1 END) AS today_paid_order_count,
			COALESCE(SUM(CASE WHEN created_at >= ? AND created_at < ? AND status IN (?, ?, ?) THEN pay_amount ELSE 0 END), 0) AS today_sales_amount,
			COUNT(CASE WHEN status = ? THEN 1 END) AS pending_payment_count,
			COUNT(CASE WHEN status = ? THEN 1 END) AS pending_delivery_count
		`,
			todayStart, todayEnd,
			todayStart, todayEnd, model.OrderStatusPendingShipment, model.OrderStatusPendingReceipt, model.OrderStatusCompleted,
			todayStart, todayEnd, model.OrderStatusPendingShipment, model.OrderStatusPendingReceipt, model.OrderStatusCompleted,
			model.OrderStatusPendingPayment,
			model.OrderStatusPendingShipment,
		).
		Where("deleted_at IS NULL").
		Scan(&orderRow).Error
	if err != nil {
		return resp, err
	}

	resp.TodayOrderCount = orderRow.TodayOrderCount
	resp.TodayPaidOrderCount = orderRow.TodayPaidOrderCount
	cents, err := util.ParseDecimalYuanToCent(orderRow.TodaySalesAmount)
	if err == nil {
		resp.TodaySalesAmount = util.FormatCentToYuan(cents)
	} else {
		resp.TodaySalesAmount = "0.00"
	}
	resp.PendingPaymentCount = orderRow.PendingPaymentCount
	resp.PendingDeliveryCount = orderRow.PendingDeliveryCount

	// 2. 待售后统计
	// 统计口径说明：
	// - 待售后：统计需要商家/后台处理的售后单，包括待审核(10)、待商家收货(30)及退款失败(80)
	var pendingAftersaleCount int64
	err = s.db.WithContext(ctx).Table("after_sale").
		Where("status IN (?, ?, ?)",
			model.AfterSaleStatusPendingReview,
			model.AfterSaleStatusWaitingMerchantReceive,
			model.AfterSaleStatusRefundFailed,
		).
		Count(&pendingAftersaleCount).Error
	if err != nil {
		return resp, err
	}
	resp.PendingAftersaleCount = pendingAftersaleCount

	// 3. 商品维度统计（商品总数、在售商品数）
	var prodRow productOverviewRow
	err = s.db.WithContext(ctx).Table("product").
		Select(`
			COUNT(*) AS product_count,
			COUNT(CASE WHEN status = 1 THEN 1 END) AS on_sale_product_count
		`).
		Where("deleted_at IS NULL").
		Scan(&prodRow).Error
	if err != nil {
		return resp, err
	}
	resp.ProductCount = prodRow.ProductCount
	resp.OnSaleProductCount = prodRow.OnSaleProductCount

	// 4. 会员维度统计（会员总数、今日新增会员数）
	var memberRow memberOverviewRow
	err = s.db.WithContext(ctx).Table("member").
		Select(`
			COUNT(*) AS member_count,
			COUNT(CASE WHEN created_at >= ? AND created_at < ? THEN 1 END) AS today_new_member_count
		`, todayStart, todayEnd).
		Where("deleted_at IS NULL").
		Scan(&memberRow).Error
	if err != nil {
		return resp, err
	}
	resp.MemberCount = memberRow.MemberCount
	resp.TodayNewMemberCount = memberRow.TodayNewMemberCount

	// 5. 低库存SKU统计
	// 统计口径说明：
	// - 可售库存 = stock - locked_stock
	// - 阈值优先使用 SKU 配置的 warning_stock（当 > 0 时），未设置预警阈值时默认以 <= 10 判定为低库存
	var lowStockSKUCount int64
	err = s.db.WithContext(ctx).Table("product_sku AS s").
		Joins("INNER JOIN product AS p ON p.id = s.product_id AND p.deleted_at IS NULL").
		Joins("LEFT JOIN product_sku_inventory AS i ON i.sku_id = s.id").
		Where("s.deleted_at IS NULL").
		Where("COALESCE(i.stock - i.locked_stock, 0) <= CASE WHEN i.warning_stock IS NOT NULL AND i.warning_stock > 0 THEN i.warning_stock ELSE 10 END").
		Count(&lowStockSKUCount).Error
	if err != nil {
		return resp, err
	}
	resp.LowStockSKUCount = lowStockSKUCount

	return resp, nil
}

// Trend computes daily order count and sales amounts over the given day range (7 or 30 days).
func (s *Service) Trend(ctx context.Context, req DashboardTrendRequest) (DashboardTrendResponse, error) {
	if s.db == nil {
		return DashboardTrendResponse{}, errors.New("数据库连接不可用")
	}

	days := req.Days
	if days != 30 {
		days = 7
	}

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayEnd := todayStart.AddDate(0, 0, 1)
	startDate := todayStart.AddDate(0, 0, -(days - 1))

	// Pre-generate full continuous dates slice
	dates := make([]string, days)
	dateMap := make(map[string]int64, days)
	salesMap := make(map[string]string, days)

	for i := 0; i < days; i++ {
		d := startDate.AddDate(0, 0, i).Format("2006-01-02")
		dates[i] = d
		dateMap[d] = 0
		salesMap[d] = "0.00"
	}

	// 数据库聚合查询：按自然日统计订单量与已支付订单销售额
	var queryRows []trendQueryRow
	err := s.db.WithContext(ctx).Table("order_main").
		Select(`
			DATE(created_at) AS stat_date,
			COUNT(*) AS order_count,
			COALESCE(SUM(CASE WHEN status IN (?, ?, ?) THEN pay_amount ELSE 0 END), 0) AS sales_amount
		`, model.OrderStatusPendingShipment, model.OrderStatusPendingReceipt, model.OrderStatusCompleted).
		Where("deleted_at IS NULL AND created_at >= ? AND created_at < ?", startDate, todayEnd).
		Group("DATE(created_at)").
		Scan(&queryRows).Error
	if err != nil {
		return DashboardTrendResponse{}, err
	}

	for _, r := range queryRows {
		rawDate := strings.TrimSpace(r.StatDate)
		if len(rawDate) >= 10 {
			rawDate = rawDate[:10]
		}
		if _, ok := dateMap[rawDate]; ok {
			dateMap[rawDate] = r.OrderCount
			cents, parseErr := util.ParseDecimalYuanToCent(r.SalesAmount)
			if parseErr == nil {
				salesMap[rawDate] = util.FormatCentToYuan(cents)
			} else {
				salesMap[rawDate] = "0.00"
			}
		}
	}

	orderCounts := make([]int64, days)
	salesAmounts := make([]string, days)
	for i, d := range dates {
		orderCounts[i] = dateMap[d]
		salesAmounts[i] = salesMap[d]
	}

	return DashboardTrendResponse{
		Dates:        dates,
		OrderCounts:  orderCounts,
		SalesAmounts: salesAmounts,
	}, nil
}

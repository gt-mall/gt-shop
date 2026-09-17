package overview

import "time"

// OverviewQueryRequest represents query parameters for financial overview.
type OverviewQueryRequest struct {
	TimeRange string `form:"time_range" json:"time_range" example:"today"` // today, yesterday, last_7_days, last_30_days, this_month, last_month, custom
	StartTime string `form:"start_time" json:"start_time" example:"2026-09-01 00:00:00"`
	EndTime   string `form:"end_time" json:"end_time" example:"2026-09-17 23:59:59"`
} // @name AnalyticsOverviewQueryRequest

// OverviewMetric represents high-level metrics with cycle-over-cycle comparisons.
type OverviewMetric struct {
	PayAmount                 string `json:"pay_amount" example:"12800.00"`
	RefundAmount              string `json:"refund_amount" example:"650.00"`
	NetSalesAmount            string `json:"net_sales_amount" example:"12150.00"`
	PaidOrderCount            int64  `json:"paid_order_count" example:"64"`
	PaidMemberCount           int64  `json:"paid_member_count" example:"52"`
	AverageOrderValue         string `json:"average_order_value" example:"200.00"`
	PayAmountGrowthRate       string `json:"pay_amount_growth_rate" example:"+12.50%"`
	RefundAmountGrowthRate    string `json:"refund_amount_growth_rate" example:"-5.00%"`
	NetSalesGrowthRate        string `json:"net_sales_growth_rate" example:"+14.20%"`
	PaidOrderCountGrowthRate  string `json:"paid_order_count_growth_rate" example:"+8.33%"`
	PaidMemberCountGrowthRate string `json:"paid_member_count_growth_rate" example:"+6.12%"`
	AOVGrowthRate             string `json:"aov_growth_rate" example:"+3.85%"`
} // @name AnalyticsOverviewMetric

// SalesTrendItem represents daily sales, refund and net amounts.
type SalesTrendItem struct {
	Date           string `json:"date" example:"2026-09-10"`
	PayAmount      string `json:"pay_amount" example:"1200.00"`
	RefundAmount   string `json:"refund_amount" example:"150.00"`
	NetSalesAmount string `json:"net_sales_amount" example:"1050.00"`
} // @name AnalyticsSalesTrendItem

// OrderTrendItem represents daily created and paid order counts.
type OrderTrendItem struct {
	Date           string `json:"date" example:"2026-09-10"`
	OrderCount     int64  `json:"order_count" example:"20"`      // 下单订单数
	PaidOrderCount int64  `json:"paid_order_count" example:"16"` // 支付订单数
} // @name AnalyticsOrderTrendItem

// TopProductItem represents top 10 sold products in the selected period.
type TopProductItem struct {
	ProductID      int64  `json:"product_id" example:"1"`
	ProductName    string `json:"product_name" example:"经典纯棉T恤"`
	ImageURL       string `json:"image_url" example:"https://example.com/p1.png"`
	Quantity       int64  `json:"quantity" example:"120"`
	SalesAmount    string `json:"sales_amount" example:"11880.00"`
	PaidOrderCount int64  `json:"paid_order_count" example:"85"`
} // @name AnalyticsTopProductItem

// RecentFlowItem represents a recent payment or refund record.
type RecentFlowItem struct {
	ID            int64     `json:"id" example:"101"`
	BizNo         string    `json:"biz_no" example:"PAY202609170001"`
	FlowType      string    `json:"flow_type" example:"payment"` // payment 或 refund
	Direction     string    `json:"direction" example:"income"`  // income 或 expense
	OrderID       int64     `json:"order_id" example:"1"`
	OrderNo       string    `json:"order_no" example:"ORD202609170001"`
	Channel       string    `json:"channel" example:"wechat"`
	Amount        string    `json:"amount" example:"199.00"`
	Status        int8      `json:"status" example:"20"`
	StatusText    string    `json:"status_text" example:"支付成功"`
	TransactionNo string    `json:"transaction_no" example:"WX_TRANS_123456"`
	OccurredAt    time.Time `json:"occurred_at"`
} // @name AnalyticsRecentFlowItem

// OverviewResponse represents the full financial overview data.
type OverviewResponse struct {
	Metrics     OverviewMetric   `json:"metrics"`
	SalesTrends []SalesTrendItem `json:"sales_trends"`
	OrderTrends []OrderTrendItem `json:"order_trends"`
	TopProducts []TopProductItem `json:"top_products"`
	RecentFlows []RecentFlowItem `json:"recent_flows"`
} // @name AnalyticsOverviewResponse

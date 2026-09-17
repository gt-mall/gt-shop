package trade

// TradeQueryRequest represents query parameters for trade analytics.
type TradeQueryRequest struct {
	TimeRange string `form:"time_range" json:"time_range" example:"last_7_days"` // today, yesterday, last_7_days, last_30_days, this_month, last_month, custom
	StartTime string `form:"start_time" json:"start_time" example:"2026-09-01 00:00:00"`
	EndTime   string `form:"end_time" json:"end_time" example:"2026-09-17 23:59:59"`
} // @name AnalyticsTradeQueryRequest

// TradeMetric represents summary metrics for trade analysis.
type TradeMetric struct {
	OrderCount          int64  `json:"order_count" example:"100"`           // 下单订单数
	PaidOrderCount      int64  `json:"paid_order_count" example:"80"`       // 支付订单数
	CancelledOrderCount int64  `json:"cancelled_order_count" example:"15"`  // 取消订单数
	PayAmount           string `json:"pay_amount" example:"16000.00"`       // 支付金额
	DiscountAmount      string `json:"discount_amount" example:"800.00"`    // 优惠金额
	FreightAmount       string `json:"freight_amount" example:"400.00"`     // 运费金额
	AverageOrderValue   string `json:"average_order_value" example:"200.00"`// 客单价
} // @name AnalyticsTradeMetric

// TradeOrderTrendItem represents daily order status volume.
type TradeOrderTrendItem struct {
	Date                string `json:"date" example:"2026-09-10"`
	OrderCount          int64  `json:"order_count" example:"15"`
	PaidOrderCount      int64  `json:"paid_order_count" example:"12"`
	CancelledOrderCount int64  `json:"cancelled_order_count" example:"2"`
} // @name AnalyticsTradeOrderTrendItem

// TradeAmountTrendItem represents daily monetary volume.
type TradeAmountTrendItem struct {
	Date           string `json:"date" example:"2026-09-10"`
	PayAmount      string `json:"pay_amount" example:"2400.00"`
	RefundAmount   string `json:"refund_amount" example:"200.00"`
	NetSalesAmount string `json:"net_sales_amount" example:"2200.00"`
} // @name AnalyticsTradeAmountTrendItem

// OrderStatusDistributionItem represents order count by status.
type OrderStatusDistributionItem struct {
	Status     int8   `json:"status" example:"20"`
	StatusText string `json:"status_text" example:"待发货"`
	Count      int64  `json:"count" example:"25"`
	Percentage string `json:"percentage" example:"25.00%"`
} // @name AnalyticsOrderStatusDistributionItem

// ChannelDistributionItem represents payment volume by channel.
type ChannelDistributionItem struct {
	Channel     string `json:"channel" example:"wechat"`
	ChannelName string `json:"channel_name" example:"微信支付"`
	PayCount    int64  `json:"pay_count" example:"70"`
	PayAmount   string `json:"pay_amount" example:"14000.00"`
	Percentage  string `json:"percentage" example:"87.50%"`
} // @name AnalyticsChannelDistributionItem

// TradeResponse represents full trade analytics response.
type TradeResponse struct {
	Metrics                  TradeMetric                   `json:"metrics"`
	OrderTrends              []TradeOrderTrendItem         `json:"order_trends"`
	AmountTrends             []TradeAmountTrendItem        `json:"amount_trends"`
	OrderStatusDistributions []OrderStatusDistributionItem `json:"order_status_distributions"`
	ChannelDistributions     []ChannelDistributionItem     `json:"channel_distributions"`
} // @name AnalyticsTradeResponse

package dashboard

// DashboardOverviewResponse represents the statistics summary for admin home.
type DashboardOverviewResponse struct {
	TodayOrderCount     int64  `json:"today_order_count" example:"12"`
	TodayPaidOrderCount int64  `json:"today_paid_order_count" example:"10"`
	TodaySalesAmount    string `json:"today_sales_amount" example:"1280.00"` // 今日实际支付销售额（单位：元，十进制字符串保留精度）

	PendingPaymentCount   int64 `json:"pending_payment_count" example:"5"`   // 待支付订单数
	PendingDeliveryCount  int64 `json:"pending_delivery_count" example:"8"`  // 待发货订单数（已支付且未发货）
	PendingAftersaleCount int64 `json:"pending_aftersale_count" example:"2"` // 待处理售后数（待审核/待收货/退款失败）

	ProductCount       int64 `json:"product_count" example:"100"`        // 商品总数
	OnSaleProductCount int64 `json:"on_sale_product_count" example:"80"` // 在售商品数
	LowStockSKUCount   int64 `json:"low_stock_sku_count" example:"3"`    // 低库存SKU数

	MemberCount         int64 `json:"member_count" example:"500"`         // 会员总数
	TodayNewMemberCount int64 `json:"today_new_member_count" example:"6"` // 今日新增会员数
} // @name DashboardOverviewResponse

// DashboardTrendRequest represents query parameters for trend statistics.
type DashboardTrendRequest struct {
	Days int `form:"days" json:"days" example:"7"` // 趋势天数，支持 7 或 30，默认 7
} // @name DashboardTrendRequest

// DashboardTrendResponse represents daily sales and order trends.
type DashboardTrendResponse struct {
	Dates        []string `json:"dates" example:"2026-09-08,2026-09-09"`   // 连续自然日列表
	OrderCounts  []int64  `json:"order_counts" example:"12,18"`            // 每日订单量
	SalesAmounts []string `json:"sales_amounts" example:"1200.00,1850.00"` // 每日销售额（单位：元，十进制字符串保留精度）
} // @name DashboardTrendResponse

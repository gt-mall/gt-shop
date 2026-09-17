package product

// ProductRankingsRequest represents query parameters for product rankings.
type ProductRankingsRequest struct {
	TimeRange string `form:"time_range" json:"time_range" example:"last_7_days"`
	StartTime string `form:"start_time" json:"start_time" example:"2026-09-01 00:00:00"`
	EndTime   string `form:"end_time" json:"end_time" example:"2026-09-17 23:59:59"`
} // @name AnalyticsProductRankingsRequest

// ProductRankingItem represents an item in top product rankings.
type ProductRankingItem struct {
	ProductID      int64  `json:"product_id" example:"1"`
	ProductName    string `json:"product_name" example:"纯棉圆领T恤"`
	ImageURL       string `json:"image_url" example:"https://example.com/p1.jpg"`
	SalesQuantity  int64  `json:"sales_quantity" example:"150"`
	SalesAmount    string `json:"sales_amount" example:"14850.00"`
	RefundQuantity int64  `json:"refund_quantity" example:"5"`
	RefundAmount   string `json:"refund_amount" example:"495.00"`
	RefundRate     string `json:"refund_rate" example:"3.33%"`
} // @name AnalyticsProductRankingItem

// ProductRankingsResponse represents top rankings for sales volume, amount, and refunds.
type ProductRankingsResponse struct {
	SalesQuantityRankings []ProductRankingItem `json:"sales_quantity_rankings"` // 销量排行 TOP 10
	SalesAmountRankings   []ProductRankingItem `json:"sales_amount_rankings"`   // 销售额排行 TOP 10
	RefundRankings        []ProductRankingItem `json:"refund_rankings"`         // 退款排行 TOP 10
} // @name AnalyticsProductRankingsResponse

// ProductListRequest represents pagination and filter parameters for product analysis table.
type ProductListRequest struct {
	Dimension string `json:"dimension" example:"product"` // "product" 或 "sku"，默认 "product"
	TimeRange string `json:"time_range" example:"last_7_days"`
	StartTime string `json:"start_time" example:"2026-09-01 00:00:00"`
	EndTime   string `json:"end_time" example:"2026-09-17 23:59:59"`
	Keyword   string `json:"keyword" example:"T恤"`
	ProductID *int64 `json:"product_id" example:"1"`
	SKUCode   string `json:"sku_code" example:"TSHIRT-BLK-L"`
	Page      int    `json:"page" example:"1"`
	PageSize  int    `json:"page_size" example:"10"`
	SortBy    string `json:"sort_by" example:"sales_amount"` // sales_amount, sales_quantity, refund_amount, refund_quantity, paid_order_count
	SortOrder string `json:"sort_order" example:"desc"`      // desc, asc
} // @name AnalyticsProductListRequest

// ProductAnalysisRow represents a product or SKU row with comprehensive analytics.
type ProductAnalysisRow struct {
	ProductID      int64  `json:"product_id" example:"1"`
	ProductName    string `json:"product_name" example:"纯棉圆领T恤"`
	SKUID          int64  `json:"sku_id,omitempty" example:"10"`
	SKUCode        string `json:"sku_code,omitempty" example:"TSHIRT-BLK-L"`
	SKUText        string `json:"sku_text,omitempty" example:"黑色 / L"`
	ImageURL       string `json:"image_url" example:"https://example.com/p1.jpg"`
	SalesQuantity  int64  `json:"sales_quantity" example:"120"`
	SalesAmount    string `json:"sales_amount" example:"11880.00"`
	PaidOrderCount int64  `json:"paid_order_count" example:"80"`
	BuyerCount     int64  `json:"buyer_count" example:"75"`
	RefundQuantity int64  `json:"refund_quantity" example:"4"`
	RefundAmount   string `json:"refund_amount" example:"396.00"`
	RefundRate     string `json:"refund_rate" example:"3.33%"`
} // @name AnalyticsProductAnalysisRow

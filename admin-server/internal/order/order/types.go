package order

import "time"

// OrderListRequest represents admin query filters for orders.
type OrderListRequest struct {
	Page      int    `json:"page"`
	PageSize  int    `json:"page_size"`
	Tab       string `json:"tab"`
	Keyword   string `json:"keyword"`
	Status    *int8  `json:"status"`
	MemberID  *int64 `json:"member_id"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
} // @name OrderListRequest

// OrderTabCountsResponse represents counts for order status tabs.
type OrderTabCountsResponse struct {
	All             int64 `json:"all"`
	PendingPayment  int64 `json:"pending_payment"`
	PendingShipment int64 `json:"pending_shipment"`
	PendingReceipt  int64 `json:"pending_receipt"`
	Completed       int64 `json:"completed"`
	Cancelled       int64 `json:"cancelled"`
	Refunded        int64 `json:"refunded"`
} // @name OrderTabCountsResponse

// OrderItemRow represents an item in an order snapshot.
type OrderItemRow struct {
	ID            int64   `json:"id"`
	ProductID     int64   `json:"product_id"`
	SKUID         int64   `json:"sku_id"`
	ProductName   string  `json:"product_name"`
	SKUCode       string  `json:"sku_code"`
	SKUSpecValues *string `json:"sku_spec_values" extensions:"x-nullable"`
	ImageURL      string  `json:"image_url"`
	UnitPrice     string  `json:"unit_price"`
	Quantity      int     `json:"quantity"`
	TotalAmount   string  `json:"total_amount"`
} // @name OrderItemRow

// OrderAddressRow represents the address snapshot of an order.
type OrderAddressRow struct {
	Name         string  `json:"name"`
	Mobile       string  `json:"mobile"`
	ProvinceCode string  `json:"province_code"`
	Province     string  `json:"province"`
	CityCode     string  `json:"city_code"`
	City         string  `json:"city"`
	DistrictCode string  `json:"district_code"`
	District     string  `json:"district"`
	Address      string  `json:"address"`
	PostalCode   *string `json:"postal_code" extensions:"x-nullable"`
} // @name OrderAddressRow

// OrderLogRow represents a log entry of an order.
type OrderLogRow struct {
	ID             int64     `json:"id"`
	Action         string    `json:"action"`
	OperatorType   string    `json:"operator_type"`
	OperatorName   string    `json:"operator_name"`
	FromStatus     int8      `json:"from_status"`
	FromStatusText string    `json:"from_status_text"`
	ToStatus       int8      `json:"to_status"`
	ToStatusText   string    `json:"to_status_text"`
	Remark         string    `json:"remark"`
	CreatedAt      time.Time `json:"created_at"`
} // @name OrderLogRow

// OrderListRow represents a summary order item in admin list.
type OrderListRow struct {
	ID             int64            `json:"id"`
	OrderNo        string           `json:"order_no"`
	MemberID       int64            `json:"member_id"`
	MemberNickname string           `json:"member_nickname"`
	MemberMobile   string           `json:"member_mobile"`
	Status         int8             `json:"status"`
	StatusText     string           `json:"status_text"`
	TotalAmount    string           `json:"total_amount"`
	PayAmount      string           `json:"pay_amount"`
	FreightAmount  string           `json:"freight_amount"`
	TotalQuantity  int              `json:"total_quantity"`
	ExpressCompany *string          `json:"express_company" extensions:"x-nullable"`
	ExpressNo      *string          `json:"express_no" extensions:"x-nullable"`
	PaidAt         *time.Time       `json:"paid_at" extensions:"x-nullable"`
	ShippedAt      *time.Time       `json:"shipped_at" extensions:"x-nullable"`
	CompletedAt    *time.Time       `json:"completed_at" extensions:"x-nullable"`
	CancelledAt    *time.Time       `json:"cancelled_at" extensions:"x-nullable"`
	CreatedAt      time.Time        `json:"created_at"`
	Address        *OrderAddressRow `json:"address" extensions:"x-nullable"`
	Items          []OrderItemRow   `json:"items"`
} // @name OrderListRow

// OrderDetailRequest represents request to get an order's detail.
type OrderDetailRequest struct {
	ID int64 `json:"id" binding:"required,gt=0"`
} // @name OrderDetailRequest

// OrderDetailResponse represents full order details for admin.
type OrderDetailResponse struct {
	ID             int64            `json:"id"`
	OrderNo        string           `json:"order_no"`
	MemberID       int64            `json:"member_id"`
	MemberNickname string           `json:"member_nickname"`
	MemberMobile   string           `json:"member_mobile"`
	Status         int8             `json:"status"`
	StatusText     string           `json:"status_text"`
	TotalAmount    string           `json:"total_amount"`
	PayAmount      string           `json:"pay_amount"`
	FreightAmount  string           `json:"freight_amount"`
	DiscountAmount string           `json:"discount_amount"`
	ExpressCompany *string          `json:"express_company" extensions:"x-nullable"`
	ExpressNo      *string          `json:"express_no" extensions:"x-nullable"`
	Remark         string           `json:"remark"`
	PaidAt         *time.Time       `json:"paid_at" extensions:"x-nullable"`
	ShippedAt      *time.Time       `json:"shipped_at" extensions:"x-nullable"`
	CompletedAt    *time.Time       `json:"completed_at" extensions:"x-nullable"`
	CancelledAt    *time.Time       `json:"cancelled_at" extensions:"x-nullable"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
	Address        *OrderAddressRow `json:"address" extensions:"x-nullable"`
	Items          []OrderItemRow   `json:"items"`
	Logs           []OrderLogRow    `json:"logs"`
} // @name OrderDetailResponse

// OrderShipRequest represents shipping request from admin.
type OrderShipRequest struct {
	ID             int64  `json:"id" binding:"required,gt=0"`
	ExpressCompany string `json:"express_company" binding:"required,max=64"`
	ExpressNo      string `json:"express_no" binding:"required,max=64"`
} // @name OrderShipRequest

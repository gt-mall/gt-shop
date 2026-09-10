package order

import "time"

// CheckoutItemRequest represents an item in checkout preview or creation.
type CheckoutItemRequest struct {
	SKUID    int64 `json:"sku_id" binding:"required,gt=0"`
	Quantity int   `json:"quantity" binding:"required,gt=0"`
} // @name MiniappCheckoutItemRequest

// CheckoutPreviewRequest represents the parameters for checkout preview.
type CheckoutPreviewRequest struct {
	AddressID *int64                `json:"address_id"`
	Items     []CheckoutItemRequest `json:"items" binding:"required,min=1"`
} // @name MiniappCheckoutPreviewRequest

// CheckoutAddressResponse represents address snapshot in preview.
type CheckoutAddressResponse struct {
	ID           int64   `json:"id"`
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
	IsDefault    int8    `json:"is_default"`
} // @name MiniappCheckoutAddressResponse

// CheckoutItemResponse represents an item snapshot in preview.
type CheckoutItemResponse struct {
	SKUID         int64  `json:"sku_id"`
	ProductID     int64  `json:"product_id"`
	ProductName   string `json:"product_name"`
	SKUCode       string `json:"sku_code"`
	SKUSpecValues string `json:"sku_spec_values"`
	ImageURL      string `json:"image_url"`
	UnitPrice     string `json:"unit_price"`
	Quantity      int    `json:"quantity"`
	TotalAmount   string `json:"total_amount"`
	Stock         int    `json:"stock"`
} // @name MiniappCheckoutItemResponse

// CheckoutPreviewResponse represents the preview calculations.
type CheckoutPreviewResponse struct {
	Address        *CheckoutAddressResponse `json:"address" extensions:"x-nullable"`
	Items          []CheckoutItemResponse   `json:"items"`
	TotalQuantity  int                      `json:"total_quantity"`
	TotalAmount    string                   `json:"total_amount"`
	FreightAmount  string                   `json:"freight_amount"`
	DiscountAmount string                   `json:"discount_amount"`
	PayAmount      string                   `json:"pay_amount"`
} // @name MiniappCheckoutPreviewResponse

// OrderCreateRequest represents the request to submit an order.
type OrderCreateRequest struct {
	RequestID string                `json:"request_id" binding:"required,max=64"`
	AddressID int64                 `json:"address_id" binding:"required,gt=0"`
	Items     []CheckoutItemRequest `json:"items" binding:"required,min=1"`
	Remark    string                `json:"remark" binding:"max=255"`
} // @name MiniappOrderCreateRequest

// OrderItemResponse represents snapshot item in an order.
type OrderItemResponse struct {
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
} // @name MiniappOrderItemResponse

// OrderAddressResponse represents the address snapshot of an order.
type OrderAddressResponse struct {
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
} // @name MiniappOrderAddressResponse

// OrderLogResponse represents a record in the order log.
type OrderLogResponse struct {
	Action       string    `json:"action"`
	OperatorType string    `json:"operator_type"`
	OperatorName string    `json:"operator_name"`
	Remark       string    `json:"remark"`
	CreatedAt    time.Time `json:"created_at"`
} // @name MiniappOrderLogResponse

// OrderDetailResponse represents full details of an order.
type OrderDetailResponse struct {
	ID             int64                 `json:"id"`
	OrderNo        string                `json:"order_no"`
	Status         int8                  `json:"status"`
	StatusText     string                `json:"status_text"`
	TotalAmount    string                `json:"total_amount"`
	PayAmount      string                `json:"pay_amount"`
	FreightAmount  string                `json:"freight_amount"`
	DiscountAmount string                `json:"discount_amount"`
	ExpressCompany *string               `json:"express_company" extensions:"x-nullable"`
	ExpressNo      *string               `json:"express_no" extensions:"x-nullable"`
	Remark         string                `json:"remark"`
	PaidAt         *time.Time            `json:"paid_at" extensions:"x-nullable"`
	ShippedAt      *time.Time            `json:"shipped_at" extensions:"x-nullable"`
	CompletedAt    *time.Time            `json:"completed_at" extensions:"x-nullable"`
	CancelledAt    *time.Time            `json:"cancelled_at" extensions:"x-nullable"`
	CreatedAt      time.Time             `json:"created_at"`
	Address        *OrderAddressResponse `json:"address" extensions:"x-nullable"`
	Items          []OrderItemResponse   `json:"items"`
	Logs           []OrderLogResponse    `json:"logs,omitempty"`
} // @name MiniappOrderDetailResponse

// OrderListItemResponse represents a summarized order for order list.
type OrderListItemResponse struct {
	ID             int64               `json:"id"`
	OrderNo        string              `json:"order_no"`
	Status         int8                `json:"status"`
	StatusText     string              `json:"status_text"`
	TotalAmount    string              `json:"total_amount"`
	PayAmount      string              `json:"pay_amount"`
	FreightAmount  string              `json:"freight_amount"`
	TotalQuantity  int                 `json:"total_quantity"`
	CreatedAt      time.Time           `json:"created_at"`
	ExpressCompany *string             `json:"express_company" extensions:"x-nullable"`
	ExpressNo      *string             `json:"express_no" extensions:"x-nullable"`
	Items          []OrderItemResponse `json:"items"`
} // @name MiniappOrderListItemResponse

// OrderListRequest represents query params for member's order list.
type OrderListRequest struct {
	Page     int   `form:"page" json:"page"`
	PageSize int   `form:"page_size" json:"page_size"`
	Status   *int8 `form:"status" json:"status"`
} // @name MiniappOrderListRequest

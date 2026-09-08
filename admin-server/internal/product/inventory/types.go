package inventory

import "time"

const (
	ErrCodeInvalidArgument     = "INVALID_ARGUMENT"
	ErrCodeNotFound            = "NOT_FOUND"
	ErrCodeNotInitialized      = "NOT_INITIALIZED"
	ErrCodeVersionConflict     = "VERSION_CONFLICT"
	ErrCodeInsufficientStock   = "INSUFFICIENT_STOCK"
	ErrCodeNoChange            = "NO_CHANGE"
	ErrCodeIdempotencyConflict = "IDEMPOTENCY_CONFLICT"
	ErrCodeRetryable           = "RETRYABLE"
	ErrCodeResultUnknown       = "RESULT_UNKNOWN"
)

const (
	ChangeTypeManualIn   = "manual_in"
	ChangeTypeManualOut  = "manual_out"
	ChangeTypeStocktake  = "stocktake"
	ChangeTypeWarningSet = "warning_set"
)

type BusinessError struct {
	Code    string
	Message string
}

func (e *BusinessError) Error() string {
	return e.Message
}

func NewBusinessError(code string, message string) *BusinessError {
	return &BusinessError{Code: code, Message: message}
}

type ProductInventorySpecOption struct {
	SpecID    int64  `json:"spec_id"`
	SpecName  string `json:"spec_name"`
	ValueID   int64  `json:"value_id"`
	ValueName string `json:"value_name"`
} // @name ProductInventorySpecOption

type ProductInventoryRow struct {
	InventoryID    *int64                       `json:"inventory_id" extensions:"x-nullable"`
	ProductID      int64                        `json:"product_id"`
	SKUID          int64                        `json:"sku_id"`
	ProductName    string                       `json:"product_name"`
	SKUCode        *string                      `json:"sku_code" extensions:"x-nullable"`
	Barcode        *string                      `json:"barcode" extensions:"x-nullable"`
	Unit           *string                      `json:"unit" extensions:"x-nullable"`
	Specs          []ProductInventorySpecOption `json:"specs"`
	ImageURL       string                       `json:"image_url"`
	ProductStatus  int8                         `json:"product_status"`
	SKUStatus      int8                         `json:"sku_status"`
	Initialized    bool                         `json:"initialized"`
	Stock          *int                         `json:"stock" extensions:"x-nullable"`
	LockedStock    *int                         `json:"locked_stock" extensions:"x-nullable"`
	AvailableStock *int                         `json:"available_stock" extensions:"x-nullable"`
	WarningStock   *int                         `json:"warning_stock" extensions:"x-nullable"`
	Version        *uint32                      `json:"version" extensions:"x-nullable"`
	IsLowStock     *bool                        `json:"is_low_stock" extensions:"x-nullable"`
	IsOutOfStock   *bool                        `json:"is_out_of_stock" extensions:"x-nullable"`
	UpdatedAt      *time.Time                   `json:"updated_at" extensions:"x-nullable"`
} // @name ProductInventoryRow

type ProductInventoryListRequest struct {
	Page          int     `json:"page"`
	PageSize      int     `json:"page_size"`
	Keyword       string  `json:"keyword"`
	ProductID     *int64  `json:"product_id"`
	SKUID         *int64  `json:"sku_id"`
	ProductStatus *int8   `json:"product_status"`
	SKUStatus     *int8   `json:"sku_status"`
	StockFilter   string  `json:"stock_filter"` // all, low, empty, normal, uninitialized
} // @name ProductInventoryListRequest

type ProductInventoryDetailRequest struct {
	SKUID int64 `json:"sku_id" binding:"required"`
} // @name ProductInventoryDetailRequest

type ProductInventoryAdjustRequest struct {
	SKUID           int64   `json:"sku_id" binding:"required"`
	RequestID       string  `json:"request_id" binding:"required"`
	ExpectedVersion *uint32 `json:"expected_version"`
	ChangeType      string  `json:"change_type" binding:"required"`
	Quantity        *int    `json:"quantity"`
	Remark          string  `json:"remark"`
} // @name ProductInventoryAdjustRequest

type ProductInventoryWarningRequest struct {
	SKUID           int64   `json:"sku_id" binding:"required"`
	RequestID       string  `json:"request_id" binding:"required"`
	ExpectedVersion *uint32 `json:"expected_version"`
	Quantity        *int    `json:"quantity"`
	Remark          string  `json:"remark"`
} // @name ProductInventoryWarningRequest

type ProductInventorySKUSnapshot struct {
	ProductName string                       `json:"product_name"`
	SKUCode     *string                      `json:"sku_code" extensions:"x-nullable"`
	Unit        *string                      `json:"unit" extensions:"x-nullable"`
	Specs       []ProductInventorySpecOption `json:"specs"`
} // @name ProductInventorySKUSnapshot

type ProductInventoryLogRow struct {
	ID              int64                       `json:"id"`
	InventoryID     int64                       `json:"inventory_id"`
	ProductID       int64                       `json:"product_id"`
	SKUID           int64                       `json:"sku_id"`
	RequestID       string                      `json:"request_id"`
	ChangeType      string                      `json:"change_type"`
	Quantity        int                         `json:"quantity"`
	StockBefore     int                         `json:"stock_before"`
	StockDelta      int                         `json:"stock_delta"`
	StockAfter      int                         `json:"stock_after"`
	LockedBefore    int                         `json:"locked_before"`
	LockedDelta     int                         `json:"locked_delta"`
	LockedAfter     int                         `json:"locked_after"`
	AvailableBefore int                         `json:"available_before"`
	AvailableAfter  int                         `json:"available_after"`
	WarningBefore   int                         `json:"warning_before"`
	WarningAfter    int                         `json:"warning_after"`
	VersionBefore   uint32                      `json:"version_before"`
	VersionAfter    uint32                      `json:"version_after"`
	SKUSnapshot     ProductInventorySKUSnapshot `json:"sku_snapshot"`
	OperatorID      int64                       `json:"operator_id"`
	OperatorName    string                      `json:"operator_name"`
	Remark          string                      `json:"remark"`
	CreatedAt       time.Time                   `json:"created_at"`
} // @name ProductInventoryLogRow

type ProductInventoryMutationResult struct {
	ErrorCode *string                 `json:"error_code,omitempty" extensions:"x-nullable"`
	Log       *ProductInventoryLogRow `json:"log,omitempty" extensions:"x-nullable"`
	Replayed  bool                    `json:"replayed"`
} // @name ProductInventoryMutationResult

type ProductInventoryLogListRequest struct {
	Page        int     `json:"page"`
	PageSize    int     `json:"page_size"`
	ProductID   *int64  `json:"product_id"`
	SKUID       *int64  `json:"sku_id"`
	RequestID   *string `json:"request_id"`
	ChangeType  *string `json:"change_type"`
	OperatorID  *int64  `json:"operator_id"`
	CreatedFrom *string `json:"created_from"` // RFC3339
	CreatedTo   *string `json:"created_to"`   // RFC3339
} // @name ProductInventoryLogListRequest

type IdempotencyPayload struct {
	OperatorID      int64  `json:"operator_id"`
	SKUID           int64  `json:"sku_id"`
	ChangeType      string `json:"change_type"`
	Quantity        int    `json:"quantity"`
	ExpectedVersion uint32 `json:"expected_version"`
	Remark          string `json:"remark"`
}

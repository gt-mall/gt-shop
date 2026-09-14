package aftersale

import (
	"time"
)

// AdminAfterSaleListRequest represents the query parameters for filtering after-sales.
type AdminAfterSaleListRequest struct {
	AfterSaleNo string `json:"after_sale_no"`
	OrderNo     string `json:"order_no"`
	Keyword     string `json:"keyword"`
	Type        *int8  `json:"type"`
	Status      *int8  `json:"status"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	Page        int    `json:"page"`
	PageSize    int    `json:"page_size"`
} // @name AdminAfterSaleListRequest

// AdminAfterSaleListRow represents an after-sale entry in the admin list.
type AdminAfterSaleListRow struct {
	ID             int64     `json:"id"`
	AfterSaleNo    string    `json:"after_sale_no"`
	OrderID        int64     `json:"order_id"`
	OrderNo        string    `json:"order_no"`
	MemberID       int64     `json:"member_id"`
	MemberNickname string    `json:"member_nickname"`
	MemberMobile   string    `json:"member_mobile"`
	Type           int8      `json:"type"`
	TypeText       string    `json:"type_text"`
	Status         int8      `json:"status"`
	StatusText     string    `json:"status_text"`
	RefundAmount   string    `json:"refund_amount"`
	Reason         string    `json:"reason"`
	AppliedAt      time.Time `json:"applied_at"`
	CreatedAt      time.Time `json:"created_at"`
} // @name AdminAfterSaleListRow

// AdminAfterSaleDetailRequest represents the request to fetch an after-sale detail.
type AdminAfterSaleDetailRequest struct {
	ID int64 `json:"id" binding:"required"`
} // @name AdminAfterSaleDetailRequest

// AfterSaleItemRow represents a snapshot item of an after-sale.
type AfterSaleItemRow struct {
	ID              int64  `json:"id"`
	AfterSaleID     int64  `json:"after_sale_id"`
	OrderItemID     int64  `json:"order_item_id"`
	ProductID       int64  `json:"product_id"`
	SKUID           int64  `json:"sku_id"`
	Quantity        int    `json:"quantity"`
	RefundAmount    string `json:"refund_amount"`
	ProductName     string `json:"product_name"`
	SKUText         string `json:"sku_text"`
	SKUCode         string `json:"sku_code"`
	SKUImageURL     string `json:"sku_image_url"`
	RestockQuantity int    `json:"restock_quantity"`
} // @name AfterSaleItemRow

// AfterSaleLogRow represents an audit log entry of an after-sale.
type AfterSaleLogRow struct {
	ID               int64     `json:"id"`
	OperatorType     int8      `json:"operator_type"`
	OperatorTypeText string    `json:"operator_type_text"`
	OperatorID       int64     `json:"operator_id"`
	OperatorName     string    `json:"operator_name"`
	FromStatus       int8      `json:"from_status"`
	FromStatusText   string    `json:"from_status_text"`
	ToStatus         int8      `json:"to_status"`
	ToStatusText     string    `json:"to_status_text"`
	Action           string    `json:"action"`
	Remark           string    `json:"remark"`
	CreatedAt        time.Time `json:"created_at"`
} // @name AfterSaleLogRow

// OrderRefundRow represents refund transaction details for an after-sale.
type OrderRefundRow struct {
	ID               int64      `json:"id"`
	RefundNo         string     `json:"refund_no"`
	RefundAmount     string     `json:"refund_amount"`
	Status           int8       `json:"status"`
	StatusText       string     `json:"status_text"`
	Provider         string     `json:"provider"`
	ProviderRefundNo *string    `json:"provider_refund_no" extensions:"x-nullable"`
	RetryCount       int        `json:"retry_count"`
	LastError        string     `json:"last_error"`
	RequestedAt      *time.Time `json:"requested_at" extensions:"x-nullable"`
	RefundedAt       *time.Time `json:"refunded_at" extensions:"x-nullable"`
} // @name OrderRefundRow

// AfterSaleProofFileResponse represents detailed metadata for a proof file in after-sale detail.
type AfterSaleProofFileResponse struct {
	ID           int64  `json:"id"`
	FileType     string `json:"file_type"`
	OriginalName string `json:"original_name"`
	MIMEType     string `json:"mime_type"`
	Size         int64  `json:"size"`
	URL          string `json:"url"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	DurationMS   int64  `json:"duration_ms"`
	CoverURL     string `json:"cover_url"`
} // @name AfterSaleProofFile

// AdminAfterSaleDetailResponse represents the full after-sale details for admin.
type AdminAfterSaleDetailResponse struct {
	ID                     int64              `json:"id"`
	AfterSaleNo            string             `json:"after_sale_no"`
	RequestID              string             `json:"request_id"`
	OrderID                int64              `json:"order_id"`
	OrderNo                string             `json:"order_no"`
	MemberID               int64              `json:"member_id"`
	MemberNickname         string             `json:"member_nickname"`
	MemberMobile           string             `json:"member_mobile"`
	Type                   int8               `json:"type"`
	TypeText               string             `json:"type_text"`
	Status                 int8               `json:"status"`
	StatusText             string             `json:"status_text"`
	OrderStatusAtApply     int8               `json:"order_status_at_apply"`
	OrderStatusAtApplyText string             `json:"order_status_at_apply_text"`
	Reason                 string             `json:"reason"`
	Description            string             `json:"description"`
	ProofFileIDs           []int64                      `json:"proof_file_ids"`
	ProofURLs              []string                     `json:"proof_urls"`
	ProofFiles             []AfterSaleProofFileResponse `json:"proof_files"`
	RefundAmount           string                       `json:"refund_amount"`
	ReturnReceiver         string             `json:"return_receiver"`
	ReturnPhone            string             `json:"return_phone"`
	ReturnAddress          string             `json:"return_address"`
	BuyerReturnCompany     string             `json:"buyer_return_company"`
	BuyerReturnNo          string             `json:"buyer_return_no"`
	RejectReason           string             `json:"reject_reason"`
	AppliedAt              time.Time          `json:"applied_at"`
	AuditedAt              *time.Time         `json:"audited_at" extensions:"x-nullable"`
	BuyerReturnedAt        *time.Time         `json:"buyer_returned_at" extensions:"x-nullable"`
	MerchantReceivedAt     *time.Time         `json:"merchant_received_at" extensions:"x-nullable"`
	RefundStartedAt        *time.Time         `json:"refund_started_at" extensions:"x-nullable"`
	CompletedAt            *time.Time         `json:"completed_at" extensions:"x-nullable"`
	CancelledAt            *time.Time         `json:"cancelled_at" extensions:"x-nullable"`
	CreatedAt              time.Time          `json:"created_at"`
	UpdatedAt              time.Time          `json:"updated_at"`
	Items                  []AfterSaleItemRow `json:"items"`
	Logs                   []AfterSaleLogRow  `json:"logs"`
	Refund                 *OrderRefundRow    `json:"refund" extensions:"x-nullable"`
} // @name AdminAfterSaleDetailResponse

// AdminApproveRequest represents the request for approving an after-sale.
type AdminApproveRequest struct {
	ID             int64  `json:"id" binding:"required"`
	ReturnReceiver string `json:"return_receiver"`
	ReturnPhone    string `json:"return_phone"`
	ReturnAddress  string `json:"return_address"`
} // @name AdminApproveRequest

// AdminRejectRequest represents the request for rejecting an after-sale.
type AdminRejectRequest struct {
	ID           int64  `json:"id" binding:"required"`
	RejectReason string `json:"reject_reason" binding:"required"`
} // @name AdminRejectRequest

// AdminConfirmReceiveItem represents the restock quantity for an item.
type AdminConfirmReceiveItem struct {
	AfterSaleItemID int64 `json:"after_sale_item_id" binding:"required"`
	RestockQuantity int   `json:"restock_quantity"`
} // @name AdminConfirmReceiveItem

// AdminConfirmReceiveRequest represents the request for confirming return shipment receipt.
type AdminConfirmReceiveRequest struct {
	ID    int64                     `json:"id" binding:"required"`
	Items []AdminConfirmReceiveItem `json:"items" binding:"required"`
} // @name AdminConfirmReceiveRequest

// AdminRetryRefundRequest represents the request for retrying a failed refund.
type AdminRetryRefundRequest struct {
	ID int64 `json:"id" binding:"required"`
} // @name AdminRetryRefundRequest

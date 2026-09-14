package aftersale

import (
	"time"
)

// MiniappApplyAfterSaleRequest represents the parameters to apply for an after-sale.
type MiniappApplyAfterSaleRequest struct {
	OrderID      int64   `json:"order_id" binding:"required"`
	Type         int8    `json:"type" binding:"required"`
	Reason       string  `json:"reason" binding:"required"`
	Description  string  `json:"description"`
	ProofFileIDs []int64 `json:"proof_file_ids"`
	RequestID    string  `json:"request_id" binding:"required"`
} // @name MiniappApplyAfterSaleRequest

// MiniappCancelAfterSaleRequest represents the parameters to cancel an after-sale.
type MiniappCancelAfterSaleRequest struct {
	AfterSaleID int64 `json:"after_sale_id" binding:"required"`
} // @name MiniappCancelAfterSaleRequest

// MiniappReturnLogisticsRequest represents the return shipping information filled by buyer.
type MiniappReturnLogisticsRequest struct {
	AfterSaleID int64  `json:"after_sale_id" binding:"required"`
	Company     string `json:"company" binding:"required"`
	TrackingNo  string `json:"tracking_no" binding:"required"`
} // @name MiniappReturnLogisticsRequest

// MiniappAfterSaleListRequest represents the pagination and filter options for miniapp.
type MiniappAfterSaleListRequest struct {
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Status   *int8 `json:"status"`
} // @name MiniappAfterSaleListRequest

// MiniappAfterSaleDetailRequest represents the request for viewing after-sale detail.
type MiniappAfterSaleDetailRequest struct {
	AfterSaleID int64 `json:"after_sale_id" binding:"required"`
} // @name MiniappAfterSaleDetailRequest

// MiniappAfterSaleItemResponse represents the item info in miniapp responses.
type MiniappAfterSaleItemResponse struct {
	ID           int64  `json:"id"`
	ProductID    int64  `json:"product_id"`
	SKUID        int64  `json:"sku_id"`
	Quantity     int    `json:"quantity"`
	RefundAmount string `json:"refund_amount"`
	ProductName  string `json:"product_name"`
	SKUText      string `json:"sku_text"`
	SKUCode      string `json:"sku_code"`
	SKUImageURL  string `json:"sku_image_url"`
} // @name MiniappAfterSaleItemResponse

// MiniappAfterSaleLogRow represents a customer-facing log entry.
type MiniappAfterSaleLogRow struct {
	Action         string    `json:"action"`
	OperatorType   string    `json:"operator_type"`
	FromStatusText string    `json:"from_status_text"`
	ToStatusText   string    `json:"to_status_text"`
	Remark         string    `json:"remark"`
	CreatedAt      time.Time `json:"created_at"`
} // @name MiniappAfterSaleLogRow

// MiniappAfterSaleRefundInfo represents refund status for miniapp view.
type MiniappAfterSaleRefundInfo struct {
	Status       int8       `json:"status"`
	StatusText   string     `json:"status_text"`
	RefundAmount string     `json:"refund_amount"`
	RefundedAt   *time.Time `json:"refunded_at" extensions:"x-nullable"`
} // @name MiniappAfterSaleRefundInfo

// MiniappAfterSaleListItemResponse represents an after-sale item in the member's list.
type MiniappAfterSaleListItemResponse struct {
	ID           int64                          `json:"id"`
	AfterSaleNo  string                         `json:"after_sale_no"`
	OrderID      int64                          `json:"order_id"`
	Type         int8                           `json:"type"`
	TypeText     string                         `json:"type_text"`
	Status       int8                           `json:"status"`
	StatusText   string                         `json:"status_text"`
	RefundAmount string                         `json:"refund_amount"`
	Reason       string                         `json:"reason"`
	CreatedAt    time.Time                      `json:"created_at"`
	Items        []MiniappAfterSaleItemResponse `json:"items"`
} // @name MiniappAfterSaleListItemResponse

// MiniappAfterSaleDetailResponse represents the complete detail of an after-sale for the member.
type MiniappAfterSaleDetailResponse struct {
	ID                 int64                          `json:"id"`
	AfterSaleNo        string                         `json:"after_sale_no"`
	OrderID            int64                          `json:"order_id"`
	Type               int8                           `json:"type"`
	TypeText           string                         `json:"type_text"`
	Status             int8                           `json:"status"`
	StatusText         string                         `json:"status_text"`
	Reason             string                         `json:"reason"`
	Description        string                         `json:"description"`
	ProofURLs          []string                       `json:"proof_urls"`
	ProofFiles         []MiniappAfterSaleProofFile    `json:"proof_files"`
	RefundAmount       string                         `json:"refund_amount"`
	ReturnReceiver     string                         `json:"return_receiver"`
	ReturnPhone        string                         `json:"return_phone"`
	ReturnAddress      string                         `json:"return_address"`
	BuyerReturnCompany string                         `json:"buyer_return_company"`
	BuyerReturnNo      string                         `json:"buyer_return_no"`
	RejectReason       string                         `json:"reject_reason"`
	AppliedAt          time.Time                      `json:"applied_at"`
	AuditedAt          *time.Time                     `json:"audited_at" extensions:"x-nullable"`
	BuyerReturnedAt    *time.Time                     `json:"buyer_returned_at" extensions:"x-nullable"`
	MerchantReceivedAt *time.Time                     `json:"merchant_received_at" extensions:"x-nullable"`
	CompletedAt        *time.Time                     `json:"completed_at" extensions:"x-nullable"`
	CancelledAt        *time.Time                     `json:"cancelled_at" extensions:"x-nullable"`
	CreatedAt          time.Time                      `json:"created_at"`
	Items              []MiniappAfterSaleItemResponse `json:"items"`
	Refund             *MiniappAfterSaleRefundInfo    `json:"refund" extensions:"x-nullable"`
	Logs               []MiniappAfterSaleLogRow       `json:"logs"`
} // @name MiniappAfterSaleDetailResponse

// MiniappAfterSaleProofUploadResponse represents the upload response for an after-sale proof file.
type MiniappAfterSaleProofUploadResponse struct {
	ID           int64  `json:"id"`
	FileType     string `json:"file_type"`
	OriginalName string `json:"original_name"`
	MIMEType     string `json:"mime_type"`
	Size         int64  `json:"size"`
	URL          string `json:"url"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	DurationMS   int64  `json:"duration_ms"`
} // @name MiniappAfterSaleProofUploadResponse

// MiniappAfterSaleProofFile represents a proof file attachment in detail view.
type MiniappAfterSaleProofFile struct {
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
} // @name MiniappAfterSaleProofFile

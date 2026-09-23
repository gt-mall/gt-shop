package model

import (
	"time"
)

const (
	// DefaultAfterSaleDays specifies the maximum days after order completion that an after-sale can be requested.
	DefaultAfterSaleDays = 7

	// After-sale types
	AfterSaleTypeRefundOnly   int8 = 1 // 仅退款
	AfterSaleTypeReturnRefund int8 = 2 // 退货退款

	// After-sale statuses
	AfterSaleStatusPendingReview          int8 = 10 // 待审核
	AfterSaleStatusWaitingBuyerReturn     int8 = 20 // 待买家退货
	AfterSaleStatusWaitingMerchantReceive int8 = 30 // 待商家收货
	AfterSaleStatusRefunding              int8 = 40 // 退款处理中
	AfterSaleStatusCompleted              int8 = 50 // 已完成
	AfterSaleStatusRejected               int8 = 60 // 已拒绝
	AfterSaleStatusCancelled              int8 = 70 // 已取消
	AfterSaleStatusRefundFailed           int8 = 80 // 退款失败

	// After-sale log operator types
	AfterSaleOperatorTypeMember int8 = 1 // 用户
	AfterSaleOperatorTypeAdmin  int8 = 2 // 管理员
	AfterSaleOperatorTypeSystem int8 = 3 // 系统

	// Order refund statuses
	RefundStatusPending    int8 = 10 // 待退款
	RefundStatusProcessing int8 = 20 // 退款处理中
	RefundStatusSuccess    int8 = 30 // 退款成功
	RefundStatusFailed     int8 = 40 // 退款失败
)

func AfterSaleTypeText(t int8) string {
	switch t {
	case AfterSaleTypeRefundOnly:
		return "仅退款"
	case AfterSaleTypeReturnRefund:
		return "退货退款"
	default:
		return "未知类型"
	}
}

func AfterSaleStatusText(status int8) string {
	switch status {
	case AfterSaleStatusPendingReview:
		return "待审核"
	case AfterSaleStatusWaitingBuyerReturn:
		return "待买家退货"
	case AfterSaleStatusWaitingMerchantReceive:
		return "待商家收货"
	case AfterSaleStatusRefunding:
		return "退款处理中"
	case AfterSaleStatusCompleted:
		return "已完成"
	case AfterSaleStatusRejected:
		return "已拒绝"
	case AfterSaleStatusCancelled:
		return "已取消"
	case AfterSaleStatusRefundFailed:
		return "退款失败"
	default:
		return "未知状态"
	}
}

func AfterSaleOperatorTypeText(opType int8) string {
	switch opType {
	case AfterSaleOperatorTypeMember:
		return "用户"
	case AfterSaleOperatorTypeAdmin:
		return "管理员"
	case AfterSaleOperatorTypeSystem:
		return "系统"
	default:
		return "未知操作人"
	}
}

func RefundStatusText(status int8) string {
	switch status {
	case RefundStatusPending:
		return "待退款"
	case RefundStatusProcessing:
		return "退款处理中"
	case RefundStatusSuccess:
		return "退款成功"
	case RefundStatusFailed:
		return "退款失败"
	default:
		return "未知状态"
	}
}

// IsActiveAfterSaleStatus returns whether the after-sale status indicates an active/in-progress after-sale.
func IsActiveAfterSaleStatus(status int8) bool {
	switch status {
	case AfterSaleStatusPendingReview,
		AfterSaleStatusWaitingBuyerReturn,
		AfterSaleStatusWaitingMerchantReceive,
		AfterSaleStatusRefunding,
		AfterSaleStatusRefundFailed:
		return true
	default:
		return false
	}
}

type AfterSale struct {
	ID                 int64      `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	AfterSaleNo        string     `gorm:"column:after_sale_no;size:64;not null;uniqueIndex:uk_after_sale_no" json:"after_sale_no"`
	RequestID          string     `gorm:"column:request_id;size:64;not null;uniqueIndex:uk_after_sale_member_request,priority:2" json:"request_id"`
	OrderID            int64      `gorm:"column:order_id;not null;index:idx_after_sale_order_id" json:"order_id"`
	MemberID           int64      `gorm:"column:member_id;not null;uniqueIndex:uk_after_sale_member_request,priority:1;index:idx_after_sale_member_status,priority:1" json:"member_id"`
	Type               int8       `gorm:"column:type;not null" json:"type"`
	Status             int8       `gorm:"column:status;not null;default:10;index:idx_after_sale_member_status,priority:2;index:idx_after_sale_status_created,priority:1" json:"status"`
	OrderStatusAtApply int8       `gorm:"column:order_status_at_apply;not null" json:"order_status_at_apply"`
	Reason             string     `gorm:"column:reason;size:100;not null;default:''" json:"reason"`
	Description        string     `gorm:"column:description;size:1000;not null;default:''" json:"description"`
	ProofFileIDs       *string    `gorm:"column:proof_file_ids;type:json;default:null" json:"proof_file_ids"`
	RefundAmount       string     `gorm:"column:refund_amount;type:decimal(12,2);not null;default:0.00" json:"refund_amount"`
	ReturnReceiver     string     `gorm:"column:return_receiver;size:255;not null;default:''" json:"return_receiver"`
	ReturnPhone        string     `gorm:"column:return_phone;size:64;not null;default:''" json:"return_phone"`
	ReturnAddress      string     `gorm:"column:return_address;size:500;not null;default:''" json:"return_address"`
	BuyerReturnCompany string     `gorm:"column:buyer_return_company;size:64;not null;default:''" json:"buyer_return_company"`
	BuyerReturnNo      string     `gorm:"column:buyer_return_no;size:64;not null;default:''" json:"buyer_return_no"`
	RejectReason       string     `gorm:"column:reject_reason;size:255;not null;default:''" json:"reject_reason"`
	AppliedAt          time.Time  `gorm:"column:applied_at;not null" json:"applied_at"`
	AuditedAt          *time.Time `gorm:"column:audited_at;default:null" json:"audited_at"`
	BuyerReturnedAt    *time.Time `gorm:"column:buyer_returned_at;default:null" json:"buyer_returned_at"`
	MerchantReceivedAt *time.Time `gorm:"column:merchant_received_at;default:null" json:"merchant_received_at"`
	RefundStartedAt    *time.Time `gorm:"column:refund_started_at;default:null" json:"refund_started_at"`
	CompletedAt        *time.Time `gorm:"column:completed_at;default:null" json:"completed_at"`
	CancelledAt        *time.Time `gorm:"column:cancelled_at;default:null" json:"cancelled_at"`
	CreatedAt          time.Time  `gorm:"column:created_at;index:idx_after_sale_status_created,priority:2" json:"created_at"`
	UpdatedAt          time.Time  `gorm:"column:updated_at" json:"updated_at"`
}

func (AfterSale) TableName() string { return "after_sale" }

type AfterSaleItem struct {
	ID              int64     `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	AfterSaleID     int64     `gorm:"column:after_sale_id;not null;index:idx_after_sale_item_after_sale_id" json:"after_sale_id"`
	OrderItemID     int64     `gorm:"column:order_item_id;not null;index:idx_after_sale_item_order_item_id" json:"order_item_id"`
	ProductID       int64     `gorm:"column:product_id;not null" json:"product_id"`
	SKUID           int64     `gorm:"column:sku_id;not null" json:"sku_id"`
	Quantity        int       `gorm:"column:quantity;not null;default:1" json:"quantity"`
	RefundAmount    string    `gorm:"column:refund_amount;type:decimal(12,2);not null;default:0.00" json:"refund_amount"`
	ProductName     string    `gorm:"column:product_name;size:255;not null;default:''" json:"product_name"`
	SKUText         string    `gorm:"column:sku_text;size:512;not null;default:''" json:"sku_text"`
	SKUCode         string    `gorm:"column:sku_code;size:255;not null;default:''" json:"sku_code"`
	SKUImageURL     string    `gorm:"column:sku_image_url;size:1024;not null;default:''" json:"sku_image_url"`
	RestockQuantity int       `gorm:"column:restock_quantity;not null;default:0" json:"restock_quantity"`
	CreatedAt       time.Time `gorm:"column:created_at" json:"created_at"`
}

func (AfterSaleItem) TableName() string { return "after_sale_item" }

type AfterSaleLog struct {
	ID           int64     `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	AfterSaleID  int64     `gorm:"column:after_sale_id;not null;index:idx_after_sale_log_after_sale_id_id,priority:1" json:"after_sale_id"`
	OperatorType int8      `gorm:"column:operator_type;not null" json:"operator_type"`
	OperatorID   int64     `gorm:"column:operator_id;not null;default:0" json:"operator_id"`
	OperatorName string    `gorm:"column:operator_name;size:100;not null;default:''" json:"operator_name"`
	FromStatus   int8      `gorm:"column:from_status;not null;default:0" json:"from_status"`
	ToStatus     int8      `gorm:"column:to_status;not null;default:0" json:"to_status"`
	Action       string    `gorm:"column:action;size:64;not null" json:"action"`
	Remark       string    `gorm:"column:remark;size:500;not null;default:''" json:"remark"`
	CreatedAt    time.Time `gorm:"column:created_at" json:"created_at"`
}

func (AfterSaleLog) TableName() string { return "after_sale_log" }

type OrderRefund struct {
	ID               int64      `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	RefundNo         string     `gorm:"column:refund_no;size:64;not null;uniqueIndex:uk_order_refund_refund_no" json:"refund_no"`
	AfterSaleID      int64      `gorm:"column:after_sale_id;not null;uniqueIndex:uk_order_refund_after_sale_id" json:"after_sale_id"`
	OrderID          int64      `gorm:"column:order_id;not null;index:idx_order_refund_order_id" json:"order_id"`
	PaymentID        int64      `gorm:"column:payment_id;not null;default:0" json:"payment_id"`
	MemberID         int64      `gorm:"column:member_id;not null" json:"member_id"`
	RefundAmount     string     `gorm:"column:refund_amount;type:decimal(12,2);not null;default:0.00" json:"refund_amount"`
	Status           int8       `gorm:"column:status;not null;default:10" json:"status"`
	Provider         string     `gorm:"column:provider;size:32;not null;default:'wechat'" json:"provider"`
	ProviderRefundNo *string    `gorm:"column:provider_refund_no;size:128;default:null" json:"provider_refund_no"`
	RetryCount       int        `gorm:"column:retry_count;not null;default:0" json:"retry_count"`
	LastError        string     `gorm:"column:last_error;size:500;not null;default:''" json:"last_error"`
	RequestedAt      *time.Time `gorm:"column:requested_at;default:null" json:"requested_at"`
	RefundedAt       *time.Time `gorm:"column:refunded_at;default:null" json:"refunded_at"`
	CreatedAt        time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt        time.Time  `gorm:"column:updated_at" json:"updated_at"`
}

func (OrderRefund) TableName() string { return "order_refund" }

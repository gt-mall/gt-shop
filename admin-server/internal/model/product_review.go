package model

import (
	"time"

	"gorm.io/gorm"
)

// Review statuses
const (
	ReviewStatusPendingReview int8 = 10 // 待审核
	ReviewStatusApproved      int8 = 20 // 公开
	ReviewStatusRejected      int8 = 30 // 驳回
	ReviewStatusHidden        int8 = 40 // 隐藏
)

func ReviewStatusText(status int8) string {
	switch status {
	case ReviewStatusPendingReview:
		return "待审核"
	case ReviewStatusApproved:
		return "公开"
	case ReviewStatusRejected:
		return "驳回"
	case ReviewStatusHidden:
		return "隐藏"
	default:
		return "未知状态"
	}
}

// Review operator types
const (
	ReviewOperatorTypeMember int8 = 1 // 会员
	ReviewOperatorTypeAdmin  int8 = 2 // 管理员
	ReviewOperatorTypeSystem int8 = 3 // 系统
)

// Review log actions
const (
	ReviewActionCreate       = "create"
	ReviewActionAuditApprove = "audit_approve"
	ReviewActionAuditReject  = "audit_reject"
	ReviewActionHide         = "hide"
	ReviewActionRestore      = "restore"
	ReviewActionReply        = "reply"
)

// ProductReview represents the product_review table.
type ProductReview struct {
	ID            int64          `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	OrderID       int64          `gorm:"column:order_id;not null;index:idx_review_order_id" json:"order_id"`
	OrderItemID   int64          `gorm:"column:order_item_id;not null;uniqueIndex:uk_review_order_item_id" json:"order_item_id"`
	MemberID      int64          `gorm:"column:member_id;not null;uniqueIndex:uk_review_member_request,priority:1;index:idx_review_member_id" json:"member_id"`
	RequestID     string         `gorm:"column:request_id;size:64;not null;uniqueIndex:uk_review_member_request,priority:2" json:"request_id"`
	ProductID     int64          `gorm:"column:product_id;not null;index:idx_review_product_status,priority:1;index:idx_review_product_has_image,priority:1" json:"product_id"`
	SKUID         int64          `gorm:"column:sku_id;not null" json:"sku_id"`
	ProductName   string         `gorm:"column:product_name;size:255;not null;default:''" json:"product_name"`
	SKUSpecValues *string        `gorm:"column:sku_spec_values;type:json;default:null" json:"sku_spec_values"`
	Rating        int            `gorm:"column:rating;not null;default:5" json:"rating"`
	Content       string         `gorm:"column:content;size:1000;not null;default:''" json:"content"`
	IsAnonymous   int8           `gorm:"column:is_anonymous;not null;default:0" json:"is_anonymous"`
	Status        int8           `gorm:"column:status;not null;default:10;index:idx_review_product_status,priority:2;index:idx_review_product_has_image,priority:2;index:idx_review_status_created,priority:1" json:"status"`
	HasImage      int8           `gorm:"column:has_image;not null;default:0;index:idx_review_product_has_image,priority:3" json:"has_image"`
	AuditReason   string         `gorm:"column:audit_reason;size:255;not null;default:''" json:"audit_reason"`
	AuditUserID   *int64         `gorm:"column:audit_user_id;default:null" json:"audit_user_id"`
	AuditedAt     *time.Time     `gorm:"column:audited_at;default:null" json:"audited_at"`
	ReplyContent  string         `gorm:"column:reply_content;size:1000;not null;default:''" json:"reply_content"`
	ReplyUserID   *int64         `gorm:"column:reply_user_id;default:null" json:"reply_user_id"`
	ReplyAt       *time.Time     `gorm:"column:reply_at;default:null" json:"reply_at"`
	CreatedAt     time.Time      `gorm:"column:created_at;index:idx_review_product_status,priority:3;index:idx_review_status_created,priority:2;index:idx_review_member_id,priority:2" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"column:deleted_at;index:idx_review_deleted_at" json:"deleted_at,omitempty"`
}

func (ProductReview) TableName() string { return "product_review" }

// ProductReviewImage represents the product_review_image table.
type ProductReviewImage struct {
	ID        int64     `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	ReviewID  int64     `gorm:"column:review_id;not null;uniqueIndex:uk_review_file,priority:1;index:idx_review_image_review_id,priority:1" json:"review_id"`
	FileID    int64     `gorm:"column:file_id;not null;uniqueIndex:uk_review_file,priority:2;index:idx_review_image_file_id" json:"file_id"`
	Sort      int       `gorm:"column:sort;not null;default:0;index:idx_review_image_review_id,priority:2" json:"sort"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
}

func (ProductReviewImage) TableName() string { return "product_review_image" }

// ProductReviewLog represents the product_review_log table.
type ProductReviewLog struct {
	ID           int64     `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	ReviewID     int64     `gorm:"column:review_id;not null;index:idx_review_log_review_id,priority:1" json:"review_id"`
	OperatorType int8      `gorm:"column:operator_type;not null" json:"operator_type"`
	OperatorID   int64     `gorm:"column:operator_id;not null;default:0" json:"operator_id"`
	OperatorName string    `gorm:"column:operator_name;size:100;not null;default:''" json:"operator_name"`
	FromStatus   int8      `gorm:"column:from_status;not null;default:0" json:"from_status"`
	ToStatus     int8      `gorm:"column:to_status;not null;default:0" json:"to_status"`
	Action       string    `gorm:"column:action;size:64;not null" json:"action"`
	Remark       string    `gorm:"column:remark;size:500;not null;default:''" json:"remark"`
	CreatedAt    time.Time `gorm:"column:created_at;index:idx_review_log_review_id,priority:2" json:"created_at"`
}

func (ProductReviewLog) TableName() string { return "product_review_log" }

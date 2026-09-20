package review

import (
	"time"

	systemfile "mall/admin_core/internal/system/file"
)

// ReviewListRequest represents the query filter for listing reviews in admin.
type ReviewListRequest struct {
	Page      int    `json:"page"`
	PageSize  int    `json:"page_size"`
	ProductID *int64 `json:"product_id"`
	Status    *int8  `json:"status"` // 10:待审核, 20:公开, 30:驳回, 40:隐藏
	Rating    *int   `json:"rating"` // 1-5
	HasImage  *int8  `json:"has_image"` // 0:无图, 1:有图
	Keyword   string `json:"keyword"` // 搜索评价内容、商品名称
	MemberID  *int64 `json:"member_id"`
	StartDate string `json:"start_date"` // YYYY-MM-DD
	EndDate   string `json:"end_date"`   // YYYY-MM-DD
} // @name ReviewListRequest

// ReviewListItemResponse represents a review in the admin list.
type ReviewListItemResponse struct {
	ID             int64                    `json:"id"`
	OrderID        int64                    `json:"order_id"`
	OrderItemID    int64                    `json:"order_item_id"`
	MemberID       int64                    `json:"member_id"`
	MemberNickname string                   `json:"member_nickname"`
	MemberMobile   string                   `json:"member_mobile"`
	ProductID      int64                    `json:"product_id"`
	ProductName    string                   `json:"product_name"`
	SKUID          int64                    `json:"sku_id"`
	SKUSpecValues  *string                  `json:"sku_spec_values" extensions:"x-nullable"`
	Rating         int                      `json:"rating"`
	Content        string                   `json:"content"`
	IsAnonymous    int8                     `json:"is_anonymous"`
	Status         int8                     `json:"status"`
	StatusText     string                   `json:"status_text"`
	HasImage       int8                     `json:"has_image"`
	Images         []systemfile.FileResponse `json:"images"`
	AuditReason    string                   `json:"audit_reason"`
	AuditUserName  string                   `json:"audit_user_name"`
	AuditedAt      *time.Time               `json:"audited_at" extensions:"x-nullable"`
	ReplyContent   string                   `json:"reply_content"`
	ReplyUserName  string                   `json:"reply_user_name"`
	ReplyAt        *time.Time               `json:"reply_at" extensions:"x-nullable"`
	CreatedAt      time.Time                `json:"created_at"`
} // @name ReviewListItemResponse

// ReviewDetailRequest represents the request to get a review detail.
type ReviewDetailRequest struct {
	ID int64 `json:"id" binding:"required,gt=0"`
} // @name ReviewDetailRequest

// ReviewLogResponse represents an audit or action log record.
type ReviewLogResponse struct {
	ID               int64     `json:"id"`
	OperatorType     int8      `json:"operator_type"`
	OperatorTypeName string    `json:"operator_type_name"`
	OperatorID       int64     `json:"operator_id"`
	OperatorName     string    `json:"operator_name"`
	FromStatus       int8      `json:"from_status"`
	FromStatusText   string    `json:"from_status_text"`
	ToStatus         int8      `json:"to_status"`
	ToStatusText     string    `json:"to_status_text"`
	Action           string    `json:"action"`
	Remark           string    `json:"remark"`
	CreatedAt        time.Time `json:"created_at"`
} // @name ReviewLogResponse

// ReviewDetailResponse represents full review details in admin.
type ReviewDetailResponse struct {
	ID              int64                    `json:"id"`
	OrderID         int64                    `json:"order_id"`
	OrderNo         string                   `json:"order_no"`
	OrderItemID     int64                    `json:"order_item_id"`
	MemberID        int64                    `json:"member_id"`
	MemberNickname  string                   `json:"member_nickname"`
	MemberMobile    string                   `json:"member_mobile"`
	MemberAvatarURL string                   `json:"member_avatar_url"`
	ProductID       int64                    `json:"product_id"`
	ProductName     string                   `json:"product_name"`
	SKUID           int64                    `json:"sku_id"`
	SKUSpecValues   *string                  `json:"sku_spec_values" extensions:"x-nullable"`
	Rating          int                      `json:"rating"`
	Content         string                   `json:"content"`
	IsAnonymous     int8                     `json:"is_anonymous"`
	Status          int8                     `json:"status"`
	StatusText      string                   `json:"status_text"`
	HasImage        int8                     `json:"has_image"`
	Images          []systemfile.FileResponse `json:"images"`
	AuditReason     string                   `json:"audit_reason"`
	AuditUserName   string                   `json:"audit_user_name"`
	AuditedAt       *time.Time               `json:"audited_at" extensions:"x-nullable"`
	ReplyContent    string                   `json:"reply_content"`
	ReplyUserName   string                   `json:"reply_user_name"`
	ReplyAt         *time.Time               `json:"reply_at" extensions:"x-nullable"`
	CreatedAt       time.Time                `json:"created_at"`
	UpdatedAt       time.Time                `json:"updated_at"`
	Logs            []ReviewLogResponse      `json:"logs"`
} // @name ReviewDetailResponse

// ReviewAuditRequest represents the request to audit or change review status.
type ReviewAuditRequest struct {
	ID             int64  `json:"id" binding:"required,gt=0"`
	ExpectedStatus int8   `json:"expected_status" binding:"required"` // 期望旧状态（乐观锁防并发覆盖）
	Action         string `json:"action" binding:"required"`          // "approve", "reject", "hide", "restore"
	Reason         string `json:"reason"`                            // 审核原因/驳回原因
} // @name ReviewAuditRequest

// ReviewReplyRequest represents the request to reply to a review.
type ReviewReplyRequest struct {
	ID      int64  `json:"id" binding:"required,gt=0"`
	Content string `json:"content" binding:"required"` // 最多500字符
} // @name ReviewReplyRequest

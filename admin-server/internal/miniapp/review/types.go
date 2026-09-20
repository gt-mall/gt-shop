package review

import "time"

// MiniappReviewUploadResponse represents the upload response for review image.
type MiniappReviewUploadResponse struct {
	FileID int64  `json:"file_id"`
	URL    string `json:"url"`
} // @name MiniappReviewUploadResponse

// MiniappReviewCreateRequest represents parameters for submitting a product review.
type MiniappReviewCreateRequest struct {
	OrderItemID int64   `json:"order_item_id" binding:"required,gt=0"`
	Rating      int     `json:"rating" binding:"required,min=1,max=5"`
	Content     string  `json:"content" binding:"required"`
	ImageIDs    []int64 `json:"image_ids"`
	IsAnonymous int8    `json:"is_anonymous"` // 0: 实名, 1: 匿名
	RequestID   string  `json:"request_id" binding:"required,max=64"`
} // @name MiniappReviewCreateRequest

// MiniappReviewCreateResponse represents the creation response.
type MiniappReviewCreateResponse struct {
	ID         int64  `json:"id"`
	Status     int8   `json:"status"`
	StatusText string `json:"status_text"`
} // @name MiniappReviewCreateResponse

// MiniappMineReviewItemResponse represents a review written by the current member.
type MiniappMineReviewItemResponse struct {
	ID            int64      `json:"id"`
	OrderID       int64      `json:"order_id"`
	OrderItemID   int64      `json:"order_item_id"`
	ProductID     int64      `json:"product_id"`
	ProductName   string     `json:"product_name"`
	SKUID         int64      `json:"sku_id"`
	SKUSpecValues *string    `json:"sku_spec_values" extensions:"x-nullable"`
	Rating        int        `json:"rating"`
	Content       string     `json:"content"`
	IsAnonymous   int8       `json:"is_anonymous"`
	Status        int8       `json:"status"`
	StatusText    string     `json:"status_text"`
	AuditReason   string     `json:"audit_reason"`
	HasImage      int8       `json:"has_image"`
	Images        []string   `json:"images"`
	ReplyContent  string     `json:"reply_content"`
	ReplyAt       *time.Time `json:"reply_at" extensions:"x-nullable"`
	CreatedAt     time.Time  `json:"created_at"`
} // @name MiniappMineReviewItemResponse

// MiniappReviewDetailResponse represents the detail response for a product review in miniapp.
// For authors, includes private audit notes and order links; for others, private fields are omitted.
type MiniappReviewDetailResponse struct {
	ID            int64      `json:"id"`
	ProductID     int64      `json:"product_id"`
	ProductName   string     `json:"product_name"`
	SKUID         int64      `json:"sku_id"`
	SKUSpecValues *string    `json:"sku_spec_values" extensions:"x-nullable"`
	Rating        int        `json:"rating"`
	Content       string     `json:"content"`
	IsAnonymous   int8       `json:"is_anonymous"`
	Status        int8       `json:"status"`
	StatusText    string     `json:"status_text"`
	HasImage      int8       `json:"has_image"`
	Images        []string   `json:"images"`
	ReplyContent  string     `json:"reply_content"`
	ReplyAt       *time.Time `json:"reply_at" extensions:"x-nullable"`
	CreatedAt     time.Time  `json:"created_at"`
	IsAuthor      bool       `json:"is_author"`
	UserNickname  string     `json:"user_nickname"`
	UserAvatar    string     `json:"user_avatar"`

	// Author-only private fields (nil for non-authors, omitted from JSON)
	OrderID     *int64  `json:"order_id,omitempty" extensions:"x-nullable"`
	OrderItemID *int64  `json:"order_item_id,omitempty" extensions:"x-nullable"`
	AuditReason *string `json:"audit_reason,omitempty" extensions:"x-nullable"`
} // @name MiniappReviewDetailResponse

// MiniappProductReviewSummary represents statistical summaries of public reviews.
type MiniappProductReviewSummary struct {
	TotalCount    int64  `json:"total_count"`
	HasImageCount int64  `json:"has_image_count"`
	GoodCount     int64  `json:"good_count"`
	MediumCount   int64  `json:"medium_count"`
	BadCount      int64  `json:"bad_count"`
	AverageRating string `json:"average_rating"` // e.g. "4.8"
	GoodRate      string `json:"good_rate"`      // e.g. "98.0%"
} // @name MiniappProductReviewSummary

// MiniappProductReviewItemResponse represents an approved public review visible on product detail.
type MiniappProductReviewItemResponse struct {
	ID            int64      `json:"id"`
	Rating        int        `json:"rating"`
	Content       string     `json:"content"`
	SKUSpecValues *string    `json:"sku_spec_values" extensions:"x-nullable"`
	IsAnonymous   int8       `json:"is_anonymous"`
	UserNickname  string     `json:"user_nickname"`
	UserAvatar    string     `json:"user_avatar"`
	HasImage      int8       `json:"has_image"`
	Images        []string   `json:"images"`
	ReplyContent  string     `json:"reply_content"`
	ReplyAt       *time.Time `json:"reply_at" extensions:"x-nullable"`
	CreatedAt     time.Time  `json:"created_at"`
} // @name MiniappProductReviewItemResponse

// MiniappProductReviewsResponse represents the product review list with summary.
type MiniappProductReviewsResponse struct {
	Summary MiniappProductReviewSummary        `json:"summary"`
	Items   []MiniappProductReviewItemResponse `json:"items"`
} // @name MiniappProductReviewsResponse

// MiniappProductReviewsQuery represents the query parameters for product review list.
type MiniappProductReviewsQuery struct {
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=10"`
	Filter   string `form:"filter"` // "all", "has_image", "good", "medium", "bad"
	Rating   int    `form:"rating"` // 1-5
} // @name MiniappProductReviewsQuery

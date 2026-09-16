package dict_service

import "time"

// ProductDictServiceRow represents a single product service guarantee row.
type ProductDictServiceRow struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Sort        int       `json:"sort"`
	Status      int8      `json:"status"` // 0: 禁用, 1: 启用
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
} // @name ProductDictServiceRow

// DictServiceListRequest represents query parameters for listing service guarantees.
type DictServiceListRequest struct {
	Page     int    `form:"page" json:"page"`
	PageSize int    `form:"page_size" json:"page_size"`
	Keyword  string `form:"keyword" json:"keyword"`
	Status   *int8  `form:"status" json:"status"` // 0: 禁用, 1: 启用
} // @name DictServiceListRequest

// DictServiceCreateRequest represents payload to create a service guarantee.
type DictServiceCreateRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Sort        int    `json:"sort"`
	Status      *int8  `json:"status"` // 0: 禁用, 1: 启用, 默认为 1
} // @name DictServiceCreateRequest

// DictServiceUpdateRequest represents payload to update a service guarantee.
type DictServiceUpdateRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Sort        *int   `json:"sort"`
	Status      *int8  `json:"status"` // 0: 禁用, 1: 启用
} // @name DictServiceUpdateRequest

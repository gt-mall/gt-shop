package dict_unit

import "time"

// ProductDictUnitRow represents a single product unit row.
type ProductDictUnitRow struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Sort        int       `json:"sort"`
	Status      int8      `json:"status"` // 0: 禁用, 1: 启用
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
} // @name ProductDictUnitRow

// DictUnitListRequest represents query parameters for listing product units.
type DictUnitListRequest struct {
	Page     int    `form:"page" json:"page"`
	PageSize int    `form:"page_size" json:"page_size"`
	Keyword  string `form:"keyword" json:"keyword"`
	Status   *int8  `form:"status" json:"status"` // 0: 禁用, 1: 启用
} // @name DictUnitListRequest

// DictUnitCreateRequest represents payload to create a product unit.
type DictUnitCreateRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Sort        int    `json:"sort"`
	Status      *int8  `json:"status"` // 0: 禁用, 1: 启用, 默认为 1
} // @name DictUnitCreateRequest

// DictUnitUpdateRequest represents payload to update a product unit.
type DictUnitUpdateRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Sort        *int   `json:"sort"`
	Status      *int8  `json:"status"` // 0: 禁用, 1: 启用
} // @name DictUnitUpdateRequest

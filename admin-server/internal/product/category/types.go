package category

import "time"

type CreateRequest struct {
	ParentID int64  `json:"parent_id"`
	Name     string `json:"name" binding:"required,max=255"`
	ImageID  *int64 `json:"image_id" extensions:"x-nullable"`
	Sort     *int   `json:"sort"`
	Status   *int8  `json:"status"`
} // @name ProductCategoryCreateRequest

type UpdateRequest struct {
	ID       int64   `json:"id" binding:"required"`
	ParentID *int64  `json:"parent_id"`
	Name     *string `json:"name"`
	ImageID  *int64  `json:"image_id" extensions:"x-nullable"`
	Sort     *int    `json:"sort"`
	Status   *int8   `json:"status"`
} // @name ProductCategoryUpdateRequest

type DeleteRequest struct {
	ID int64 `json:"id" binding:"required"`
} // @name ProductCategoryDeleteRequest

type CategoryResponse struct {
	ID        int64     `json:"id"`
	ParentID  int64     `json:"parent_id"`
	Name      string    `json:"name"`
	ImageID   *int64    `json:"image_id" extensions:"x-nullable"`
	ImageURL  string    `json:"image_url"`
	Sort      int       `json:"sort"`
	Status    int8      `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
} // @name ProductCategoryRow

type OptionResponse struct {
	ID       int64  `json:"id"`
	ParentID int64  `json:"parent_id"`
	Name     string `json:"name"`
	Status   int8   `json:"status"`
	Sort     int    `json:"sort"`
} // @name ProductCategoryOption

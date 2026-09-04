package role

import "time"

type CreateRequest struct {
	Name   string `json:"name" binding:"required,max=64"`
	Status *int8  `json:"status"`
	Sort   *int   `json:"sort"`
	Remark string `json:"remark" binding:"max=255"`
} // @name RoleCreateRequest

type IDRequest struct {
	ID int64 `json:"id" binding:"required"`
} // @name RoleIDRequest

type UpdateRequest struct {
	ID     int64   `json:"id" binding:"required"`
	Name   *string `json:"name"`
	Status *int8   `json:"status"`
	Sort   *int    `json:"sort"`
	Remark *string `json:"remark"`
} // @name RoleUpdateRequest

type ListRequest struct {
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	Keyword  string `json:"keyword"`
	Status   *int8  `json:"status"`
} // @name RoleListRequest

type RoleResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Code      string    `json:"code"`
	Status    int8      `json:"status"`
	Sort      int       `json:"sort"`
	Remark    string    `json:"remark"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
} // @name RoleRow

type PermissionRequest struct {
	ID int64 `json:"id" binding:"required"`
} // @name RolePermissionRequest

type SavePermissionRequest struct {
	ID          int64    `json:"id" binding:"required"`
	Permissions []string `json:"permissions"`
} // @name RoleSavePermissionRequest

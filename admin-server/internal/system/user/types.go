package user

import "time"

type CreateRequest struct {
	Username string `json:"username" binding:"required,max=64"`
	Password string `json:"password" binding:"required,min=6,max=72"`
	Nickname string `json:"nickname" binding:"max=100"`
	Status   *int8  `json:"status"`
}

type IDRequest struct {
	ID int64 `json:"id" binding:"required"`
}

type UpdateRequest struct {
	Username *string `json:"username"`
	Password *string `json:"password"`
	Nickname *string `json:"nickname"`
	Status   *int8   `json:"status"`
	ID       int64   `json:"id" binding:"required"`
}

type ListRequest struct {
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	Username string `json:"username"`
}

type UserResponse struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Nickname  string    `json:"nickname"`
	Status    int8      `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

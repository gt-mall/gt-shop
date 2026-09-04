package auth

import "time"

type LoginRequest struct {
	Username string `json:"username" binding:"required,max=64"`
	Password string `json:"password" binding:"required,max=72"`
	Remember bool   `json:"remember"`
} // @name LoginRequest

type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
} // @name LoginResponse

type UserInfo struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Nickname  string    `json:"nickname"`
	Status    int8      `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
} // @name CurrentUser

type RoleInfo struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Code   string `json:"code"`
	Status int8   `json:"status"`
} // @name CurrentRole

type CurrentUserResponse struct {
	User        UserInfo   `json:"user"`
	Roles       []RoleInfo `json:"roles"`
	Permissions []string   `json:"permissions"`
} // @name CurrentUserResponse

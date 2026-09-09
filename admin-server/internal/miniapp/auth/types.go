package auth

import "time"

type WechatLoginRequest struct {
	Code string `json:"code" binding:"required"`
} // @name WechatLoginRequest

type MemberInfo struct {
	ID          int64      `json:"id"`
	Nickname    string     `json:"nickname"`
	AvatarID    *int64     `json:"avatar_id"`
	AvatarURL   string     `json:"avatar_url"`
	Mobile      *string    `json:"mobile"`
	Status      int8       `json:"status"`
	LastLoginAt *time.Time `json:"last_login_at"`
	CreatedAt   time.Time  `json:"created_at"`
} // @name MemberInfo

type WechatLoginResponse struct {
	Token       string     `json:"token"`
	ExpiresAt   time.Time  `json:"expires_at"`
	IsNewMember bool       `json:"is_new_member"`
	Member      MemberInfo `json:"member"`
} // @name WechatLoginResponse

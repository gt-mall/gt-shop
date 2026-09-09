package member

import "time"

type MemberListRequest struct {
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	Keyword  string `json:"keyword"`
	Status   *int8  `json:"status"`
} // @name MemberListRequest

type MemberIDRequest struct {
	ID int64 `json:"id" binding:"required"`
} // @name MemberIDRequest

type MemberRow struct {
	ID          int64      `json:"id"`
	Nickname    string     `json:"nickname"`
	AvatarID    *int64     `json:"avatar_id"`
	AvatarURL   string     `json:"avatar_url"`
	Mobile      *string    `json:"mobile"`
	Status      int8       `json:"status"`
	LastLoginAt *time.Time `json:"last_login_at"`
	CreatedAt   time.Time  `json:"created_at"`
} // @name MemberRow

type MemberAuthRow struct {
	AuthType     string    `json:"auth_type"`
	AppID        string    `json:"app_id"`
	OpenID       string    `json:"openid"`
	UnionID      *string   `json:"unionid"`
	ProviderData *string   `json:"provider_data"`
	CreatedAt    time.Time `json:"created_at"`
} // @name MemberAuthRow

type MemberAddressRow struct {
	ID           int64     `json:"id"`
	MemberID     int64     `json:"member_id"`
	Name         string    `json:"name"`
	Mobile       string    `json:"mobile"`
	ProvinceCode string    `json:"province_code"`
	Province     string    `json:"province"`
	CityCode     string    `json:"city_code"`
	City         string    `json:"city"`
	DistrictCode string    `json:"district_code"`
	District     string    `json:"district"`
	Address      string    `json:"address"`
	PostalCode   *string   `json:"postal_code"`
	IsDefault    int8      `json:"is_default"`
	CreatedAt    time.Time `json:"created_at"`
} // @name MemberAddressRow

type MemberDetailResponse struct {
	Member    MemberRow          `json:"member"`
	Auth      []MemberAuthRow    `json:"auth"`
	Addresses []MemberAddressRow `json:"addresses"`
} // @name MemberDetailResponse

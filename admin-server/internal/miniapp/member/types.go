package member

import "time"

type MemberInfo struct {
	ID          int64      `json:"id"`
	Nickname    string     `json:"nickname"`
	AvatarID    *int64     `json:"avatar_id"`
	AvatarURL   string     `json:"avatar_url"`
	Mobile      *string    `json:"mobile"`
	Status      int8       `json:"status"`
	LastLoginAt *time.Time `json:"last_login_at"`
	CreatedAt   time.Time  `json:"created_at"`
} // @name MiniappMemberInfo

type UpdateProfileRequest struct {
	Nickname *string `json:"nickname"`
	Mobile   *string `json:"mobile"`
} // @name MiniappUpdateProfileRequest

type AvatarResponse struct {
	AvatarID  int64  `json:"avatar_id"`
	AvatarURL string `json:"avatar_url"`
} // @name MiniappAvatarResponse

type AddressResponse struct {
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
} // @name MiniappAddressResponse

type AddressCreateRequest struct {
	Name         string  `json:"name" binding:"required"`
	Mobile       string  `json:"mobile" binding:"required"`
	ProvinceCode string  `json:"province_code" binding:"required"`
	Province     string  `json:"province" binding:"required"`
	CityCode     string  `json:"city_code" binding:"required"`
	City         string  `json:"city" binding:"required"`
	DistrictCode string  `json:"district_code" binding:"required"`
	District     string  `json:"district" binding:"required"`
	Address      string  `json:"address" binding:"required"`
	PostalCode   *string `json:"postal_code"`
	IsDefault    *int8   `json:"is_default"`
} // @name MiniappAddressCreateRequest

type AddressUpdateRequest struct {
	ID           int64   `json:"id" binding:"required"`
	Name         string  `json:"name" binding:"required"`
	Mobile       string  `json:"mobile" binding:"required"`
	ProvinceCode string  `json:"province_code" binding:"required"`
	Province     string  `json:"province" binding:"required"`
	CityCode     string  `json:"city_code" binding:"required"`
	City         string  `json:"city" binding:"required"`
	DistrictCode string  `json:"district_code" binding:"required"`
	District     string  `json:"district" binding:"required"`
	Address      string  `json:"address" binding:"required"`
	PostalCode   *string `json:"postal_code"`
	IsDefault    *int8   `json:"is_default"`
} // @name MiniappAddressUpdateRequest

type AddressIDRequest struct {
	ID int64 `json:"id" binding:"required"`
} // @name MiniappAddressIDRequest

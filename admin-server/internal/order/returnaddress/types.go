package returnaddress

import "time"

// ReturnAddressItem 退货地址信息
type ReturnAddressItem struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Receiver     string    `json:"receiver"`
	Phone        string    `json:"phone"`
	ProvinceCode string    `json:"province_code"`
	CityCode     string    `json:"city_code"`
	DistrictCode string    `json:"district_code"`
	Address      string    `json:"address"`
	IsDefault    int8      `json:"is_default"`
	Status       int8      `json:"status"`
	Sort         int       `json:"sort"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
} // @name ReturnAddressItem

// ListReturnAddressesRequest 列表查询请求
type ListReturnAddressesRequest struct {
	Status *int8  `json:"status,omitempty"`
	Search string `json:"search,omitempty"`
} // @name ListReturnAddressesRequest

// CreateReturnAddressRequest 创建退货地址请求
type CreateReturnAddressRequest struct {
	Name         string `json:"name" binding:"required"`
	Receiver     string `json:"receiver" binding:"required"`
	Phone        string `json:"phone" binding:"required"`
	ProvinceCode string `json:"province_code"`
	CityCode     string `json:"city_code"`
	DistrictCode string `json:"district_code"`
	Address      string `json:"address" binding:"required"`
	IsDefault    int8   `json:"is_default"`
	Status       int8   `json:"status"`
	Sort         int    `json:"sort"`
} // @name CreateReturnAddressRequest

// UpdateReturnAddressRequest 更新退货地址请求
type UpdateReturnAddressRequest struct {
	ID           int64  `json:"id" binding:"required"`
	Name         string `json:"name" binding:"required"`
	Receiver     string `json:"receiver" binding:"required"`
	Phone        string `json:"phone" binding:"required"`
	ProvinceCode string `json:"province_code"`
	CityCode     string `json:"city_code"`
	DistrictCode string `json:"district_code"`
	Address      string `json:"address" binding:"required"`
	IsDefault    int8   `json:"is_default"`
	Status       int8   `json:"status"`
	Sort         int    `json:"sort"`
} // @name UpdateReturnAddressRequest

// IDRequest 单记录操作请求
type IDRequest struct {
	ID int64 `json:"id" binding:"required"`
} // @name ReturnAddressIDRequest

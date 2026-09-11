package freight

import "time"

type ExtraFeeRule struct {
	ProvinceCode string `json:"province_code"`
	CityCode     string `json:"city_code"`
	Fee          string `json:"fee"`
} // @name FreightExtraFeeRule

type NoDeliveryRule struct {
	ProvinceCode string `json:"province_code"`
	CityCode     string `json:"city_code"`
} // @name FreightNoDeliveryRule

type FreightTemplateCreateRequest struct {
	Name       string           `json:"name" binding:"required,max=100"`
	BaseFee    string           `json:"base_fee" binding:"required"`
	ExtraFees  []ExtraFeeRule   `json:"extra_fees"`
	NoDelivery []NoDeliveryRule `json:"no_delivery"`
	Status     *int8            `json:"status"` // 0:禁用, 1:启用, 默认1
} // @name FreightTemplateCreateRequest

type FreightTemplateUpdateRequest struct {
	ID         int64            `json:"id" binding:"required"`
	Name       string           `json:"name" binding:"required,max=100"`
	BaseFee    string           `json:"base_fee" binding:"required"`
	ExtraFees  []ExtraFeeRule   `json:"extra_fees"`
	NoDelivery []NoDeliveryRule `json:"no_delivery"`
	Status     *int8            `json:"status"`
} // @name FreightTemplateUpdateRequest

type FreightTemplateStatusRequest struct {
	ID     int64 `json:"id" binding:"required"`
	Status int8  `json:"status"` // 0:禁用, 1:启用
} // @name FreightTemplateStatusRequest

type FreightTemplateIDRequest struct {
	ID int64 `json:"id" binding:"required"`
} // @name FreightTemplateIDRequest

type FreightTemplateListRequest struct {
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	Keyword  string `json:"keyword"`
	Status   *int8  `json:"status"`
} // @name FreightTemplateListRequest

type FreightTemplateRow struct {
	ID              int64     `json:"id"`
	Name            string    `json:"name"`
	BaseFee         string    `json:"base_fee"`
	ExtraFeeCount   int       `json:"extra_fee_count"`
	NoDeliveryCount int       `json:"no_delivery_count"`
	Status          int8      `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
} // @name FreightTemplateRow

type FreightTemplateDetail struct {
	ID         int64            `json:"id"`
	Name       string           `json:"name"`
	BaseFee    string           `json:"base_fee"`
	ExtraFees  []ExtraFeeRule   `json:"extra_fees"`
	NoDelivery []NoDeliveryRule `json:"no_delivery"`
	Status     int8             `json:"status"`
	CreatedAt  time.Time        `json:"created_at"`
	UpdatedAt  time.Time        `json:"updated_at"`
} // @name FreightTemplateDetail

type FreightTemplateOption struct {
	ID         int64            `json:"id"`
	Name       string           `json:"name"`
	BaseFee    string           `json:"base_fee"`
	Status     int8             `json:"status"`
	ExtraFees  []ExtraFeeRule   `json:"extra_fees"`
	NoDelivery []NoDeliveryRule `json:"no_delivery"`
} // @name FreightTemplateOption

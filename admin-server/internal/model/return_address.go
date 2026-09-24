package model

import "time"

// OrderReturnAddress 退货地址表
type OrderReturnAddress struct {
	ID           int64     `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	Name         string    `gorm:"column:name;size:64;not null" json:"name"`
	Receiver     string    `gorm:"column:receiver;size:64;not null" json:"receiver"`
	Phone        string    `gorm:"column:phone;size:32;not null" json:"phone"`
	ProvinceCode string    `gorm:"column:province_code;size:32;not null;default:''" json:"province_code"`
	CityCode     string    `gorm:"column:city_code;size:32;not null;default:''" json:"city_code"`
	DistrictCode string    `gorm:"column:district_code;size:32;not null;default:''" json:"district_code"`
	Address      string    `gorm:"column:address;size:255;not null" json:"address"`
	IsDefault    int8      `gorm:"column:is_default;not null;default:0" json:"is_default"` // 是否默认：1是 0否
	Status       int8      `gorm:"column:status;not null;default:1" json:"status"`         // 状态：1启用 0禁用
	Sort         int       `gorm:"column:sort;not null;default:0" json:"sort"`             // 排序值，越小越靠前
	CreatedAt    time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (OrderReturnAddress) TableName() string { return "order_return_address" }

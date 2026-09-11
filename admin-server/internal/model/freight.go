package model

import (
	"time"

	"gorm.io/gorm"
)

type FreightTemplate struct {
	ID         int64          `gorm:"primaryKey;autoIncrement;column:id"`
	Name       string         `gorm:"column:name;size:100;not null"`
	BaseFee    string         `gorm:"column:base_fee;type:decimal(12,2);not null"`
	ExtraFees  *string        `gorm:"column:extra_fees;type:json;default:null"`
	NoDelivery *string        `gorm:"column:no_delivery;type:json;default:null"`
	Status     int8           `gorm:"column:status;not null"`
	CreatedAt  time.Time      `gorm:"column:created_at"`
	UpdatedAt  time.Time      `gorm:"column:updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (FreightTemplate) TableName() string { return "freight_template" }

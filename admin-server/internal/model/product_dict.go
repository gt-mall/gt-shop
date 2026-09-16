package model

import "time"

// ProductDictService represents the product_dict_service table.
type ProductDictService struct {
	ID          int64     `gorm:"primaryKey;autoIncrement;column:id"`
	Name        string    `gorm:"column:name;size:64;not null"`
	Description string    `gorm:"column:description;size:255;not null;default:''"`
	Sort        int       `gorm:"column:sort;not null"`
	Status      int8      `gorm:"column:status;not null"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

func (ProductDictService) TableName() string { return "product_dict_service" }

// ProductDictUnit represents the product_dict_unit table.
type ProductDictUnit struct {
	ID          int64     `gorm:"primaryKey;autoIncrement;column:id"`
	Name        string    `gorm:"column:name;size:32;not null"`
	Description string    `gorm:"column:description;size:255;not null;default:''"`
	Sort        int       `gorm:"column:sort;not null"`
	Status      int8      `gorm:"column:status;not null"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

func (ProductDictUnit) TableName() string { return "product_dict_unit" }

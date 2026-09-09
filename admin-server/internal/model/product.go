package model

import (
	"time"

	"gorm.io/gorm"
)

type ProductCategory struct {
	ID        int64          `gorm:"primaryKey;autoIncrement;column:id"`
	ParentID  int64          `gorm:"column:parent_id;not null"`
	Name      string         `gorm:"column:name;size:255;not null"`
	ImageID   *int64         `gorm:"column:image_id;default:null"`
	Sort      int            `gorm:"column:sort;not null"`
	Status    int8           `gorm:"column:status;not null"`
	CreatedAt time.Time      `gorm:"column:created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (ProductCategory) TableName() string { return "product_category" }

type Product struct {
	ID             int64          `gorm:"primaryKey;autoIncrement;column:id"`
	CategoryID     int64          `gorm:"column:category_id;not null"`
	Name           string         `gorm:"column:name;size:255;not null"`
	Subtitle       string         `gorm:"column:subtitle;size:255;not null"`
	MainImageID    *int64         `gorm:"column:main_image_id;default:null"`
	GalleryFileIDs *string        `gorm:"column:gallery_file_ids;type:json;default:null"`
	Specs          *string        `gorm:"column:specs;type:json;default:null"`
	Params         *string        `gorm:"column:params;type:json;default:null"`
	Detail         *string        `gorm:"column:detail;type:longtext;default:null"`
	Status         int8           `gorm:"column:status;not null"`
	Sort           int            `gorm:"column:sort;not null"`
	SalesCount     int64          `gorm:"column:sales_count;not null"`
	VirtualSales   int            `gorm:"column:virtual_sales;not null"`
	CreatedAt      time.Time      `gorm:"column:created_at"`
	UpdatedAt      time.Time      `gorm:"column:updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (Product) TableName() string { return "product" }

type ProductSKU struct {
	ID          int64          `gorm:"primaryKey;autoIncrement;column:id"`
	ProductID   int64          `gorm:"column:product_id;not null"`
	SpecValues  string         `gorm:"column:spec_values;type:json;not null"`
	SpecHash    string         `gorm:"column:spec_hash;size:64;not null"`
	SKUCode     *string        `gorm:"column:sku_code;size:255;default:null"`
	Barcode     *string        `gorm:"column:barcode;size:255;default:null"`
	ImageID     *int64         `gorm:"column:image_id;default:null"`
	Price       string         `gorm:"column:price;type:decimal(12,2);not null"`
	MarketPrice string         `gorm:"column:market_price;type:decimal(12,2);not null"`
	CostPrice   string         `gorm:"column:cost_price;type:decimal(12,2);not null"`
	Unit        *string        `gorm:"column:unit;size:255;default:null"`
	WeightGram  *int           `gorm:"column:weight_gram;default:null"`
	Sort        int            `gorm:"column:sort;not null"`
	Status      int8           `gorm:"column:status;not null"`
	CreatedAt   time.Time      `gorm:"column:created_at"`
	UpdatedAt   time.Time      `gorm:"column:updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (ProductSKU) TableName() string { return "product_sku" }

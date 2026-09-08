package model

import "time"

type ProductSkuInventory struct {
	ID           int64     `gorm:"primaryKey;autoIncrement;column:id"`
	ProductID    int64     `gorm:"column:product_id;not null"`
	SKUID        int64     `gorm:"column:sku_id;not null;uniqueIndex:uk_inventory_sku"`
	Stock        int       `gorm:"column:stock;not null;default:0"`
	LockedStock  int       `gorm:"column:locked_stock;not null;default:0"`
	WarningStock int       `gorm:"column:warning_stock;not null;default:0"`
	Version      uint32    `gorm:"column:version;not null;default:0"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

func (ProductSkuInventory) TableName() string { return "product_sku_inventory" }

type ProductSkuInventoryLog struct {
	ID            int64     `gorm:"primaryKey;autoIncrement;column:id"`
	InventoryID   int64     `gorm:"column:inventory_id;not null"`
	ProductID     int64     `gorm:"column:product_id;not null"`
	SKUID         int64     `gorm:"column:sku_id;not null"`
	RequestID     string    `gorm:"column:request_id;size:36;not null;uniqueIndex:uk_inventory_log_request"`
	RequestHash   string    `gorm:"column:request_hash;size:64;not null"`
	ChangeType    string    `gorm:"column:change_type;size:32;not null"`
	Quantity      int       `gorm:"column:quantity;not null"`
	StockBefore   int       `gorm:"column:stock_before;not null"`
	StockDelta    int       `gorm:"column:stock_delta;not null"`
	StockAfter    int       `gorm:"column:stock_after;not null"`
	LockedBefore  int       `gorm:"column:locked_before;not null"`
	LockedDelta   int       `gorm:"column:locked_delta;not null"`
	LockedAfter   int       `gorm:"column:locked_after;not null"`
	WarningBefore int       `gorm:"column:warning_before;not null"`
	WarningAfter  int       `gorm:"column:warning_after;not null"`
	VersionBefore uint32    `gorm:"column:version_before;not null"`
	VersionAfter  uint32    `gorm:"column:version_after;not null"`
	SKUSnapshot   string    `gorm:"column:sku_snapshot;type:json;not null"`
	OperatorID    int64     `gorm:"column:operator_id;not null"`
	OperatorName  string    `gorm:"column:operator_name;size:100;not null"`
	Remark        string    `gorm:"column:remark;size:500;not null"`
	CreatedAt     time.Time `gorm:"column:created_at"`
}

func (ProductSkuInventoryLog) TableName() string { return "product_sku_inventory_log" }

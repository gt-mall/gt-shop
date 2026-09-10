package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	OrderStatusPendingPayment  int8 = 10
	OrderStatusPendingShipment int8 = 20
	OrderStatusPendingReceipt  int8 = 30
	OrderStatusCompleted       int8 = 40
	OrderStatusCancelled       int8 = 50
)

func OrderStatusText(status int8) string {
	switch status {
	case OrderStatusPendingPayment:
		return "待支付"
	case OrderStatusPendingShipment:
		return "待发货"
	case OrderStatusPendingReceipt:
		return "待收货"
	case OrderStatusCompleted:
		return "已完成"
	case OrderStatusCancelled:
		return "已取消"
	default:
		return "未知状态"
	}
}

type OrderMain struct {
	ID             int64          `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	OrderNo        string         `gorm:"column:order_no;size:64;not null;uniqueIndex:uk_order_no" json:"order_no"`
	MemberID       int64          `gorm:"column:member_id;not null;index:idx_order_main_member_id;uniqueIndex:uk_order_member_request,priority:1" json:"member_id"`
	RequestID      string         `gorm:"column:request_id;size:64;not null;uniqueIndex:uk_order_member_request,priority:2" json:"request_id"`
	Status         int8           `gorm:"column:status;not null;default:20;index:idx_order_main_status" json:"status"`
	TotalAmount    string         `gorm:"column:total_amount;type:decimal(10,2);not null;default:0.00" json:"total_amount"`
	PayAmount      string         `gorm:"column:pay_amount;type:decimal(10,2);not null;default:0.00" json:"pay_amount"`
	FreightAmount  string         `gorm:"column:freight_amount;type:decimal(10,2);not null;default:0.00" json:"freight_amount"`
	DiscountAmount string         `gorm:"column:discount_amount;type:decimal(10,2);not null;default:0.00" json:"discount_amount"`
	ExpressCompany *string        `gorm:"column:express_company;size:64;default:null" json:"express_company"`
	ExpressNo      *string        `gorm:"column:express_no;size:64;default:null" json:"express_no"`
	Remark         string         `gorm:"column:remark;size:255;not null;default:''" json:"remark"`
	PaidAt         *time.Time     `gorm:"column:paid_at;default:null" json:"paid_at"`
	ShippedAt      *time.Time     `gorm:"column:shipped_at;default:null" json:"shipped_at"`
	CompletedAt    *time.Time     `gorm:"column:completed_at;default:null" json:"completed_at"`
	CancelledAt    *time.Time     `gorm:"column:cancelled_at;default:null" json:"cancelled_at"`
	CreatedAt      time.Time      `gorm:"column:created_at;index:idx_order_main_created_at" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"column:deleted_at;index" json:"deleted_at,omitempty"`
}

func (OrderMain) TableName() string { return "order_main" }

type OrderItem struct {
	ID            int64     `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	OrderID       int64     `gorm:"column:order_id;not null;index:idx_order_item_order_id" json:"order_id"`
	ProductID     int64     `gorm:"column:product_id;not null" json:"product_id"`
	SKUID         int64     `gorm:"column:sku_id;not null;index:idx_order_item_sku_id" json:"sku_id"`
	ProductName   string    `gorm:"column:product_name;size:255;not null" json:"product_name"`
	SKUCode       string    `gorm:"column:sku_code;size:64;not null;default:''" json:"sku_code"`
	SKUSpecValues *string   `gorm:"column:sku_spec_values;type:json;default:null" json:"sku_spec_values"`
	SKUImageID    *int64    `gorm:"column:sku_image_id;default:null" json:"sku_image_id"`
	SKUImageURL   string    `gorm:"column:sku_image_url;size:1024;not null;default:''" json:"sku_image_url"`
	UnitPrice     string    `gorm:"column:unit_price;type:decimal(10,2);not null;default:0.00" json:"unit_price"`
	Quantity      int       `gorm:"column:quantity;not null;default:1" json:"quantity"`
	TotalAmount   string    `gorm:"column:total_amount;type:decimal(10,2);not null;default:0.00" json:"total_amount"`
	CreatedAt     time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (OrderItem) TableName() string { return "order_item" }

type OrderAddress struct {
	ID           int64     `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	OrderID      int64     `gorm:"column:order_id;not null;uniqueIndex:uk_order_address_order_id" json:"order_id"`
	Name         string    `gorm:"column:name;size:255;not null" json:"name"`
	Mobile       string    `gorm:"column:mobile;size:255;not null" json:"mobile"`
	ProvinceCode string    `gorm:"column:province_code;size:255;not null;default:''" json:"province_code"`
	Province     string    `gorm:"column:province;size:255;not null" json:"province"`
	CityCode     string    `gorm:"column:city_code;size:255;not null;default:''" json:"city_code"`
	City         string    `gorm:"column:city;size:255;not null" json:"city"`
	DistrictCode string    `gorm:"column:district_code;size:255;not null;default:''" json:"district_code"`
	District     string    `gorm:"column:district;size:255;not null" json:"district"`
	Address      string    `gorm:"column:address;size:255;not null" json:"address"`
	PostalCode   *string   `gorm:"column:postal_code;size:255;default:null" json:"postal_code"`
	CreatedAt    time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (OrderAddress) TableName() string { return "order_address" }

type OrderLog struct {
	ID           int64     `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	OrderID      int64     `gorm:"column:order_id;not null;index:idx_order_log_order_id" json:"order_id"`
	Action       string    `gorm:"column:action;size:64;not null" json:"action"`
	OperatorType string    `gorm:"column:operator_type;size:32;not null" json:"operator_type"`
	OperatorID   int64     `gorm:"column:operator_id;not null;default:0" json:"operator_id"`
	OperatorName string    `gorm:"column:operator_name;size:100;not null;default:''" json:"operator_name"`
	Remark       string    `gorm:"column:remark;size:255;not null;default:''" json:"remark"`
	CreatedAt    time.Time `gorm:"column:created_at" json:"created_at"`
}

func (OrderLog) TableName() string { return "order_log" }

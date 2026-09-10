package model

import "time"

type CartItem struct {
	ID        int64     `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	MemberID  int64     `gorm:"column:member_id;not null;uniqueIndex:uk_cart_item_member_sku,priority:1;index:idx_cart_item_member_id" json:"member_id"`
	ProductID int64     `gorm:"column:product_id;not null" json:"product_id"`
	SKUID     int64     `gorm:"column:sku_id;not null;uniqueIndex:uk_cart_item_member_sku,priority:2" json:"sku_id"`
	Quantity  int       `gorm:"column:quantity;not null;default:1" json:"quantity"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (CartItem) TableName() string { return "cart_item" }

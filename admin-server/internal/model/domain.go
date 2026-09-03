package model

import "time"

// These database-shaped models keep the current commerce vocabulary together.
// GORM Gen output will use this package when database generation is introduced.
type Product struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Status    int16     `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type SKU struct {
	ID        int64  `json:"id"`
	ProductID int64  `json:"product_id"`
	Code      string `json:"code"`
	PriceCent int64  `json:"price_cent"`
}

type Inventory struct {
	SKUID     int64 `json:"sku_id"`
	Available int64 `json:"available"`
	Reserved  int64 `json:"reserved"`
}

type Order struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	TotalCent int64     `json:"total_cent"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type OrderItem struct {
	ID        int64 `json:"id"`
	OrderID   int64 `json:"order_id"`
	SKUID     int64 `json:"sku_id"`
	Quantity  int64 `json:"quantity"`
	PriceCent int64 `json:"price_cent"`
}

type Payment struct {
	ID         int64  `json:"id"`
	OrderID    int64  `json:"order_id"`
	AmountCent int64  `json:"amount_cent"`
	Status     string `json:"status"`
}

type Refund struct {
	ID         int64  `json:"id"`
	PaymentID  int64  `json:"payment_id"`
	AmountCent int64  `json:"amount_cent"`
	Status     string `json:"status"`
}

type AfterSale struct {
	ID      int64  `json:"id"`
	OrderID int64  `json:"order_id"`
	Reason  string `json:"reason"`
	Status  string `json:"status"`
}

type Coupon struct {
	ID           int64  `json:"id"`
	Code         string `json:"code"`
	DiscountCent int64  `json:"discount_cent"`
}

type User struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

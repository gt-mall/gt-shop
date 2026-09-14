package model

// These database-shaped models keep the current commerce vocabulary together.
// GORM Gen output will use this package when database generation is introduced.

type Inventory struct {
	SKUID     int64 `json:"sku_id"`
	Available int64 `json:"available"`
	Reserved  int64 `json:"reserved"`
}

// Order and OrderItem are defined in internal/model/order.go
// AfterSale and OrderRefund are defined in internal/model/after_sale.go

type Coupon struct {
	ID           int64  `json:"id"`
	Code         string `json:"code"`
	DiscountCent int64  `json:"discount_cent"`
}

type User struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

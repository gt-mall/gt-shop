package model

// These database-shaped models keep the current commerce vocabulary together.
// GORM Gen output will use this package when database generation is introduced.

type Inventory struct {
	SKUID     int64 `json:"sku_id"`
	Available int64 `json:"available"`
	Reserved  int64 `json:"reserved"`
}

// Order and OrderItem are defined in internal/model/order.go

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

package model

import (
	"time"
)

const (
	PaymentStatusPending int8 = 10 // 待支付
	PaymentStatusSuccess int8 = 20 // 支付成功
	PaymentStatusFailed  int8 = 30 // 支付失败
	PaymentStatusClosed  int8 = 40 // 已关闭
)

func PaymentStatusText(status int8) string {
	switch status {
	case PaymentStatusPending:
		return "待支付"
	case PaymentStatusSuccess:
		return "支付成功"
	case PaymentStatusFailed:
		return "支付失败"
	case PaymentStatusClosed:
		return "已关闭"
	default:
		return "未知状态"
	}
}

type OrderPayment struct {
	ID            int64      `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	PaymentNo     string     `gorm:"column:payment_no;size:64;not null;uniqueIndex:uk_order_payment_payment_no" json:"payment_no"`
	OrderID       int64      `gorm:"column:order_id;not null;index:idx_order_payment_order_id" json:"order_id"`
	Channel       string     `gorm:"column:channel;size:32;not null;default:'wechat'" json:"channel"`
	Amount        string     `gorm:"column:amount;type:decimal(12,2);not null;default:0.00" json:"amount"`
	Status        int8       `gorm:"column:status;not null;default:10;index:idx_order_payment_status" json:"status"`
	TransactionNo *string    `gorm:"column:transaction_no;size:128;default:null" json:"transaction_no"`
	PaidAt        *time.Time `gorm:"column:paid_at;default:null" json:"paid_at"`
	CreatedAt     time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at" json:"updated_at"`
}

func (OrderPayment) TableName() string { return "order_payment" }

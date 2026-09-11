package payment

import (
	"errors"
	"time"

	"mall/admin_core/internal/model"
)

var (
	ErrOrderNotFound      = errors.New("订单不存在或已被删除")
	ErrOrderNotBelong     = errors.New("订单不属于当前用户")
	ErrOrderAlreadyPaid   = errors.New("订单已支付，无法重复支付")
	ErrOrderCancelled     = errors.New("订单已取消，无法发起支付")
	ErrOrderInvalidStatus = errors.New("订单状态不可支付")
	ErrPaymentNotFound    = errors.New("支付单不存在")
	ErrPaymentClosed      = errors.New("支付单已关闭或已失效")
)

type CreatePaymentRequest struct {
	OrderID int64  `json:"order_id" binding:"required"` // 订单ID
	Channel string `json:"channel"`                     // 支付渠道，默认 mock
} // @name CreatePaymentRequest

type MockPayRequest struct {
	PaymentNo string `json:"payment_no"` // 支付单号 (P前缀)
	OrderID   int64  `json:"order_id"`   // 订单ID
	Channel   string `json:"channel"`    // 支付渠道
} // @name MockPayRequest

type PaymentResponse struct {
	ID            int64      `json:"id"`
	PaymentNo     string     `json:"payment_no"`
	OrderID       int64      `json:"order_id"`
	Channel       string     `json:"channel"`
	Amount        string     `json:"amount"`
	Status        int8       `json:"status"`
	StatusText    string     `json:"status_text"`
	TransactionNo *string    `json:"transaction_no"`
	PaidAt        *time.Time `json:"paid_at"`
	CreatedAt     time.Time  `json:"created_at"`
} // @name PaymentResponse

func BuildPaymentResponse(p *model.OrderPayment) PaymentResponse {
	if p == nil {
		return PaymentResponse{}
	}
	return PaymentResponse{
		ID:            p.ID,
		PaymentNo:     p.PaymentNo,
		OrderID:       p.OrderID,
		Channel:       p.Channel,
		Amount:        p.Amount,
		Status:        p.Status,
		StatusText:    model.PaymentStatusText(p.Status),
		TransactionNo: p.TransactionNo,
		PaidAt:        p.PaidAt,
		CreatedAt:     p.CreatedAt,
	}
}

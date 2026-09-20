package ordertimeout

import "time"

const (
	// DefaultOrderExpiry 待付款订单默认支付超时时间：30 分钟
	DefaultOrderExpiry = 30 * time.Minute
)

// CalculateExpiresAt 计算订单支付截止时间（基于订单创建时间）
func CalculateExpiresAt(createdAt time.Time) time.Time {
	return createdAt.Add(DefaultOrderExpiry)
}

// IsExpired 判断在指定的 now 时刻，订单是否已过支付截止时间
func IsExpired(createdAt time.Time, now time.Time) bool {
	return !now.Before(CalculateExpiresAt(createdAt))
}

// CanPay 判断订单当前是否可发起/完成支付
func CanPay(status int8, createdAt time.Time, now time.Time) bool {
	// 只有待付款状态（10）且未超时才允许支付
	if status != 10 {
		return false
	}
	return !IsExpired(createdAt, now)
}

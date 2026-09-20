package ordertimeout

import (
	"testing"
	"time"
)

func TestOrdertimeout(t *testing.T) {
	baseTime := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	expiresAt := CalculateExpiresAt(baseTime)

	expectedExpires := time.Date(2026, 9, 20, 12, 30, 0, 0, time.UTC)
	if !expiresAt.Equal(expectedExpires) {
		t.Fatalf("expected expires at %v, got %v", expectedExpires, expiresAt)
	}

	// 1秒前：未超时
	if IsExpired(baseTime, expiresAt.Add(-time.Second)) {
		t.Fatalf("1 second before expiresAt should not be expired")
	}

	// 精确到达截止时间：判定超时
	if !IsExpired(baseTime, expiresAt) {
		t.Fatalf("exact expiresAt should be expired")
	}

	// 超过截止时间：超时
	if !IsExpired(baseTime, expiresAt.Add(time.Second)) {
		t.Fatalf("after expiresAt should be expired")
	}

	// CanPay 检查
	if !CanPay(10, baseTime, expiresAt.Add(-time.Second)) {
		t.Fatalf("status 10 before expiry should be can pay")
	}
	if CanPay(10, baseTime, expiresAt) {
		t.Fatalf("status 10 at expiry should not be can pay")
	}
	if CanPay(20, baseTime, expiresAt.Add(-time.Second)) {
		t.Fatalf("status 20 (paid) should not be can pay")
	}
	if CanPay(50, baseTime, expiresAt.Add(-time.Second)) {
		t.Fatalf("status 50 (canceled) should not be can pay")
	}
}

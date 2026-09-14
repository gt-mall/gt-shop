package refund

import (
	"context"
	"fmt"
	"sync"
)

type RefundRequest struct {
	RefundNo      string
	PaymentNo     string
	TransactionNo string
	RefundAmount  string
	TotalAmount   string
	Reason        string
}

type RefundResult struct {
	Success          bool
	ProviderRefundNo string
	ErrorCode        string
	ErrorMessage     string
}

type Provider interface {
	Refund(ctx context.Context, req RefundRequest) (*RefundResult, error)
}

// MockProvider implements a mock refund provider for testing and local operation.
type MockProvider struct {
	mu         sync.RWMutex
	shouldFail bool
	failError  string
}

func NewMockProvider() *MockProvider {
	return &MockProvider{}
}

func (m *MockProvider) SetFailure(shouldFail bool, failError string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFail = shouldFail
	m.failError = failError
}

func (m *MockProvider) Refund(ctx context.Context, req RefundRequest) (*RefundResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.shouldFail {
		msg := m.failError
		if msg == "" {
			msg = "模拟退款失败"
		}
		return &RefundResult{
			Success:      false,
			ErrorCode:    "MOCK_REFUND_FAIL",
			ErrorMessage: msg,
		}, nil
	}

	return &RefundResult{
		Success:          true,
		ProviderRefundNo: fmt.Sprintf("MOCK_REFUND_TXN_%s", req.RefundNo),
	}, nil
}

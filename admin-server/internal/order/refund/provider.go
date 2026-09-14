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

type RefundQueryStatus string

const (
	RefundQueryStatusSuccess    RefundQueryStatus = "SUCCESS"
	RefundQueryStatusFailed     RefundQueryStatus = "FAILED"
	RefundQueryStatusProcessing RefundQueryStatus = "PROCESSING"
	RefundQueryStatusUnknown    RefundQueryStatus = "UNKNOWN"
)

type RefundQueryResult struct {
	Status           RefundQueryStatus
	ProviderRefundNo string
	ErrorCode        string
	ErrorMessage     string
}

type Provider interface {
	Refund(ctx context.Context, req RefundRequest) (*RefundResult, error)
	QueryRefund(ctx context.Context, refundNo, providerRefundNo string) (*RefundQueryResult, error)
}

// MockProvider implements a mock refund provider for testing and local operation.
type MockProvider struct {
	mu             sync.RWMutex
	shouldFail     bool
	failError      string
	records        map[string]*RefundResult
	queryOverrides map[string]*RefundQueryResult
}

func NewMockProvider() *MockProvider {
	return &MockProvider{
		records:        make(map[string]*RefundResult),
		queryOverrides: make(map[string]*RefundQueryResult),
	}
}

func (m *MockProvider) SetFailure(shouldFail bool, failError string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFail = shouldFail
	m.failError = failError
}

func (m *MockProvider) SetQueryOverride(refundNo string, result *RefundQueryResult) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.queryOverrides == nil {
		m.queryOverrides = make(map[string]*RefundQueryResult)
	}
	m.queryOverrides[refundNo] = result
}

func (m *MockProvider) Refund(ctx context.Context, req RefundRequest) (*RefundResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.records == nil {
		m.records = make(map[string]*RefundResult)
	}

	if m.shouldFail {
		msg := m.failError
		if msg == "" {
			msg = "模拟退款失败"
		}
		res := &RefundResult{
			Success:      false,
			ErrorCode:    "MOCK_REFUND_FAIL",
			ErrorMessage: msg,
		}
		m.records[req.RefundNo] = res
		return res, nil
	}

	res := &RefundResult{
		Success:          true,
		ProviderRefundNo: fmt.Sprintf("MOCK_REFUND_TXN_%s", req.RefundNo),
	}
	m.records[req.RefundNo] = res
	return res, nil
}

func (m *MockProvider) QueryRefund(ctx context.Context, refundNo, providerRefundNo string) (*RefundQueryResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.queryOverrides != nil {
		if res, ok := m.queryOverrides[refundNo]; ok {
			return res, nil
		}
	}

	if m.records != nil {
		if rec, ok := m.records[refundNo]; ok {
			if rec.Success {
				return &RefundQueryResult{
					Status:           RefundQueryStatusSuccess,
					ProviderRefundNo: rec.ProviderRefundNo,
				}, nil
			}
			return &RefundQueryResult{
				Status:       RefundQueryStatusFailed,
				ErrorCode:    rec.ErrorCode,
				ErrorMessage: rec.ErrorMessage,
			}, nil
		}
	}

	if m.shouldFail {
		return &RefundQueryResult{
			Status:       RefundQueryStatusFailed,
			ErrorCode:    "MOCK_REFUND_FAIL",
			ErrorMessage: m.failError,
		}, nil
	}

	// Completely unrecorded refundNo -> UNKNOWN, never synthesize SUCCESS
	return &RefundQueryResult{
		Status:       RefundQueryStatusUnknown,
		ErrorMessage: "未找到退款记录",
	}, nil
}

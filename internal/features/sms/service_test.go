package sms

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/luli-tech/twilio-Boss/internal/config"
	"github.com/luli-tech/twilio-Boss/internal/shared/apperrors"
	"github.com/luli-tech/twilio-Boss/pkg/logger"
)

// MockSMSRepo implements in-memory Repository for unit testing.
type MockSMSRepo struct {
	mu       sync.RWMutex
	messages map[string]*Message
}

func NewMockSMSRepo() *MockSMSRepo {
	return &MockSMSRepo{
		messages: make(map[string]*Message),
	}
}

func (m *MockSMSRepo) Create(ctx context.Context, msg *Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages[msg.SID] = msg
	return nil
}

func (m *MockSMSRepo) GetByID(ctx context.Context, id string) (*Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, msg := range m.messages {
		if msg.ID == id {
			return msg, nil
		}
	}
	return nil, apperrors.ErrSMSNotFound
}

func (m *MockSMSRepo) GetBySID(ctx context.Context, sid string) (*Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	msg, ok := m.messages[sid]
	if !ok {
		return nil, apperrors.ErrSMSNotFound
	}
	return msg, nil
}

func (m *MockSMSRepo) GetByCarrierID(ctx context.Context, carrierID string) (*Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, msg := range m.messages {
		if msg.CarrierID == carrierID {
			return msg, nil
		}
	}
	return nil, apperrors.ErrSMSNotFound
}

func (m *MockSMSRepo) UpdateStatus(ctx context.Context, sid string, status Status, errCode, errMsg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	msg, ok := m.messages[sid]
	if !ok {
		return apperrors.ErrSMSNotFound
	}
	msg.Status = status
	msg.ErrorCode = errCode
	msg.ErrorMessage = errMsg
	return nil
}

func (m *MockSMSRepo) UpdateCarrierID(ctx context.Context, sid, carrierID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	msg, ok := m.messages[sid]
	if !ok {
		return apperrors.ErrSMSNotFound
	}
	msg.CarrierID = carrierID
	return nil
}

func (m *MockSMSRepo) ListByAccount(ctx context.Context, accountID string, limit, offset int) ([]*Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []*Message
	for _, msg := range m.messages {
		if msg.AccountID == accountID {
			list = append(list, msg)
		}
	}
	return list, nil
}

// MockBillingService implements billing.Service for unit testing.
type MockBillingService struct {
	mu       sync.Mutex
	balances map[string]int64
	refunded map[string]int64
}

func NewMockBillingService(initialBalances map[string]int64) *MockBillingService {
	b := make(map[string]int64)
	for k, v := range initialBalances {
		b[k] = v
	}
	return &MockBillingService{
		balances: b,
		refunded: make(map[string]int64),
	}
}

func (b *MockBillingService) CheckBalance(ctx context.Context, accountID string, required int64) (bool, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.balances[accountID] >= required, nil
}

func (b *MockBillingService) DeductBalance(ctx context.Context, accountID string, amount int64, refID, desc string) (int64, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	curr := b.balances[accountID]
	if curr < amount {
		return 0, apperrors.ErrInsufficientFunds
	}
	b.balances[accountID] = curr - amount
	return b.balances[accountID], nil
}

func (b *MockBillingService) RefundBalance(ctx context.Context, accountID string, amount int64, refID, desc string) (int64, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.balances[accountID] += amount
	b.refunded[refID] += amount
	return b.balances[accountID], nil
}

func (b *MockBillingService) GetBalance(ctx context.Context, accountID string) (int64, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.balances[accountID], nil
}

type testGateway struct {
	latency time.Duration
}

func newTestGateway(latency time.Duration) *testGateway {
	return &testGateway{latency: latency}
}

func (g *testGateway) Dispatch(ctx context.Context, msg *Message) (*DispatchResult, error) {
	select {
	case <-time.After(g.latency):
	case <-ctx.Done():
		return nil, apperrors.ErrCarrierTimeout
	}

	return &DispatchResult{
		CarrierMessageID: fmt.Sprintf("CARRIER-%s", uuid.New().String()[:8]),
		Status:           StatusSent,
	}, nil
}

func TestSMSService_Send_Success(t *testing.T) {
	repo := NewMockSMSRepo()
	billing := NewMockBillingService(map[string]int64{"acc-1": 100000})
	gateway := newTestGateway(1 * time.Millisecond)
	cfg := &config.BillingConfig{
		DefaultRatePerSMS: 7500,
		Currency:          "USD",
	}

	svc := NewService(repo, billing, gateway, cfg, logger.Default())
	defer svc.Close()

	req := &SendRequest{
		AccountID:  "acc-1",
		AccountSID: "AC123456",
		From:       "+12025550100",
		To:         "+12025550199",
		Body:       "Hello from Callio CPaaS platform!",
	}

	msg, err := svc.Send(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if msg.Status != StatusSent {
		t.Errorf("expected status 'sent', got '%s'", msg.Status)
	}

	if msg.NumSegments != 1 {
		t.Errorf("expected 1 segment, got %d", msg.NumSegments)
	}

	if msg.Price != 7500 {
		t.Errorf("expected price 7500, got %d", msg.Price)
	}

	bal, _ := billing.GetBalance(context.Background(), "acc-1")
	expectedBal := int64(100000 - 7500)
	if bal != expectedBal {
		t.Errorf("expected remaining balance %d, got %d", expectedBal, bal)
	}
}

func TestSMSService_Send_InsufficientFunds(t *testing.T) {
	repo := NewMockSMSRepo()
	billing := NewMockBillingService(map[string]int64{"acc-1": 1000}) // balance < 7500
	gateway := newTestGateway(1 * time.Millisecond)
	cfg := &config.BillingConfig{
		DefaultRatePerSMS: 7500,
		Currency:          "USD",
	}

	svc := NewService(repo, billing, gateway, cfg, logger.Default())
	defer svc.Close()

	req := &SendRequest{
		AccountID:  "acc-1",
		AccountSID: "AC123456",
		From:       "+12025550100",
		To:         "+12025550199",
		Body:       "Insufficient balance test",
	}

	_, err := svc.Send(context.Background(), req)
	if err != apperrors.ErrInsufficientFunds {
		t.Fatalf("expected ErrInsufficientFunds, got %v", err)
	}
}

func TestSMSService_Send_InvalidPhoneNumber(t *testing.T) {
	repo := NewMockSMSRepo()
	billing := NewMockBillingService(map[string]int64{"acc-1": 100000})
	gateway := newTestGateway(1 * time.Millisecond)
	cfg := &config.BillingConfig{
		DefaultRatePerSMS: 7500,
		Currency:          "USD",
	}

	svc := NewService(repo, billing, gateway, cfg, logger.Default())
	defer svc.Close()

	req := &SendRequest{
		AccountID:  "acc-1",
		AccountSID: "AC123456",
		From:       "invalid-number",
		To:         "+12025550199",
		Body:       "Invalid number test",
	}

	_, err := svc.Send(context.Background(), req)
	if err != apperrors.ErrInvalidPhoneNumber {
		t.Fatalf("expected ErrInvalidPhoneNumber, got %v", err)
	}
}

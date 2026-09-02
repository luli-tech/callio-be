package sms

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/luli-tech/twilio-Boss/internal/config"
	"github.com/luli-tech/twilio-Boss/internal/domain"
	"github.com/luli-tech/twilio-Boss/pkg/logger"
)

// MockSMSRepo implements in-memory domain.SMSRepository for unit testing.
type MockSMSRepo struct {
	mu       sync.RWMutex
	messages map[string]*domain.SMSMessage
}

func NewMockSMSRepo() *MockSMSRepo {
	return &MockSMSRepo{
		messages: make(map[string]*domain.SMSMessage),
	}
}

func (m *MockSMSRepo) Create(ctx context.Context, msg *domain.SMSMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages[msg.SID] = msg
	return nil
}

func (m *MockSMSRepo) GetByID(ctx context.Context, id string) (*domain.SMSMessage, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, msg := range m.messages {
		if msg.ID == id {
			return msg, nil
		}
	}
	return nil, domain.ErrSMSNotFound
}

func (m *MockSMSRepo) GetBySID(ctx context.Context, sid string) (*domain.SMSMessage, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	msg, ok := m.messages[sid]
	if !ok {
		return nil, domain.ErrSMSNotFound
	}
	return msg, nil
}

func (m *MockSMSRepo) GetByCarrierID(ctx context.Context, carrierID string) (*domain.SMSMessage, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, msg := range m.messages {
		if msg.CarrierID == carrierID {
			return msg, nil
		}
	}
	return nil, domain.ErrSMSNotFound
}

func (m *MockSMSRepo) UpdateStatus(ctx context.Context, sid string, status domain.SMSStatus, errCode, errMsg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	msg, ok := m.messages[sid]
	if !ok {
		return domain.ErrSMSNotFound
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
		return domain.ErrSMSNotFound
	}
	msg.CarrierID = carrierID
	return nil
}

func (m *MockSMSRepo) ListByAccount(ctx context.Context, accountID string, limit, offset int) ([]*domain.SMSMessage, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []*domain.SMSMessage
	for _, msg := range m.messages {
		if msg.AccountID == accountID {
			list = append(list, msg)
		}
	}
	return list, nil
}

// MockBillingService implements domain.BillingService for unit testing.
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
		return 0, domain.ErrInsufficientFunds
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

func TestSMSService_Send_Success(t *testing.T) {
	repo := NewMockSMSRepo()
	billing := NewMockBillingService(map[string]int64{"acc-1": 100000})
	gateway := NewMockGateway(1 * time.Millisecond)
	cfg := &config.BillingConfig{
		DefaultRatePerSMS: 7500,
		Currency:          "USD",
	}

	svc := NewService(repo, billing, gateway, cfg, logger.Default())
	defer svc.Close()

	req := &domain.SendSMSRequest{
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

	if msg.Status != domain.SMSStatusSent {
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
	gateway := NewMockGateway(1 * time.Millisecond)
	cfg := &config.BillingConfig{
		DefaultRatePerSMS: 7500,
		Currency:          "USD",
	}

	svc := NewService(repo, billing, gateway, cfg, logger.Default())
	defer svc.Close()

	req := &domain.SendSMSRequest{
		AccountID:  "acc-1",
		AccountSID: "AC123456",
		From:       "+12025550100",
		To:         "+12025550199",
		Body:       "Insufficient balance test",
	}

	_, err := svc.Send(context.Background(), req)
	if err != domain.ErrInsufficientFunds {
		t.Fatalf("expected ErrInsufficientFunds, got %v", err)
	}
}

func TestSMSService_Send_InvalidPhoneNumber(t *testing.T) {
	repo := NewMockSMSRepo()
	billing := NewMockBillingService(map[string]int64{"acc-1": 100000})
	gateway := NewMockGateway(1 * time.Millisecond)
	cfg := &config.BillingConfig{
		DefaultRatePerSMS: 7500,
		Currency:          "USD",
	}

	svc := NewService(repo, billing, gateway, cfg, logger.Default())
	defer svc.Close()

	req := &domain.SendSMSRequest{
		AccountID:  "acc-1",
		AccountSID: "AC123456",
		From:       "invalid-number",
		To:         "+12025550199",
		Body:       "Invalid number test",
	}

	_, err := svc.Send(context.Background(), req)
	if err != domain.ErrInvalidPhoneNumber {
		t.Fatalf("expected ErrInvalidPhoneNumber, got %v", err)
	}
}

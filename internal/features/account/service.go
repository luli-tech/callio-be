package account

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/luli-tech/twilio-Boss/pkg/credentials"
)

type Service struct {
	accountRepo Repository
	billingSvc  BalanceReader
}

// NewService constructs account business workflows.
func NewService(accountRepo Repository, billingSvc BalanceReader) *Service {
	return &Service{
		accountRepo: accountRepo,
		billingSvc:  billingSvc,
	}
}

// GetCurrentAccount returns account details with the latest available balance.
func (s *Service) GetCurrentAccount(ctx context.Context, accountID string) (*Account, error) {
	acc, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}

	liveBalance, err := s.billingSvc.GetBalance(ctx, acc.ID)
	if err == nil {
		acc.Balance = liveBalance
	}

	return acc, nil
}

// CreateAccount creates a standalone customer tenant account.
func (s *Service) CreateAccount(ctx context.Context, req *CreateRequest) (*Account, error) {
	accountSID, authToken, err := credentials.GenerateAccountCredentials()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	acc := &Account{
		ID:        uuid.New().String(),
		SID:       accountSID,
		AuthToken: authToken,
		Name:      strings.TrimSpace(req.Name),
		Email:     strings.ToLower(strings.TrimSpace(req.Email)),
		Balance:   req.InitialBalance,
		Currency:  "USD",
		Status:    StatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.accountRepo.Create(ctx, acc); err != nil {
		return nil, err
	}

	return acc, nil
}

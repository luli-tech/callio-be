package billing

import (
	"context"

	"github.com/luli-tech/twilio-Boss/internal/features/account"
)

type Service interface {
	CheckBalance(ctx context.Context, accountID string, requiredAmount int64) (bool, error)
	DeductBalance(ctx context.Context, accountID string, amount int64, refID string, desc string) (int64, error)
	RefundBalance(ctx context.Context, accountID string, amount int64, refID string, desc string) (int64, error)
	GetBalance(ctx context.Context, accountID string) (int64, error)
}

type AccountStore interface {
	GetByID(ctx context.Context, id string) (*account.Account, error)
	UpdateBalance(ctx context.Context, accountID string, newBalance int64) error
	RecordTransaction(ctx context.Context, tx *account.Transaction) error
}

type BalanceMutation struct {
	Balance           int64
	CacheMiss         bool
	InsufficientFunds bool
}

type BalanceCache interface {
	Deduct(ctx context.Context, accountID string, amount int64) (BalanceMutation, error)
	Credit(ctx context.Context, accountID string, amount int64) (BalanceMutation, error)
	Get(ctx context.Context, accountID string) (balance int64, found bool, err error)
	Set(ctx context.Context, accountID string, balance int64) error
}

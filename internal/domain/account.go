package domain

import (
	"context"
	"time"
)

// AccountStatus represents the status of a tenant/account.
type AccountStatus string

const (
	AccountStatusActive    AccountStatus = "active"
	AccountStatusSuspended AccountStatus = "suspended"
	AccountStatusClosed    AccountStatus = "closed"
)

// Account represents a CPaaS customer tenant entity.
type Account struct {
	ID        string        `json:"id" bson:"_id"`
	SID       string        `json:"sid" bson:"sid"`      // Twilio-style public identifier e.g. ACxxxxxxxx
	AuthToken string        `json:"-" bson:"auth_token"` // Secret token used for API basic auth
	Name      string        `json:"name" bson:"name"`
	Email     string        `json:"email" bson:"email"`
	Balance   int64         `json:"balance" bson:"balance"`   // Balance in micro-units ($1.00 = 1,000,000 units to eliminate floating-point drift)
	Currency  string        `json:"currency" bson:"currency"` // USD, EUR, etc.
	Status    AccountStatus `json:"status" bson:"status"`
	CreatedAt time.Time     `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time     `json:"updated_at" bson:"updated_at"`
}

// IsActive returns true if the account is allowed to execute communications.
func (a *Account) IsActive() bool {
	return a.Status == AccountStatusActive
}

// TransactionType distinguishes credit and debit operations.
type TransactionType string

const (
	TxTypeDebit  TransactionType = "debit"
	TxTypeCredit TransactionType = "credit"
	TxTypeRefund TransactionType = "refund"
)

// Transaction represents an immutable ledger entry.
type Transaction struct {
	ID          string          `json:"id" bson:"_id"`
	AccountID   string          `json:"account_id" bson:"account_id"`
	Amount      int64           `json:"amount" bson:"amount"` // Positive integer in micro-units
	Type        TransactionType `json:"type" bson:"type"`
	ReferenceID string          `json:"reference_id" bson:"reference_id"` // ID of the SMS, Call, or Payment
	Description string          `json:"description" bson:"description"`
	CreatedAt   time.Time       `json:"created_at" bson:"created_at"`
}

// AccountRepository defines persistent storage operations for accounts.
type AccountRepository interface {
	GetByID(ctx context.Context, id string) (*Account, error)
	GetBySID(ctx context.Context, sid string) (*Account, error)
	Create(ctx context.Context, account *Account) error
	UpdateCredentials(ctx context.Context, accountID, sid, authToken string) error
	UpdateBalance(ctx context.Context, accountID string, newBalance int64) error
	RecordTransaction(ctx context.Context, tx *Transaction) error
}

// BillingService defines the business interface for atomic balance deduction and verification.
type BillingService interface {
	CheckBalance(ctx context.Context, accountID string, requiredAmount int64) (bool, error)
	DeductBalance(ctx context.Context, accountID string, amount int64, refID string, desc string) (int64, error)
	RefundBalance(ctx context.Context, accountID string, amount int64, refID string, desc string) (int64, error)
	GetBalance(ctx context.Context, accountID string) (int64, error)
}

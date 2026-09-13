package account

import (
	"context"
	"time"
)

type Status string

const (
	StatusActive    Status = "active"
	StatusSuspended Status = "suspended"
	StatusClosed    Status = "closed"
)

type Account struct {
	ID        string    `json:"id"`
	SID       string    `json:"sid"`
	AuthToken string    `json:"-"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Balance   int64     `json:"balance"`
	Currency  string    `json:"currency"`
	Status    Status    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (a *Account) IsActive() bool {
	return a.Status == StatusActive
}

type TransactionType string

const (
	TxTypeDebit  TransactionType = "debit"
	TxTypeCredit TransactionType = "credit"
	TxTypeRefund TransactionType = "refund"
)

type Transaction struct {
	ID          string          `json:"id"`
	AccountID   string          `json:"account_id"`
	Amount      int64           `json:"amount"`
	Type        TransactionType `json:"type"`
	ReferenceID string          `json:"reference_id"`
	Description string          `json:"description"`
	CreatedAt   time.Time       `json:"created_at"`
}

type CreateRequest struct {
	Name           string `json:"name" binding:"required"`
	Email          string `json:"email" binding:"required,email"`
	InitialBalance int64  `json:"initial_balance"`
}

type Repository interface {
	GetByID(ctx context.Context, id string) (*Account, error)
	GetBySID(ctx context.Context, sid string) (*Account, error)
	Create(ctx context.Context, account *Account) error
	UpdateCredentials(ctx context.Context, accountID, sid, authToken string) error
	UpdateBalance(ctx context.Context, accountID string, newBalance int64) error
	RecordTransaction(ctx context.Context, tx *Transaction) error
}

type ServiceContract interface {
	GetCurrentAccount(ctx context.Context, accountID string) (*Account, error)
	CreateAccount(ctx context.Context, req *CreateRequest) (*Account, error)
}

type BalanceReader interface {
	GetBalance(ctx context.Context, accountID string) (int64, error)
}

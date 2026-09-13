package mongodb

import (
	"time"

	"github.com/luli-tech/twilio-Boss/internal/features/account"
)

type accountModel struct {
	ID        string         `bson:"_id"`
	SID       string         `bson:"sid"`
	AuthToken string         `bson:"auth_token"`
	Name      string         `bson:"name"`
	Email     string         `bson:"email"`
	Balance   int64          `bson:"balance"`
	Currency  string         `bson:"currency"`
	Status    account.Status `bson:"status"`
	CreatedAt time.Time      `bson:"created_at"`
	UpdatedAt time.Time      `bson:"updated_at"`
}

type transactionModel struct {
	ID          string                  `bson:"_id"`
	AccountID   string                  `bson:"account_id"`
	Amount      int64                   `bson:"amount"`
	Type        account.TransactionType `bson:"type"`
	ReferenceID string                  `bson:"reference_id"`
	Description string                  `bson:"description"`
	CreatedAt   time.Time               `bson:"created_at"`
}

func accountModelFromDomain(acc *account.Account) *accountModel {
	if acc == nil {
		return nil
	}
	return &accountModel{
		ID:        acc.ID,
		SID:       acc.SID,
		AuthToken: acc.AuthToken,
		Name:      acc.Name,
		Email:     acc.Email,
		Balance:   acc.Balance,
		Currency:  acc.Currency,
		Status:    acc.Status,
		CreatedAt: acc.CreatedAt,
		UpdatedAt: acc.UpdatedAt,
	}
}

func (m *accountModel) toDomain() *account.Account {
	if m == nil {
		return nil
	}
	return &account.Account{
		ID:        m.ID,
		SID:       m.SID,
		AuthToken: m.AuthToken,
		Name:      m.Name,
		Email:     m.Email,
		Balance:   m.Balance,
		Currency:  m.Currency,
		Status:    m.Status,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func transactionModelFromDomain(tx *account.Transaction) *transactionModel {
	if tx == nil {
		return nil
	}
	return &transactionModel{
		ID:          tx.ID,
		AccountID:   tx.AccountID,
		Amount:      tx.Amount,
		Type:        tx.Type,
		ReferenceID: tx.ReferenceID,
		Description: tx.Description,
		CreatedAt:   tx.CreatedAt,
	}
}

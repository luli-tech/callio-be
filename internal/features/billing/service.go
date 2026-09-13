package billing

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/luli-tech/twilio-Boss/internal/features/account"
	"github.com/luli-tech/twilio-Boss/internal/shared/apperrors"
	"github.com/luli-tech/twilio-Boss/pkg/logger"
)

type Engine struct {
	accountStore AccountStore
	balanceCache BalanceCache
	logger       *logger.Logger
}

// NewEngine constructs a new real-time atomic Billing Engine.
func NewEngine(accountStore AccountStore, balanceCache BalanceCache, log *logger.Logger) *Engine {
	if log == nil {
		log = logger.Default()
	}
	return &Engine{
		accountStore: accountStore,
		balanceCache: balanceCache,
		logger:       log,
	}
}

// CheckBalance verifies if the account has at least requiredAmount micro-units available.
func (e *Engine) CheckBalance(ctx context.Context, accountID string, requiredAmount int64) (bool, error) {
	if requiredAmount <= 0 {
		return true, nil
	}

	bal, err := e.GetBalance(ctx, accountID)
	if err != nil {
		return false, err
	}

	return bal >= requiredAmount, nil
}

// DeductBalance atomically reduces the account's balance by amount micro-units.
func (e *Engine) DeductBalance(ctx context.Context, accountID string, amount int64, refID string, desc string) (int64, error) {
	if amount <= 0 {
		return 0, apperrors.ErrInvalidAmount
	}

	result, err := e.balanceCache.Deduct(ctx, accountID, amount)
	if err != nil {
		return 0, err
	}
	if result.CacheMiss {
		if err := e.hydrateCache(ctx, accountID); err != nil {
			return 0, err
		}
		result, err = e.balanceCache.Deduct(ctx, accountID, amount)
		if err != nil {
			return 0, fmt.Errorf("balance deduction retry failed: %w", err)
		}
	}

	if result.InsufficientFunds {
		return 0, apperrors.ErrInsufficientFunds
	}

	// Asynchronously record ledger entry and sync DB balance
	go func(remBalance int64) {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		tx := &account.Transaction{
			ID:          uuid.New().String(),
			AccountID:   accountID,
			Amount:      amount,
			Type:        account.TxTypeDebit,
			ReferenceID: refID,
			Description: desc,
			CreatedAt:   time.Now().UTC(),
		}

		if err := e.accountStore.RecordTransaction(bgCtx, tx); err != nil {
			e.logger.Error("failed to record billing transaction ledger", "account_id", accountID, "ref_id", refID, "error", err)
		}

		if err := e.accountStore.UpdateBalance(bgCtx, accountID, remBalance); err != nil {
			e.logger.Error("failed to persist balance update to db", "account_id", accountID, "error", err)
		}
	}(result.Balance)

	return result.Balance, nil
}

// RefundBalance credits back previously deducted funds.
func (e *Engine) RefundBalance(ctx context.Context, accountID string, amount int64, refID string, desc string) (int64, error) {
	if amount <= 0 {
		return 0, apperrors.ErrInvalidAmount
	}

	result, err := e.balanceCache.Credit(ctx, accountID, amount)
	if err != nil {
		return 0, err
	}
	if result.CacheMiss {
		if err := e.hydrateCache(ctx, accountID); err != nil {
			return 0, err
		}
		result, err = e.balanceCache.Credit(ctx, accountID, amount)
		if err != nil {
			return 0, fmt.Errorf("balance refund retry failed: %w", err)
		}
	}

	// Persist ledger refund
	go func(newBalance int64) {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		tx := &account.Transaction{
			ID:          uuid.New().String(),
			AccountID:   accountID,
			Amount:      amount,
			Type:        account.TxTypeRefund,
			ReferenceID: refID,
			Description: desc,
			CreatedAt:   time.Now().UTC(),
		}

		_ = e.accountStore.RecordTransaction(bgCtx, tx)
		_ = e.accountStore.UpdateBalance(bgCtx, accountID, newBalance)
	}(result.Balance)

	return result.Balance, nil
}

// GetBalance retrieves current balance in micro-units from Redis cache or persistent storage.
func (e *Engine) GetBalance(ctx context.Context, accountID string) (int64, error) {
	val, found, err := e.balanceCache.Get(ctx, accountID)
	if err != nil {
		return 0, err
	}
	if found {
		return val, nil
	}

	// Fallback to persistent storage
	acc, err := e.accountStore.GetByID(ctx, accountID)
	if err != nil {
		return 0, err
	}

	// Prime Redis
	_ = e.balanceCache.Set(ctx, accountID, acc.Balance)

	return acc.Balance, nil
}

func (e *Engine) hydrateCache(ctx context.Context, accountID string) error {
	acc, err := e.accountStore.GetByID(ctx, accountID)
	if err != nil {
		return err
	}

	return e.balanceCache.Set(ctx, accountID, acc.Balance)
}

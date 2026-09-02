package billing

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/luli-tech/twilio-Boss/internal/domain"
	"github.com/luli-tech/twilio-Boss/pkg/logger"
)

const (
	balanceKeyPrefix = "account:balance:"
	balanceKeyTTL    = 7 * 24 * time.Hour
)

// Lua script to perform atomic balance deduction in Redis.
// Returns:
// >= 0: new balance after deduction
// -1: key does not exist (cache miss, need to hydrate from DB)
// -2: insufficient funds
var deductLuaScript = redis.NewScript(`
	local balance = redis.call('GET', KEYS[1])
	if not balance then
		return -1
	end
	local current = tonumber(balance)
	local deduction = tonumber(ARGV[1])
	if current < deduction then
		return -2
	end
	local new_balance = current - deduction
	redis.call('SET', KEYS[1], new_balance)
	return new_balance
`)

// Lua script to perform atomic balance refund/credit in Redis.
var creditLuaScript = redis.NewScript(`
	local balance = redis.call('GET', KEYS[1])
	if not balance then
		return -1
	end
	local current = tonumber(balance)
	local credit = tonumber(ARGV[1])
	local new_balance = current + credit
	redis.call('SET', KEYS[1], new_balance)
	return new_balance
`)

// Engine implements domain.BillingService.
type Engine struct {
	accountRepo domain.AccountRepository
	redisClient *redis.Client
	logger      *logger.Logger
	syncMu      sync.RWMutex
}

// NewEngine constructs a new real-time atomic Billing Engine.
func NewEngine(accountRepo domain.AccountRepository, redisClient *redis.Client, log *logger.Logger) *Engine {
	if log == nil {
		log = logger.Default()
	}
	return &Engine{
		accountRepo: accountRepo,
		redisClient: redisClient,
		logger:      log,
	}
}

func (e *Engine) formatBalanceKey(accountID string) string {
	return balanceKeyPrefix + accountID
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
		return 0, domain.ErrInvalidAmount
	}

	key := e.formatBalanceKey(accountID)

	// Execute atomic Lua deduction
	result, err := deductLuaScript.Run(ctx, e.redisClient, []string{key}, amount).Int64()
	if err != nil && !errors.Is(err, redis.Nil) {
		return 0, fmt.Errorf("redis error during balance deduction: %w", err)
	}

	if result == -1 {
		// Cache miss: hydrate from persistent storage and retry
		if err := e.hydrateCache(ctx, accountID); err != nil {
			return 0, err
		}
		// Re-run deduction script once hydrated
		result, err = deductLuaScript.Run(ctx, e.redisClient, []string{key}, amount).Int64()
		if err != nil {
			return 0, fmt.Errorf("redis error during balance deduction retry: %w", err)
		}
	}

	if result == -2 {
		return 0, domain.ErrInsufficientFunds
	}

	// Asynchronously record ledger entry and sync DB balance
	go func(remBalance int64) {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		tx := &domain.Transaction{
			ID:          uuid.New().String(),
			AccountID:   accountID,
			Amount:      amount,
			Type:        domain.TxTypeDebit,
			ReferenceID: refID,
			Description: desc,
			CreatedAt:   time.Now().UTC(),
		}

		if err := e.accountRepo.RecordTransaction(bgCtx, tx); err != nil {
			e.logger.Error("failed to record billing transaction ledger", "account_id", accountID, "ref_id", refID, "error", err)
		}

		if err := e.accountRepo.UpdateBalance(bgCtx, accountID, remBalance); err != nil {
			e.logger.Error("failed to persist balance update to db", "account_id", accountID, "error", err)
		}
	}(result)

	return result, nil
}

// RefundBalance credits back previously deducted funds.
func (e *Engine) RefundBalance(ctx context.Context, accountID string, amount int64, refID string, desc string) (int64, error) {
	if amount <= 0 {
		return 0, domain.ErrInvalidAmount
	}

	key := e.formatBalanceKey(accountID)
	result, err := creditLuaScript.Run(ctx, e.redisClient, []string{key}, amount).Int64()
	if err != nil && !errors.Is(err, redis.Nil) {
		return 0, fmt.Errorf("redis error during balance refund: %w", err)
	}

	if result == -1 {
		if err := e.hydrateCache(ctx, accountID); err != nil {
			return 0, err
		}
		result, err = creditLuaScript.Run(ctx, e.redisClient, []string{key}, amount).Int64()
		if err != nil {
			return 0, fmt.Errorf("redis error during balance refund retry: %w", err)
		}
	}

	// Persist ledger refund
	go func(newBalance int64) {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		tx := &domain.Transaction{
			ID:          uuid.New().String(),
			AccountID:   accountID,
			Amount:      amount,
			Type:        domain.TxTypeRefund,
			ReferenceID: refID,
			Description: desc,
			CreatedAt:   time.Now().UTC(),
		}

		_ = e.accountRepo.RecordTransaction(bgCtx, tx)
		_ = e.accountRepo.UpdateBalance(bgCtx, accountID, newBalance)
	}(result)

	return result, nil
}

// GetBalance retrieves current balance in micro-units from Redis cache or persistent storage.
func (e *Engine) GetBalance(ctx context.Context, accountID string) (int64, error) {
	key := e.formatBalanceKey(accountID)
	val, err := e.redisClient.Get(ctx, key).Int64()
	if err == nil {
		return val, nil
	}

	if !errors.Is(err, redis.Nil) {
		return 0, fmt.Errorf("failed to read balance from redis: %w", err)
	}

	// Fallback to persistent storage
	acc, err := e.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return 0, err
	}

	// Prime Redis
	_ = e.redisClient.Set(ctx, key, acc.Balance, balanceKeyTTL).Err()

	return acc.Balance, nil
}

func (e *Engine) hydrateCache(ctx context.Context, accountID string) error {
	acc, err := e.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return err
	}

	key := e.formatBalanceKey(accountID)
	return e.redisClient.Set(ctx, key, acc.Balance, balanceKeyTTL).Err()
}

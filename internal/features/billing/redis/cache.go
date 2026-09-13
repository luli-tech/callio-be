package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/luli-tech/twilio-Boss/internal/features/billing"
	redisclient "github.com/redis/go-redis/v9"
)

const (
	balanceKeyPrefix = "account:balance:"
	balanceKeyTTL    = 7 * 24 * time.Hour
)

var deductLuaScript = redisclient.NewScript(`
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

var creditLuaScript = redisclient.NewScript(`
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

type BalanceCache struct {
	client *redisclient.Client
}

func NewBalanceCache(client *redisclient.Client) *BalanceCache {
	return &BalanceCache{client: client}
}

func (c *BalanceCache) Deduct(ctx context.Context, accountID string, amount int64) (billing.BalanceMutation, error) {
	result, err := deductLuaScript.Run(ctx, c.client, []string{c.key(accountID)}, amount).Int64()
	if err != nil && !errors.Is(err, redisclient.Nil) {
		return billing.BalanceMutation{}, fmt.Errorf("redis error during balance deduction: %w", err)
	}
	return mutationFromRedisResult(result), nil
}

func (c *BalanceCache) Credit(ctx context.Context, accountID string, amount int64) (billing.BalanceMutation, error) {
	result, err := creditLuaScript.Run(ctx, c.client, []string{c.key(accountID)}, amount).Int64()
	if err != nil && !errors.Is(err, redisclient.Nil) {
		return billing.BalanceMutation{}, fmt.Errorf("redis error during balance refund: %w", err)
	}
	return mutationFromRedisResult(result), nil
}

func (c *BalanceCache) Get(ctx context.Context, accountID string) (int64, bool, error) {
	balance, err := c.client.Get(ctx, c.key(accountID)).Int64()
	if err == nil {
		return balance, true, nil
	}
	if errors.Is(err, redisclient.Nil) {
		return 0, false, nil
	}
	return 0, false, fmt.Errorf("failed to read balance from redis: %w", err)
}

func (c *BalanceCache) Set(ctx context.Context, accountID string, balance int64) error {
	return c.client.Set(ctx, c.key(accountID), balance, balanceKeyTTL).Err()
}

func (c *BalanceCache) key(accountID string) string {
	return balanceKeyPrefix + accountID
}

func mutationFromRedisResult(result int64) billing.BalanceMutation {
	switch result {
	case -1:
		return billing.BalanceMutation{CacheMiss: true}
	case -2:
		return billing.BalanceMutation{InsufficientFunds: true}
	default:
		return billing.BalanceMutation{Balance: result}
	}
}

package idempotency

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Sentinel errors for idempotency operations.
var (
	ErrLockConflict = errors.New("request with this idempotency key is already in progress")
	ErrKeyNotFound  = errors.New("idempotency key not found")
)

const (
	// StatusProcessing is written during initial request locking.
	StatusProcessing = "PROCESSING"
	// KeyPrefix is prepended to all Redis idempotency keys.
	KeyPrefix = "idempotency:"
)

// CachedResponse stores completed HTTP response metadata and body.
type CachedResponse struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       []byte            `json:"body"`
	CreatedAt  time.Time         `json:"created_at"`
}

// LockResult communicates the state of an acquired idempotency key.
type LockResult struct {
	IsNew          bool
	CachedResponse *CachedResponse
}

// Manager coordinates distributed idempotency locking and response caching.
type Manager struct {
	redisClient *redis.Client
	defaultTTL  time.Duration
}

// NewManager creates a new Idempotency Manager instance.
func NewManager(client *redis.Client, defaultTTL time.Duration) *Manager {
	if defaultTTL <= 0 {
		defaultTTL = 24 * time.Hour
	}
	return &Manager{
		redisClient: client,
		defaultTTL:  defaultTTL,
	}
}

func (m *Manager) formatKey(key string) string {
	return KeyPrefix + key
}

// Acquire attempts to atomically lock the idempotency key using Redis SETNX.
// Returns LockResult with IsNew=true if newly acquired, or CachedResponse if already executed.
func (m *Manager) Acquire(ctx context.Context, key string, ttl time.Duration) (*LockResult, error) {
	if key == "" {
		return nil, errors.New("idempotency key is required")
	}

	if ttl <= 0 {
		ttl = m.defaultTTL
	}

	redisKey := m.formatKey(key)

	// Attempt atomic SETNX with processing marker
	success, err := m.redisClient.SetNX(ctx, redisKey, StatusProcessing, ttl).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to acquire idempotency lock: %w", err)
	}

	if success {
		return &LockResult{IsNew: true}, nil
	}

	// Key already exists, fetch current value
	val, err := m.redisClient.Get(ctx, redisKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			// Lock expired between SetNX and Get, retry recursively once
			return m.Acquire(ctx, key, ttl)
		}
		return nil, fmt.Errorf("failed to check existing idempotency key: %w", err)
	}

	if val == StatusProcessing {
		return nil, ErrLockConflict
	}

	// Unmarshal completed response payload
	var cached CachedResponse
	if err := json.Unmarshal([]byte(val), &cached); err != nil {
		return nil, fmt.Errorf("corrupted idempotency cache data: %w", err)
	}

	return &LockResult{
		IsNew:          false,
		CachedResponse: &cached,
	}, nil
}

// SaveResponse stores the final HTTP response for the idempotency key.
func (m *Manager) SaveResponse(ctx context.Context, key string, resp *CachedResponse, ttl time.Duration) error {
	if key == "" || resp == nil {
		return errors.New("invalid arguments to SaveResponse")
	}

	if ttl <= 0 {
		ttl = m.defaultTTL
	}

	resp.CreatedAt = time.Now().UTC()
	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("failed to serialize cached response: %w", err)
	}

	redisKey := m.formatKey(key)
	if err := m.redisClient.Set(ctx, redisKey, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to save idempotency response to redis: %w", err)
	}

	return nil
}

// Release removes the lock (typically on unhandled server error so client can retry safely).
func (m *Manager) Release(ctx context.Context, key string) error {
	if key == "" {
		return nil
	}
	redisKey := m.formatKey(key)
	return m.redisClient.Del(ctx, redisKey).Err()
}


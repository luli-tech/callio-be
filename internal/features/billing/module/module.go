package module

import (
	"github.com/luli-tech/twilio-Boss/internal/features/account"
	"github.com/luli-tech/twilio-Boss/internal/features/billing"
	billingRedis "github.com/luli-tech/twilio-Boss/internal/features/billing/redis"
	"github.com/luli-tech/twilio-Boss/pkg/logger"
	"github.com/redis/go-redis/v9"
)

type Module struct {
	service billing.Service
}

func New(accountRepo account.Repository, redisClient *redis.Client, log *logger.Logger) *Module {
	return &Module{
		service: billing.NewEngine(accountRepo, billingRedis.NewBalanceCache(redisClient), log),
	}
}

func (m *Module) Service() billing.Service {
	return m.service
}

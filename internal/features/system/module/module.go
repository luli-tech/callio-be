package module

import (
	"github.com/gin-gonic/gin"
	"github.com/luli-tech/twilio-Boss/internal/features/system"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Module struct {
	controller *system.HealthController
}

func New(db *mongo.Database, redisClient *redis.Client) *Module {
	return &Module{
		controller: system.NewHealthController(db, redisClient),
	}
}

func (m *Module) RegisterRoutes(engine *gin.Engine) {
	system.RegisterRoutes(engine, m.controller)
}

package system

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// HealthController handles health check probes.
type HealthController struct {
	db          *mongo.Database
	redisClient *redis.Client
}

// NewHealthController creates a new HealthController.
func NewHealthController(db *mongo.Database, redisClient *redis.Client) *HealthController {
	return &HealthController{
		db:          db,
		redisClient: redisClient,
	}
}

// Check evaluates subsystem health (MongoDB, Redis) and returns status.
func (h *HealthController) Check(c *gin.Context) {
	status := "healthy"
	dbStatus := "up"
	redisStatus := "up"

	if h.db != nil {
		if err := h.db.RunCommand(c.Request.Context(), bson.D{{Key: "ping", Value: 1}}).Err(); err != nil {
			dbStatus = "down"
			status = "degraded"
		}
	}

	if h.redisClient != nil {
		if err := h.redisClient.Ping(c.Request.Context()).Err(); err != nil {
			redisStatus = "down"
			status = "degraded"
		}
	}

	statusCode := http.StatusOK
	if status == "degraded" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, gin.H{
		"status":    status,
		"timestamp": time.Now().UTC(),
		"components": gin.H{
			"mongodb": dbStatus,
			"redis":   redisStatus,
		},
	})
}

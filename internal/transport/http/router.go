package http

import (
	"github.com/gin-gonic/gin"
	"github.com/luli-tech/twilio-Boss/internal/config"
	"github.com/luli-tech/twilio-Boss/internal/domain"
	"github.com/luli-tech/twilio-Boss/internal/transport/http/middleware"
	v1 "github.com/luli-tech/twilio-Boss/internal/transport/http/v1"
	"github.com/luli-tech/twilio-Boss/pkg/idempotency"
	"github.com/luli-tech/twilio-Boss/pkg/logger"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// RouterConfig contains all controller and middleware dependencies for routing.
type RouterConfig struct {
	Config         *config.Config
	Logger         *logger.Logger
	DB             *mongo.Database
	RedisClient    *redis.Client
	AccountRepo    domain.AccountRepository
	BillingService domain.BillingService
	AuthService    domain.AuthService
	UserService    domain.UserService
	SMSService     domain.SMSService
	IdempotencyMgr *idempotency.Manager
}

// NewRouter builds and configures the Gin HTTP Engine.
func NewRouter(cfg RouterConfig) *gin.Engine {
	if cfg.Config.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	engine := gin.New()

	// Global Infrastructure Middlewares
	engine.Use(middleware.RequestLogger(cfg.Logger))
	engine.Use(middleware.Recovery(cfg.Logger))
	engine.Use(middleware.Metrics())

	// Instantiate Controllers
	healthCtrl := v1.NewHealthController(cfg.DB, cfg.RedisClient)
	accountCtrl := v1.NewAccountController(cfg.AccountRepo, cfg.BillingService)
	authCtrl := v1.NewAuthController(cfg.AuthService)
	userCtrl := v1.NewUserController(cfg.UserService)
	smsCtrl := v1.NewSMSController(cfg.SMSService)

	// Public Routes
	engine.GET("/health", healthCtrl.Check)
	engine.GET("/metrics", middleware.PrometheusHandler())

	// API v1 Group
	v1Group := engine.Group("/v1")
	{
		// Public tenant registration
		v1Group.POST("/accounts", accountCtrl.CreateAccount)
		v1Group.POST("/auth/register", authCtrl.Register)
		v1Group.POST("/auth/login", authCtrl.Login)
		v1Group.POST("/auth/refresh", authCtrl.Refresh)
		v1Group.POST("/sms/dlr", smsCtrl.DeliveryReport)

		// Authenticated & Protected Subsystems
		protected := v1Group.Group("")
		protected.Use(middleware.Auth(cfg.AccountRepo, cfg.AuthService))
		protected.Use(middleware.RateLimiter(cfg.RedisClient, cfg.Config.RateLimit.RequestsPerMinute))
		protected.Use(middleware.Idempotency(cfg.IdempotencyMgr, cfg.Config.Idempotency.DefaultTTL))
		{
			// Account info
			protected.GET("/accounts/me", accountCtrl.GetCurrentAccount)
			protected.POST("/auth/account/credentials/rotate", authCtrl.RotateCredentials)

			// User APIs
			protected.GET("/users", userCtrl.List)
			protected.POST("/users", userCtrl.Create)
			protected.GET("/users/me", userCtrl.Me)
			protected.PATCH("/users/me", userCtrl.UpdateMe)
			protected.PATCH("/users/:id/role", userCtrl.UpdateRole)
			protected.PATCH("/users/:id/status", userCtrl.UpdateStatus)
			protected.DELETE("/users/:id", userCtrl.Delete)

			// SMS APIs
			protected.POST("/sms/send", smsCtrl.SendSMS)
			protected.GET("/sms/:sid", smsCtrl.GetSMS)
		}
	}

	return engine
}

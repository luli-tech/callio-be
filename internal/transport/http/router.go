package http

import (
	"github.com/gin-gonic/gin"
	"github.com/luli-tech/twilio-Boss/internal/config"
	accountModule "github.com/luli-tech/twilio-Boss/internal/features/account/module"
	authModule "github.com/luli-tech/twilio-Boss/internal/features/auth/module"
	smsModule "github.com/luli-tech/twilio-Boss/internal/features/sms/module"
	systemModule "github.com/luli-tech/twilio-Boss/internal/features/system/module"
	userModule "github.com/luli-tech/twilio-Boss/internal/features/user/module"
	"github.com/luli-tech/twilio-Boss/internal/transport/http/middleware"
	"github.com/luli-tech/twilio-Boss/pkg/idempotency"
	"github.com/luli-tech/twilio-Boss/pkg/logger"
	"github.com/redis/go-redis/v9"
)

// RouterConfig contains all controller and middleware dependencies for routing.
type RouterConfig struct {
	Config      *config.Config
	Logger      *logger.Logger
	RedisClient *redis.Client
	Account     *accountModule.Module
	Auth        *authModule.Module
	User        *userModule.Module
	SMS         *smsModule.Module
	System      *systemModule.Module
	Idempotency *idempotency.Manager
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
	engine.Use(middleware.CORS())
	engine.Use(middleware.RequestLogger(cfg.Logger))
	engine.Use(middleware.Recovery(cfg.Logger))
	engine.Use(middleware.Metrics())

	cfg.System.RegisterRoutes(engine)
	engine.GET("/metrics", middleware.PrometheusHandler())

	v1Group := engine.Group("/v1")
	{
		cfg.Account.RegisterPublicRoutes(v1Group)
		cfg.Auth.RegisterPublicRoutes(v1Group)
		cfg.SMS.RegisterPublicRoutes(v1Group)

		protected := v1Group.Group("")
		protected.Use(middleware.Auth(cfg.Account.Repository(), cfg.Auth.Service()))
		protected.Use(middleware.RateLimiter(cfg.RedisClient, cfg.Config.RateLimit.RequestsPerMinute))
		protected.Use(middleware.Idempotency(cfg.Idempotency, cfg.Config.Idempotency.DefaultTTL))
		{
			cfg.Account.RegisterProtectedRoutes(protected)
			cfg.Auth.RegisterProtectedRoutes(protected)
			cfg.User.RegisterProtectedRoutes(protected)
			cfg.SMS.RegisterProtectedRoutes(protected)
		}
	}

	return engine
}

package app

import (
	"github.com/gin-gonic/gin"
	"github.com/luli-tech/twilio-Boss/internal/config"
	accountModule "github.com/luli-tech/twilio-Boss/internal/features/account/module"
	authModule "github.com/luli-tech/twilio-Boss/internal/features/auth/module"
	billingModule "github.com/luli-tech/twilio-Boss/internal/features/billing/module"
	smsModule "github.com/luli-tech/twilio-Boss/internal/features/sms/module"
	systemModule "github.com/luli-tech/twilio-Boss/internal/features/system/module"
	userModule "github.com/luli-tech/twilio-Boss/internal/features/user/module"
	httpTransport "github.com/luli-tech/twilio-Boss/internal/transport/http"
	"github.com/luli-tech/twilio-Boss/pkg/idempotency"
	"github.com/luli-tech/twilio-Boss/pkg/logger"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Dependencies struct {
	Config      *config.Config
	Logger      *logger.Logger
	DB          *mongo.Database
	RedisClient *redis.Client
}

type App struct {
	Engine *gin.Engine
	close  func()
}

func New(deps Dependencies) *App {
	idempotencyMgr := idempotency.NewManager(deps.RedisClient, deps.Config.Idempotency.DefaultTTL)

	accountRepo := accountModule.NewRepository(deps.DB)
	userRepo := userModule.NewRepository(deps.DB)
	smsRepo := smsModule.NewRepository(deps.DB)

	billingFeature := billingModule.New(accountRepo, deps.RedisClient, deps.Logger)
	accountFeature := accountModule.New(accountRepo, billingFeature.Service())
	userFeature := userModule.New(userRepo)
	authFeature := authModule.New(accountRepo, userRepo, deps.Config.JWT)
	smsFeature := smsModule.New(smsRepo, billingFeature.Service(), deps.Config, deps.Logger)
	systemFeature := systemModule.New(deps.DB, deps.RedisClient)

	engine := httpTransport.NewRouter(httpTransport.RouterConfig{
		Config:      deps.Config,
		Logger:      deps.Logger,
		RedisClient: deps.RedisClient,
		Account:     accountFeature,
		Auth:        authFeature,
		User:        userFeature,
		SMS:         smsFeature,
		System:      systemFeature,
		Idempotency: idempotencyMgr,
	})

	return &App{
		Engine: engine,
		close:  smsFeature.Close,
	}
}

func (a *App) Close() {
	if a.close != nil {
		a.close()
	}
}

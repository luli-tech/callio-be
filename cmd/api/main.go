package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/luli-tech/twilio-Boss/internal/config"
	"github.com/luli-tech/twilio-Boss/internal/domain"
	"github.com/luli-tech/twilio-Boss/internal/repository/mongodb"
	redisRepo "github.com/luli-tech/twilio-Boss/internal/repository/redis"
	"github.com/luli-tech/twilio-Boss/internal/service/auth"
	"github.com/luli-tech/twilio-Boss/internal/service/billing"
	"github.com/luli-tech/twilio-Boss/internal/service/sms"
	"github.com/luli-tech/twilio-Boss/internal/service/user"
	"github.com/luli-tech/twilio-Boss/internal/service/voice"
	httpTransport "github.com/luli-tech/twilio-Boss/internal/transport/http"
	"github.com/luli-tech/twilio-Boss/pkg/eslclient"
	"github.com/luli-tech/twilio-Boss/pkg/idempotency"
	"github.com/luli-tech/twilio-Boss/pkg/logger"
)

func main() {
	// 1. Load System Configuration
	cfg := config.Load()

	// 2. Initialize Structured Logger
	log := logger.New(logger.Config{
		Level:  "info",
		Format: "json",
		Output: os.Stdout,
	})

	log.Info("Starting CPaaS Communications Backend",
		"app", cfg.App.Name,
		"env", cfg.App.Env,
		"port", cfg.App.Port,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 3. Initialize MongoDB
	mongoClient, db, err := mongodb.NewDB(ctx, &cfg.MongoDB)
	if err != nil {
		log.Warn("could not connect to mongodb (running in offline/mock mode)", "error", err)
	} else {
		log.Info("MongoDB connection established")
		defer mongoClient.Disconnect(context.Background())
	}

	// 4. Initialize Redis Client Pool
	redisClient, err := redisRepo.NewClient(ctx, &cfg.Redis)
	if err != nil {
		log.Warn("could not connect to redis (running in degraded mode)", "error", err)
	} else {
		log.Info("Redis connection pool established")
		defer redisClient.Close()
	}

	// 5. Initialize FreeSWITCH ESL Client
	eslClient := eslclient.NewClient(eslclient.Config{
		Host:     cfg.FreeSWITCH.Host,
		Port:     cfg.FreeSWITCH.Port,
		Password: cfg.FreeSWITCH.Password,
		Timeout:  cfg.FreeSWITCH.Timeout,
	})
	defer eslClient.Close()

	// 6. Initialize Repositories
	accountRepo := mongodb.NewAccountRepo(db)
	userRepo := mongodb.NewUserRepo(db)
	smsRepo := mongodb.NewSMSRepo(db)
	callRepo := mongodb.NewCallRepo(db)

	// 7. Initialize Idempotency Manager
	idempotencyMgr := idempotency.NewManager(redisClient, cfg.Idempotency.DefaultTTL)

	// 8. Initialize Domain Services
	billingEngine := billing.NewEngine(accountRepo, redisClient, log)
	userService := user.NewService(userRepo)
	authService := auth.NewService(accountRepo, userRepo, cfg.JWT)
	var smsGateway domain.SMSGateway = sms.NewMockGateway(50 * time.Millisecond)
	if strings.EqualFold(cfg.SMSGateway.Provider, "jasmin") {
		smsGateway = sms.NewJasminGateway(cfg.SMSGateway)
	}
	smsService := sms.NewService(smsRepo, billingEngine, smsGateway, &cfg.Billing, log)
	defer smsService.Close()

	voiceEngine := voice.NewEngine(eslClient, callRepo, billingEngine, log)
	_ = userService
	_ = authService
	_ = voiceEngine

	// 9. Assemble HTTP Transport Router
	engine := httpTransport.NewRouter(httpTransport.RouterConfig{
		Config:         cfg,
		Logger:         log,
		DB:             db,
		RedisClient:    redisClient,
		AccountRepo:    accountRepo,
		BillingService: billingEngine,
		AuthService:    authService,
		UserService:    userService,
		SMSService:     smsService,
		IdempotencyMgr: idempotencyMgr,
	})

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.App.Port),
		Handler:      engine,
		ReadTimeout:  cfg.App.ReadTimeout,
		WriteTimeout: cfg.App.WriteTimeout,
		IdleTimeout:  cfg.App.IdleTimeout,
	}

	// 10. Start HTTP Server in background goroutine
	go func() {
		log.Info("HTTP server listening", "addr", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("HTTP server listener failed", "error", err)
			os.Exit(1)
		}
	}()

	// 11. Graceful Shutdown on OS Signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	sig := <-quit
	log.Info("Received shutdown signal", "signal", sig.String())

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.App.ShutdownTimeout)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error("HTTP server graceful shutdown failed", "error", err)
	} else {
		log.Info("HTTP server stopped gracefully")
	}

	log.Info("CPaaS backend shutdown complete")
}

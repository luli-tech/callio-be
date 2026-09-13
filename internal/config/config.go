package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config aggregates all subsystem configurations.
type Config struct {
	App         AppConfig
	MongoDB     MongoDBConfig
	Redis       RedisConfig
	SMSGateway  SMSGatewayConfig
	Billing     BillingConfig
	RateLimit   RateLimitConfig
	Idempotency IdempotencyConfig
	JWT         JWTConfig
}

// JWTConfig defines JSON Web Token signing keys and expiration lifetimes.
type JWTConfig struct {
	Secret          string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	Issuer          string
}

// AppConfig defines HTTP server and general runtime settings.
type AppConfig struct {
	Name            string
	Env             string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

// MongoDBConfig defines MongoDB database connection options.
type MongoDBConfig struct {
	URI            string
	Database       string
	ConnectTimeout time.Duration
}

// RedisConfig defines Redis connection options.
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
	PoolSize int
}

// SMSGatewayConfig defines upstream SMS carrier gateway options.
type SMSGatewayConfig struct {
	Provider string
	BaseURL  string
	Username string
	Password string
	DLRURL   string
	Timeout  time.Duration
}

// Addr returns the host:port string for Redis.
func (r *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// BillingConfig defines pricing defaults in micro-units ($1.00 = 1,000,000 micro-units).
type BillingConfig struct {
	DefaultRatePerSMS int64 // e.g., 7500 micro-units = $0.0075 / SMS
	Currency          string
}

// RateLimitConfig defines API rate limiter parameters.
type RateLimitConfig struct {
	RequestsPerMinute int
	Burst             int
}

// IdempotencyConfig defines lock and cache expiration.
type IdempotencyConfig struct {
	DefaultTTL time.Duration
}

// Load parses environment variables with secure, sensible defaults.
func Load() *Config {
	return &Config{
		App: AppConfig{
			Name:            getEnv("APP_NAME", "callio-be"),
			Env:             getEnv("APP_ENV", "development"),
			Port:            getEnvAsInt("PORT", 8080),
			ReadTimeout:     getEnvAsDuration("APP_READ_TIMEOUT", 10*time.Second),
			WriteTimeout:    getEnvAsDuration("APP_WRITE_TIMEOUT", 10*time.Second),
			IdleTimeout:     getEnvAsDuration("APP_IDLE_TIMEOUT", 60*time.Second),
			ShutdownTimeout: getEnvAsDuration("APP_SHUTDOWN_TIMEOUT", 10*time.Second),
		},
		MongoDB: MongoDBConfig{
			URI:            getEnv("MONGODB_URI", "mongodb://localhost:27017"),
			Database:       getEnv("MONGODB_DATABASE", "callio"),
			ConnectTimeout: getEnvAsDuration("MONGODB_CONNECT_TIMEOUT", 3*time.Second),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnvAsInt("REDIS_PORT", 6379),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
			PoolSize: getEnvAsInt("REDIS_POOL_SIZE", 50),
		},
		SMSGateway: SMSGatewayConfig{
			Provider: getEnv("SMS_GATEWAY_PROVIDER", "mock"),
			BaseURL:  getEnv("JASMIN_BASE_URL", "http://localhost:8080"),
			Username: getEnv("JASMIN_USERNAME", ""),
			Password: getEnv("JASMIN_PASSWORD", ""),
			DLRURL:   getEnv("JASMIN_DLR_URL", ""),
			Timeout:  getEnvAsDuration("SMS_GATEWAY_TIMEOUT", 10*time.Second),
		},
		Billing: BillingConfig{
			DefaultRatePerSMS: getEnvAsInt64("BILLING_RATE_PER_SMS", 7500), // $0.0075
			Currency:          getEnv("BILLING_CURRENCY", "USD"),
		},
		RateLimit: RateLimitConfig{
			RequestsPerMinute: getEnvAsInt("RATE_LIMIT_RPM", 600),
			Burst:             getEnvAsInt("RATE_LIMIT_BURST", 50),
		},
		Idempotency: IdempotencyConfig{
			DefaultTTL: getEnvAsDuration("IDEMPOTENCY_TTL", 24*time.Hour),
		},
		JWT: JWTConfig{
			Secret:          getEnv("JWT_SECRET", "super-secret-cpaas-jwt-signing-key-change-in-production"),
			AccessTokenTTL:  getEnvAsDuration("JWT_ACCESS_TOKEN_TTL", 1*time.Hour),
			RefreshTokenTTL: getEnvAsDuration("JWT_REFRESH_TOKEN_TTL", 7*24*time.Hour),
			Issuer:          getEnv("JWT_ISSUER", "callio-cpaas"),
		},
	}
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	if val, ok := os.LookupEnv(key); ok {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func getEnvAsInt64(key string, defaultVal int64) int64 {
	if val, ok := os.LookupEnv(key); ok {
		if intVal, err := strconv.ParseInt(val, 10, 64); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func getEnvAsDuration(key string, defaultVal time.Duration) time.Duration {
	if val, ok := os.LookupEnv(key); ok {
		if durationVal, err := time.ParseDuration(val); err == nil {
			return durationVal
		}
	}
	return defaultVal
}

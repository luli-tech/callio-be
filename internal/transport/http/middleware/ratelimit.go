package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimiter enforces a sliding window rate limit using Redis.
func RateLimiter(client *redis.Client, requestsPerMinute int) gin.HandlerFunc {
	if requestsPerMinute <= 0 {
		requestsPerMinute = 600
	}

	return func(c *gin.Context) {
		var identifier string
		if acc := GetAccount(c); acc != nil {
			identifier = "acc:" + acc.SID
		} else {
			identifier = "ip:" + c.ClientIP()
		}

		key := fmt.Sprintf("ratelimit:%s:%d", identifier, time.Now().Unix()/60)
		ctx := c.Request.Context()

		pipe := client.Pipeline()
		incrCmd := pipe.Incr(ctx, key)
		pipe.Expire(ctx, key, 70*time.Second)

		if _, err := pipe.Exec(ctx); err != nil {
			// Fail open on Redis error so as not to block critical communications
			c.Next()
			return
		}

		count := incrCmd.Val()
		if count > int64(requestsPerMinute) {
			c.Header("Retry-After", "60")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code":    "RATE_LIMIT_EXCEEDED",
				"message": "Too many requests. Please throttle your traffic.",
			})
			return
		}

		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", requestsPerMinute))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", int64(requestsPerMinute)-count))
		c.Next()
	}
}

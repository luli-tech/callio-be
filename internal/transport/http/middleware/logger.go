package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/luli-tech/twilio-Boss/pkg/logger"
)

// RequestLogger logs incoming HTTP requests with latency, status code, and trace ID.
func RequestLogger(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		traceID := c.GetHeader("X-Request-ID")
		if traceID == "" {
			traceID = uuid.New().String()
		}
		c.Header("X-Request-ID", traceID)
		c.Set(string(logger.TraceIDKey), traceID)

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		path := c.Request.URL.Path

		accountSID := ""
		if acc := GetAccount(c); acc != nil {
			accountSID = acc.SID
		}

		ctxLogger := log.WithContext(c.Request.Context())
		ctxLogger.Info("http request",
			"method", method,
			"path", path,
			"status", status,
			"latency_ms", latency.Milliseconds(),
			"client_ip", clientIP,
			"account_sid", accountSID,
			"trace_id", traceID,
		)
	}
}

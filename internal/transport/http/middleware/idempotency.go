package middleware

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/luli-tech/twilio-Boss/pkg/idempotency"
)

// responseBodyWriter captures the response body and status code for caching.
type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseBodyWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *responseBodyWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

// Idempotency enforces request idempotency for state-mutating operations.
func Idempotency(mgr *idempotency.Manager, ttl time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only check POST, PUT, PATCH, DELETE
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodOptions || c.Request.Method == http.MethodHead {
			c.Next()
			return
		}

		key := c.GetHeader("Idempotency-Key")
		if key == "" {
			c.Next()
			return
		}

		// Scope idempotency key by tenant account (if authenticated) and endpoint path
		var scope string
		if acc := GetAccount(c); acc != nil {
			scope = fmt.Sprintf("%s:%s:%s", acc.SID, c.Request.URL.Path, key)
		} else {
			scope = fmt.Sprintf("anon:%s:%s", c.Request.URL.Path, key)
		}

		lockRes, err := mgr.Acquire(c.Request.Context(), scope, ttl)
		if err != nil {
			if errors.Is(err, idempotency.ErrLockConflict) {
				c.AbortWithStatusJSON(http.StatusConflict, gin.H{
					"code":    "CONCURRENT_REQUEST",
					"message": "A request with this Idempotency-Key is currently being processed",
				})
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"code":    "IDEMPOTENCY_ERROR",
				"message": "Failed to process idempotency lock",
			})
			return
		}

		// Replay cached response if already completed
		if !lockRes.IsNew && lockRes.CachedResponse != nil {
			cached := lockRes.CachedResponse
			for k, v := range cached.Headers {
				c.Header(k, v)
			}
			c.Header("X-Cache-Lookup", "HIT-IDEMPOTENT")
			c.Data(cached.StatusCode, "application/json; charset=utf-8", cached.Body)
			c.Abort()
			return
		}

		// Intercept writer for new request
		bw := &responseBodyWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBufferString(""),
		}
		c.Writer = bw

		c.Next()

		statusCode := c.Writer.Status()
		if statusCode >= 200 && statusCode < 300 {
			// Save response for replay
			headers := make(map[string]string)
			for k, v := range c.Writer.Header() {
				if len(v) > 0 {
					headers[k] = v[0]
				}
			}

			_ = mgr.SaveResponse(c.Request.Context(), scope, &idempotency.CachedResponse{
				StatusCode: statusCode,
				Headers:    headers,
				Body:       bw.body.Bytes(),
			}, ttl)
		} else if statusCode >= 500 {
			// Release lock on server failure so client can retry
			_ = mgr.Release(c.Request.Context(), scope)
		}
	}
}


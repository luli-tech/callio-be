package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	metricsMu       sync.RWMutex
	requestCounters = map[string]int64{}
	requestLatency  = map[string]float64{}
)

// Metrics records request counters and latency histograms for Prometheus.
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		status := strconv.Itoa(c.Writer.Status())
		key := metricsKey(c.Request.Method, path, status)

		metricsMu.Lock()
		requestCounters[key]++
		requestLatency[key] += time.Since(start).Seconds()
		metricsMu.Unlock()
	}
}

// PrometheusHandler emits a minimal Prometheus-compatible text exposition.
func PrometheusHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		metricsMu.RLock()
		defer metricsMu.RUnlock()

		c.Header("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		c.String(http.StatusOK, "# HELP callio_http_requests_total Total HTTP requests processed by the API.\n")
		c.Writer.WriteString("# TYPE callio_http_requests_total counter\n")
		for key, count := range requestCounters {
			method, path, status := splitMetricsKey(key)
			c.Writer.WriteString(fmt.Sprintf("callio_http_requests_total{method=%q,path=%q,status=%q} %d\n", method, path, status, count))
		}

		c.Writer.WriteString("# HELP callio_http_request_duration_seconds_sum Total HTTP request latency in seconds.\n")
		c.Writer.WriteString("# TYPE callio_http_request_duration_seconds_sum counter\n")
		for key, sum := range requestLatency {
			method, path, status := splitMetricsKey(key)
			c.Writer.WriteString(fmt.Sprintf("callio_http_request_duration_seconds_sum{method=%q,path=%q,status=%q} %.6f\n", method, path, status, sum))
		}
	}
}

func metricsKey(method, path, status string) string {
	return method + "\x00" + path + "\x00" + status
}

func splitMetricsKey(key string) (string, string, string) {
	parts := [3]string{}
	idx := 0
	start := 0
	for i, r := range key {
		if r == '\x00' && idx < len(parts)-1 {
			parts[idx] = key[start:i]
			idx++
			start = i + 1
		}
	}
	parts[idx] = key[start:]
	return parts[0], parts[1], parts[2]
}

package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
)

type ctxKey string

const (
	TraceIDKey    ctxKey = "trace_id"
	AccountSIDKey ctxKey = "account_sid"
)

// Logger wraps slog.Logger with enterprise contextual utilities.
type Logger struct {
	*slog.Logger
}

// Config defines logger configuration options.
type Config struct {
	Level  string
	Format string // "json" or "text"
	Output io.Writer
}

// New creates and configures a structured Logger.
func New(cfg Config) *Logger {
	var level slog.Level
	switch strings.ToLower(cfg.Level) {
	case "debug":
		level = slog.LevelDebug
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	}

	out := cfg.Output
	if out == nil {
		out = os.Stdout
	}

	var handler slog.Handler
	if strings.ToLower(cfg.Format) == "text" {
		handler = slog.NewTextHandler(out, opts)
	} else {
		handler = slog.NewJSONHandler(out, opts)
	}

	return &Logger{
		Logger: slog.New(handler),
	}
}

// Default returns a standard production JSON logger.
func Default() *Logger {
	return New(Config{
		Level:  "info",
		Format: "json",
		Output: os.Stdout,
	})
}

// WithContext returns a logger instance enriched with contextual attributes (TraceID, AccountSID).
func (l *Logger) WithContext(ctx context.Context) *Logger {
	if ctx == nil {
		return l
	}

	var attrs []any
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok && traceID != "" {
		attrs = append(attrs, slog.String("trace_id", traceID))
	}
	if accountSID, ok := ctx.Value(AccountSIDKey).(string); ok && accountSID != "" {
		attrs = append(attrs, slog.String("account_sid", accountSID))
	}

	if len(attrs) == 0 {
		return l
	}

	return &Logger{
		Logger: l.With(attrs...),
	}
}


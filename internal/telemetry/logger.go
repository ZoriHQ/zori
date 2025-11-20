package telemetry

import (
	"context"
	"os"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type contextKey string

const loggerKey contextKey = "logger"

// Logger is a type alias for zap.Logger to avoid importing zap everywhere
type Logger = zap.Logger

// Field is a type alias for zap.Field
type Field = zap.Field

var (
	// String is a type alias for zap.String
	String = zap.String
	// Int is a type alias for zap.Int
	Int = zap.Int
	// Bool is a type alias for zap.Bool
	Bool = zap.Bool
	// Error is a type alias for zap.Error
	Error = zap.Error
	// Any is a type alias for zap.Any
	Any = zap.Any
)

// NewLogger creates a new Zap logger configuration
func NewLogger() (*Logger, error) {
	config := zap.NewProductionConfig()
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	
	// Check if running in development
	if os.Getenv("APP_ENV") == "development" {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	return config.Build()
}

// WithContext adds the logger to the context
func WithContext(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// FromContext retrieves the logger from the context.
// If no logger is found, it returns a global default logger.
// It also enriches the logger with tracing information if available.
func FromContext(ctx context.Context) *Logger {
	logger, ok := ctx.Value(loggerKey).(*Logger)
	if !ok {
		// Fallback to a default logger if none is in context
		// In a real app, you might want to initialize a global default
		l, _ := zap.NewProduction()
		logger = l
	}

	// Enrich with tracing info
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		logger = logger.With(
			zap.String("trace_id", span.SpanContext().TraceID().String()),
			zap.String("span_id", span.SpanContext().SpanID().String()),
		)
	}

	return logger
}

package telemetry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
)

func TestLoggerContext(t *testing.T) {
	// Test WithContext and FromContext
	logger, _ := NewLogger()
	ctx := context.Background()

	// Initial context should return default logger (not nil)
	l := FromContext(ctx)
	assert.NotNil(t, l)

	// Inject logger
	ctx = WithContext(ctx, logger)
	l2 := FromContext(ctx)
	assert.Equal(t, logger, l2)
}

func TestLoggerWithTrace(t *testing.T) {
	// Test trace correlation
	logger, _ := NewLogger()
	ctx := context.Background()
	ctx = WithContext(ctx, logger)

	// Mock a span
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{1},
		SpanID:     trace.SpanID{1},
		TraceFlags: trace.FlagsSampled,
	})
	ctx = trace.ContextWithSpanContext(ctx, sc)

	// Logger from context should have trace fields (we can't easily check internal zap fields without a hook,
	// but we can ensure it doesn't panic and returns a logger)
	l := FromContext(ctx)
	assert.NotNil(t, l)
	assert.NotEqual(t, logger, l) // Should be a child logger with fields
}

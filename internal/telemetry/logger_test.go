package telemetry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
)

func TestLoggerContext(t *testing.T) {
	logger, _ := NewLogger()
	ctx := context.Background()

	l := FromContext(ctx)
	assert.NotNil(t, l)

	ctx = WithContext(ctx, logger)
	l2 := FromContext(ctx)
	assert.Equal(t, logger, l2)
}

func TestLoggerWithTrace(t *testing.T) {
	logger, _ := NewLogger()
	ctx := context.Background()
	ctx = WithContext(ctx, logger)

	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{1},
		SpanID:     trace.SpanID{1},
		TraceFlags: trace.FlagsSampled,
	})
	ctx = trace.ContextWithSpanContext(ctx, sc)

	l := FromContext(ctx)
	assert.NotNil(t, l)
	assert.NotEqual(t, logger, l)
}

package telemetry

import (
	"fmt"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// EchoMiddleware creates an Echo middleware for OpenTelemetry tracing
func EchoMiddleware(tracer trace.Tracer) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			ctx := req.Context()

			// Create span name from route
			spanName := fmt.Sprintf("%s %s", req.Method, c.Path())
			if c.Path() == "" {
				spanName = fmt.Sprintf("%s %s", req.Method, req.URL.Path)
			}

			// Start span
			ctx, span := tracer.Start(ctx, spanName,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					semconv.HTTPRequestMethodKey.String(req.Method),
					semconv.HTTPRoute(c.Path()),
					semconv.URLPath(req.URL.Path),
					semconv.URLScheme(req.URL.Scheme),
					semconv.ServerAddressKey.String(req.Host),
					semconv.UserAgentOriginal(req.UserAgent()),
					semconv.ClientAddressKey.String(c.RealIP()),
				),
			)
			defer span.End()

			// Update request context
			c.SetRequest(req.WithContext(ctx))

			// Call next handler
			err := next(c)

			// Set response attributes
			status := c.Response().Status
			span.SetAttributes(
				semconv.HTTPResponseStatusCode(status),
			)

			// Set span status based on HTTP status
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
			} else if status >= 400 {
				span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", status))
			} else {
				span.SetStatus(codes.Ok, "")
			}

			return err
		}
	}
}

// FastHTTPSpanFromContext extracts or creates a span context for FastHTTP requests
// This is a helper for manual instrumentation since FastHTTP doesn't have middleware like Echo
type FastHTTPSpanCarrier struct {
	headers map[string]string
}

func (c *FastHTTPSpanCarrier) Get(key string) string {
	return c.headers[key]
}

func (c *FastHTTPSpanCarrier) Set(key, value string) {
	c.headers[key] = value
}

func (c *FastHTTPSpanCarrier) Keys() []string {
	keys := make([]string, 0, len(c.headers))
	for k := range c.headers {
		keys = append(keys, k)
	}
	return keys
}

// AddIngestionAttributes adds common ingestion attributes to a span
func AddIngestionAttributes(span trace.Span, orgID, projectID, visitorID, eventType string) {
	if span == nil {
		return
	}

	attrs := make([]attribute.KeyValue, 0, 4)
	if orgID != "" {
		attrs = append(attrs, AttrOrgID.String(orgID))
	}
	if projectID != "" {
		attrs = append(attrs, AttrProjectID.String(projectID))
	}
	if visitorID != "" {
		attrs = append(attrs, AttrVisitorID.String(visitorID))
	}
	if eventType != "" {
		attrs = append(attrs, AttrEventType.String(eventType))
	}

	span.SetAttributes(attrs...)
}

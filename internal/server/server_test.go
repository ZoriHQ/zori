package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"zori/internal/config"
	"zori/internal/telemetry"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestServerTelemetry(t *testing.T) {
	// Setup
	cfg := &config.Config{
		ServiceName: "test-service",
	}
	logger, _ := telemetry.NewLogger()

	srv := New(cfg, logger)

	// Create a request
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	// Serve
	srv.Echo.ServeHTTP(rec, req)

	// Assertions
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Zori - API Server", rec.Body.String())
}

func TestServerMiddlewareInjection(t *testing.T) {
	// Setup
	cfg := &config.Config{
		ServiceName: "test-service",
	}
	logger, _ := telemetry.NewLogger()

	srv := New(cfg, logger)

	// Add a test handler that checks for logger in context
	srv.Echo.GET("/test-context", func(c echo.Context) error {
		req := c.Request()
		l := telemetry.FromContext(req.Context())
		if l == nil {
			return c.String(http.StatusInternalServerError, "no logger")
		}
		return c.String(http.StatusOK, "ok")
	})

	// Create a request
	req := httptest.NewRequest(http.MethodGet, "/test-context", nil)
	rec := httptest.NewRecorder()

	// Serve
	srv.Echo.ServeHTTP(rec, req)

	// Assertions
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Body.String())
}

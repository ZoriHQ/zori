package ctx

import (
	"context"
	"zori/internal/logger"
	"zori/internal/storage/postgres/models"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/trace"
)

// Ctx is a wrapper around echo.Context that provides additional functionality
// for handling user authentication, organization context, and tracing
type Ctx struct {
	context.Context
	Echo   echo.Context
	User   *models.Account
	orgID  string
	span   trace.Span
	logger *logger.Logger
}

func NewCtx(c echo.Context) *Ctx {
	ctx := c.Request().Context()
	return &Ctx{
		Context: ctx,
		Echo:    c,
		span:    trace.SpanFromContext(ctx),
		logger:  logger.Default().WithContext(ctx),
	}
}

func (c *Ctx) SetUser(user *models.Account) {
	c.User = user
}

func (c *Ctx) SetOrgID(orgID string) {
	c.orgID = orgID
}

func (c *Ctx) IsAuthenticated() bool {
	return c.User != nil
}

func (c *Ctx) HasOrg() bool {
	return c.orgID != ""
}

func (c *Ctx) UserID() string {
	if c.User != nil {
		return c.User.ID
	}
	return ""
}

func (c *Ctx) OrgID() string {
	return c.orgID
}

// Span returns the current span
func (c *Ctx) Span() trace.Span {
	return c.span
}

// StartSpan starts a new child span
func (c *Ctx) StartSpan(tracer trace.Tracer, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	ctx, span := tracer.Start(c.Context, name, opts...)
	c.Context = ctx
	c.span = span
	c.logger = logger.Default().WithContext(ctx)
	return ctx, span
}

// Logger returns a context-aware logger with trace information
func (c *Ctx) Logger() *logger.Logger {
	if c.logger == nil {
		c.logger = logger.Default().WithContext(c.Context)
	}
	return c.logger
}

// UpdateContext updates the underlying context (useful after starting spans)
func (c *Ctx) UpdateContext(ctx context.Context) {
	c.Context = ctx
	c.span = trace.SpanFromContext(ctx)
	c.logger = logger.Default().WithContext(ctx)
}

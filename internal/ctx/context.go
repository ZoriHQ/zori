package ctx

import (
	"context"
	"zori/internal/storage/postgres/models"
	"zori/internal/telemetry"

	"github.com/labstack/echo/v4"
)

// Ctx is a wrapper around echo.Context that provides additional functionality
// for handling user authentication and organization context
type Ctx struct {
	context.Context
	Echo  echo.Context
	User  *models.Account
	orgID string
}

func NewCtx(c echo.Context) *Ctx {
	return &Ctx{
		Context: c.Request().Context(),
		Echo:    c,
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

func (c *Ctx) Logger() *telemetry.Logger {
	return telemetry.FromContext(c.Context)
}

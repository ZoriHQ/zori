package server

import (
	"marker/services/auth/models"

	"github.com/labstack/echo/v4"
)

// Ctx is a wrapper around echo.Context that provides additional functionality
// for handling user authentication and organization context
type Ctx struct {
	Echo echo.Context
	User *models.Account
	Org  *models.Organization
}

// NewCtx creates a new app context from echo context
func NewCtx(c echo.Context) *Ctx {
	return &Ctx{
		Echo: c,
	}
}

// SetUser sets the authenticated user in the context
func (c *Ctx) SetUser(user *models.Account) {
	c.User = user
}

// SetOrg sets the organization in the context
func (c *Ctx) SetOrg(org *models.Organization) {
	c.Org = org
}

// IsAuthenticated returns true if user is set in context
func (c *Ctx) IsAuthenticated() bool {
	return c.User != nil
}

// HasOrg returns true if organization is set in context
func (c *Ctx) HasOrg() bool {
	return c.Org != nil
}

// UserID returns the user ID if authenticated, empty string otherwise
func (c *Ctx) UserID() string {
	if c.User != nil {
		return c.User.ID
	}
	return ""
}

// OrgID returns the organization ID if set, empty string otherwise
func (c *Ctx) OrgID() string {
	if c.Org != nil {
		return c.Org.ID
	}
	return ""
}

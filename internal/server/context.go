package server

import (
	"marker/internal/storage/postgres/models"

	"github.com/labstack/echo/v4"
)

// Ctx is a wrapper around echo.Context that provides additional functionality
// for handling user authentication and organization context
type Ctx struct {
	Echo echo.Context
	User *models.Account
	Org  *models.Organization
}

func NewCtx(c echo.Context) *Ctx {
	return &Ctx{
		Echo: c,
	}
}

func (c *Ctx) SetUser(user *models.Account) {
	c.User = user
}

func (c *Ctx) SetOrg(org *models.Organization) {
	c.Org = org
}

func (c *Ctx) IsAuthenticated() bool {
	return c.User != nil
}

func (c *Ctx) HasOrg() bool {
	return c.Org != nil
}

func (c *Ctx) UserID() string {
	if c.User != nil {
		return c.User.ID
	}
	return ""
}

func (c *Ctx) OrgID() string {
	if c.Org != nil {
		return c.Org.ID
	}
	return ""
}

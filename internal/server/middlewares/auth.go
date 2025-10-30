package middlewares

import "github.com/labstack/echo/v4"

// AuthMiddleware is an interface for authentication middleware
type AuthMiddleware interface {
	Middleware() echo.MiddlewareFunc
}

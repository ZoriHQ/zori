package middlewares

import "github.com/labstack/echo/v4"

type AuthMiddleware interface {
	Middleware() echo.MiddlewareFunc
}

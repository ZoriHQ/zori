package server

import (
	"fmt"
	"net/http"
	"zori/internal/ctx"
	"zori/internal/server/validators"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type HandlerFunc[T any] func(*ctx.Ctx) (T, error)
type HandlerFuncWithFilter[T any, F any] func(*ctx.Ctx, *F) (*T, error)

type Server struct {
	Echo *echo.Echo
}

func New() *Server {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
	e.Validator = validators.NewFiltersValidator()
	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "Zori - API Server")
	})

	return &Server{
		Echo: e,
	}
}

func GET[T any](s *Server, path string, handler HandlerFunc[T]) {
	s.Echo.GET(path, wrapHandler(s, handler))
}

func POST[T any](s *Server, path string, handler HandlerFunc[T]) {
	s.Echo.POST(path, wrapHandler(s, handler))
}

func PUT[T any](s *Server, path string, handler HandlerFunc[T]) {
	s.Echo.PUT(path, wrapHandler(s, handler))
}

func DELETE[T any](s *Server, path string, handler HandlerFunc[T]) {
	s.Echo.DELETE(path, wrapHandler(s, handler))
}

func PATCH[T any](s *Server, path string, handler HandlerFunc[T]) {
	s.Echo.PATCH(path, wrapHandler(s, handler))
}

func (s *Server) Group(prefix string) *Group {
	return &Group{
		echo:   s.Echo.Group(prefix),
		server: s,
	}
}

func wrapHandlerWithFilter[T any, F any](s *Server, handler HandlerFuncWithFilter[T, F]) echo.HandlerFunc {
	return func(c echo.Context) error {
		appctx, ok := c.Get("ctx").(*ctx.Ctx)
		if !ok {
			appctx = ctx.NewCtx(c)
			c.Set("ctx", appctx)
		}

		filter := new(F)
		if err := c.Bind(filter); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		if err := c.Validate(filter); err != nil {
			return err
		}

		result, err := handler(appctx, filter)

		if err != nil {
			return s.handleError(c, err)
		}

		statusCode := c.Response().Status
		if statusCode == 0 {
			statusCode = http.StatusOK
		}

		return c.JSON(statusCode, result)
	}
}

func wrapHandler[T any](s *Server, handler HandlerFunc[T]) echo.HandlerFunc {
	return func(c echo.Context) error {
		appctx, ok := c.Get("ctx").(*ctx.Ctx)
		if !ok {
			appctx = ctx.NewCtx(c)
			c.Set("ctx", appctx)
		}

		result, err := handler(appctx)

		if err != nil {
			return s.handleError(c, err)
		}

		statusCode := c.Response().Status
		if statusCode == 0 {
			statusCode = http.StatusOK
		}

		return c.JSON(statusCode, result)
	}
}

func (s *Server) handleError(c echo.Context, err error) error {
	if he, ok := err.(*echo.HTTPError); ok {
		return c.JSON(he.Code, map[string]string{
			"error": fmt.Sprintf("%v", he.Message),
		})
	}

	return c.JSON(http.StatusInternalServerError, map[string]string{
		"error": err.Error(),
	})
}

type Group struct {
	echo   *echo.Group
	server *Server
}

func GroupGET[T any](g *Group, path string, handler HandlerFunc[T], middleware ...echo.MiddlewareFunc) {
	wrappedHandler := wrapHandler(g.server, handler)
	if len(middleware) > 0 {
		for i := len(middleware) - 1; i >= 0; i-- {
			wrappedHandler = middleware[i](wrappedHandler)
		}
	}
	g.echo.GET(path, wrappedHandler)
}

func GroupGetWithFilter[T any, F any](g *Group, path string, handler HandlerFuncWithFilter[T, F], middleware ...echo.MiddlewareFunc) {
	wrappedHandler := wrapHandlerWithFilter(g.server, handler)
	if len(middleware) > 0 {
		for i := len(middleware) - 1; i >= 0; i-- {
			wrappedHandler = middleware[i](wrappedHandler)
		}
	}
	g.echo.GET(path, wrappedHandler)
}

func GroupPOST[T any](g *Group, path string, handler HandlerFunc[T], middleware ...echo.MiddlewareFunc) {
	wrappedHandler := wrapHandler(g.server, handler)
	if len(middleware) > 0 {
		for i := len(middleware) - 1; i >= 0; i-- {
			wrappedHandler = middleware[i](wrappedHandler)
		}
	}
	g.echo.POST(path, wrappedHandler)
}

func GroupPUT[T any](g *Group, path string, handler HandlerFunc[T], middleware ...echo.MiddlewareFunc) {
	wrappedHandler := wrapHandler(g.server, handler)
	if len(middleware) > 0 {
		for i := len(middleware) - 1; i >= 0; i-- {
			wrappedHandler = middleware[i](wrappedHandler)
		}
	}
	g.echo.PUT(path, wrappedHandler)
}

func GroupDELETE[T any](g *Group, path string, handler HandlerFunc[T], middleware ...echo.MiddlewareFunc) {
	wrappedHandler := wrapHandler(g.server, handler)
	if len(middleware) > 0 {
		for i := len(middleware) - 1; i >= 0; i-- {
			wrappedHandler = middleware[i](wrappedHandler)
		}
	}
	g.echo.DELETE(path, wrappedHandler)
}

func GroupPATCH[T any](g *Group, path string, handler HandlerFunc[T], middleware ...echo.MiddlewareFunc) {
	wrappedHandler := wrapHandler(g.server, handler)
	if len(middleware) > 0 {
		for i := len(middleware) - 1; i >= 0; i-- {
			wrappedHandler = middleware[i](wrappedHandler)
		}
	}
	g.echo.PATCH(path, wrappedHandler)
}

func (g *Group) Use(middleware ...echo.MiddlewareFunc) {
	g.echo.Use(middleware...)
}

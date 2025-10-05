package server

import (
	"marker/internal/ctx"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// HandlerFunc is a generic handler function that returns a specific type T
type HandlerFunc[T any] func(*ctx.Ctx) (T, error)

type Server struct {
	Echo *echo.Echo
}

func New() *Server {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	return &Server{
		Echo: e,
	}
}

// GET registers a GET route with a generic handler
func GET[T any](s *Server, path string, handler HandlerFunc[T]) {
	s.Echo.GET(path, wrapHandler(s, handler))
}

// POST registers a POST route with a generic handler
func POST[T any](s *Server, path string, handler HandlerFunc[T]) {
	s.Echo.POST(path, wrapHandler(s, handler))
}

// PUT registers a PUT route with a generic handler
func PUT[T any](s *Server, path string, handler HandlerFunc[T]) {
	s.Echo.PUT(path, wrapHandler(s, handler))
}

// DELETE registers a DELETE route with a generic handler
func DELETE[T any](s *Server, path string, handler HandlerFunc[T]) {
	s.Echo.DELETE(path, wrapHandler(s, handler))
}

// PATCH registers a PATCH route with a generic handler
func PATCH[T any](s *Server, path string, handler HandlerFunc[T]) {
	s.Echo.PATCH(path, wrapHandler(s, handler))
}

// Group creates a new route group with the given prefix
func (s *Server) Group(prefix string) *Group {
	return &Group{
		echo:   s.Echo.Group(prefix),
		server: s,
	}
}

// wrapHandler wraps a generic handler into an echo.HandlerFunc
func wrapHandler[T any](s *Server, handler HandlerFunc[T]) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := ctx.NewCtx(c)

		result, err := handler(ctx)

		if err != nil {
			return s.handleError(c, err)
		}

		return c.JSON(http.StatusOK, result)
	}
}

// handleError handles errors and returns appropriate HTTP responses
func (s *Server) handleError(c echo.Context, err error) error {
	return c.JSON(http.StatusInternalServerError, map[string]string{
		"error": err.Error(),
	})
}

// Group represents a group of routes with a common prefix
type Group struct {
	echo   *echo.Group
	server *Server
}

// GET registers a GET route within the group with a generic handler
func GroupGET[T any](g *Group, path string, handler HandlerFunc[T]) {
	g.echo.GET(path, wrapHandler(g.server, handler))
}

// POST registers a POST route within the group with a generic handler
func GroupPOST[T any](g *Group, path string, handler HandlerFunc[T]) {
	g.echo.POST(path, wrapHandler(g.server, handler))
}

// PUT registers a PUT route within the group with a generic handler
func GroupPUT[T any](g *Group, path string, handler HandlerFunc[T]) {
	g.echo.PUT(path, wrapHandler(g.server, handler))
}

// DELETE registers a DELETE route within the group with a generic handler
func GroupDELETE[T any](g *Group, path string, handler HandlerFunc[T]) {
	g.echo.DELETE(path, wrapHandler(g.server, handler))
}

// PATCH registers a PATCH route within the group with a generic handler
func GroupPATCH[T any](g *Group, path string, handler HandlerFunc[T]) {
	g.echo.PATCH(path, wrapHandler(g.server, handler))
}

// Use applies middleware to the group
func (g *Group) Use(middleware ...echo.MiddlewareFunc) {
	g.echo.Use(middleware...)
}

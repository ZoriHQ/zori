package server

import (
	"marker/internal/ctx"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type HandlerFunc func(*ctx.Ctx) (any, error)

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

func (s *Server) GET(path string, handler HandlerFunc) {
	s.Echo.GET(path, s.wrapHandler(handler))
}

func (s *Server) POST(path string, handler HandlerFunc) {
	s.Echo.POST(path, s.wrapHandler(handler))
}

func (s *Server) PUT(path string, handler HandlerFunc) {
	s.Echo.PUT(path, s.wrapHandler(handler))
}

func (s *Server) DELETE(path string, handler HandlerFunc) {
	s.Echo.DELETE(path, s.wrapHandler(handler))
}

func (s *Server) PATCH(path string, handler HandlerFunc) {
	s.Echo.PATCH(path, s.wrapHandler(handler))
}

func (s *Server) Group(prefix string) *Group {
	return &Group{
		echo:   s.Echo.Group(prefix),
		server: s,
	}
}

func (s *Server) wrapHandler(handler HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := ctx.NewCtx(c)

		result, err := handler(ctx)

		if err != nil {
			return s.handleError(c, err)
		}

		return c.JSON(http.StatusOK, result)
	}
}

func (s *Server) handleError(c echo.Context, err error) error {
	return c.JSON(http.StatusInternalServerError, map[string]string{
		"error": err.Error(),
	})
}

type Group struct {
	echo   *echo.Group
	server *Server
}

func (g *Group) GET(path string, handler HandlerFunc) {
	g.echo.GET(path, g.server.wrapHandler(handler))
}

func (g *Group) POST(path string, handler HandlerFunc) {
	g.echo.POST(path, g.server.wrapHandler(handler))
}

func (g *Group) PUT(path string, handler HandlerFunc) {
	g.echo.PUT(path, g.server.wrapHandler(handler))
}

func (g *Group) DELETE(path string, handler HandlerFunc) {
	g.echo.DELETE(path, g.server.wrapHandler(handler))
}

func (g *Group) PATCH(path string, handler HandlerFunc) {
	g.echo.PATCH(path, g.server.wrapHandler(handler))
}

func (g *Group) Use(middleware ...echo.MiddlewareFunc) {
	g.echo.Use(middleware...)
}

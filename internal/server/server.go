package server

import (
	"net/http"
	"zori/internal/ctx"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

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

func (s *Server) handleError(c echo.Context, err error) error {
	return c.JSON(http.StatusInternalServerError, map[string]string{
		"error": err.Error(),
	})
}

type Group struct {
	echo   *echo.Group
	server *Server
}

func GroupGET[T any](g *Group, path string, handler HandlerFunc[T]) {
	g.echo.GET(path, wrapHandler(g.server, handler))
}

func GroupPOST[T any](g *Group, path string, handler HandlerFunc[T]) {
	g.echo.POST(path, wrapHandler(g.server, handler))
}

func GroupPUT[T any](g *Group, path string, handler HandlerFunc[T]) {
	g.echo.PUT(path, wrapHandler(g.server, handler))
}

func GroupDELETE[T any](g *Group, path string, handler HandlerFunc[T]) {
	g.echo.DELETE(path, wrapHandler(g.server, handler))
}

func GroupPATCH[T any](g *Group, path string, handler HandlerFunc[T]) {
	g.echo.PATCH(path, wrapHandler(g.server, handler))
}

func (g *Group) Use(middleware ...echo.MiddlewareFunc) {
	g.echo.Use(middleware...)
}

package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// HandlerFunc defines the signature for our custom handlers
type HandlerFunc func(*Ctx) (any, error)

// Server wraps echo.Echo with custom functionality
type Server struct {
	Echo *echo.Echo
}

// New creates a new server wrapper
func New() *Server {
	return &Server{
		Echo: echo.New(),
	}
}

// GET registers a GET route with custom handler
func (s *Server) GET(path string, handler HandlerFunc) {
	s.Echo.GET(path, s.wrapHandler(handler))
}

// POST registers a POST route with custom handler
func (s *Server) POST(path string, handler HandlerFunc) {
	s.Echo.POST(path, s.wrapHandler(handler))
}

// PUT registers a PUT route with custom handler
func (s *Server) PUT(path string, handler HandlerFunc) {
	s.Echo.PUT(path, s.wrapHandler(handler))
}

// DELETE registers a DELETE route with custom handler
func (s *Server) DELETE(path string, handler HandlerFunc) {
	s.Echo.DELETE(path, s.wrapHandler(handler))
}

// PATCH registers a PATCH route with custom handler
func (s *Server) PATCH(path string, handler HandlerFunc) {
	s.Echo.PATCH(path, s.wrapHandler(handler))
}

// Group creates a new route group
func (s *Server) Group(prefix string) *Group {
	return &Group{
		echo:   s.Echo.Group(prefix),
		server: s,
	}
}

// wrapHandler converts our custom HandlerFunc to echo.HandlerFunc
func (s *Server) wrapHandler(handler HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Create our custom context
		ctx := NewCtx(c)

		// Call the handler
		result, err := handler(ctx)

		// Handle error
		if err != nil {
			return s.handleError(c, err)
		}

		// Handle success response
		return c.JSON(http.StatusOK, result)
	}
}

// handleError handles errors returned by handlers
func (s *Server) handleError(c echo.Context, err error) error {
	// TODO: Add more sophisticated error handling based on error types
	return c.JSON(http.StatusInternalServerError, map[string]string{
		"error": err.Error(),
	})
}

// Group wraps echo.Group with custom functionality
type Group struct {
	echo   *echo.Group
	server *Server
}

// GET registers a GET route in the group
func (g *Group) GET(path string, handler HandlerFunc) {
	g.echo.GET(path, g.server.wrapHandler(handler))
}

// POST registers a POST route in the group
func (g *Group) POST(path string, handler HandlerFunc) {
	g.echo.POST(path, g.server.wrapHandler(handler))
}

// PUT registers a PUT route in the group
func (g *Group) PUT(path string, handler HandlerFunc) {
	g.echo.PUT(path, g.server.wrapHandler(handler))
}

// DELETE registers a DELETE route in the group
func (g *Group) DELETE(path string, handler HandlerFunc) {
	g.echo.DELETE(path, g.server.wrapHandler(handler))
}

// PATCH registers a PATCH route in the group
func (g *Group) PATCH(path string, handler HandlerFunc) {
	g.echo.PATCH(path, g.server.wrapHandler(handler))
}

// Use adds middleware to the group
func (g *Group) Use(middleware ...echo.MiddlewareFunc) {
	g.echo.Use(middleware...)
}

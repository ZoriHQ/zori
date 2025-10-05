package server

import (
	_ "marker/docs" // Import generated docs
	"net/http"

	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"
)

// RegisterSwaggerRoutes registers Swagger UI routes
func RegisterSwaggerRoutes(s *Server) {
	// Swagger UI endpoint
	s.Echo.GET("/swagger/*", echoSwagger.WrapHandler)

	// Redirect /swagger to /swagger/index.html
	s.Echo.GET("/swagger", func(c echo.Context) error {
		return c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
	})

	// API documentation JSON endpoint
	s.Echo.GET("/api/docs", func(c echo.Context) error {
		return c.File("docs/swagger.json")
	})

	// API documentation YAML endpoint
	s.Echo.GET("/api/docs.yaml", func(c echo.Context) error {
		return c.File("docs/swagger.yaml")
	})
}

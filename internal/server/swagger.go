package server

import (
	"net/http"
	_ "zori/docs" // Import generated docs

	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"
)

func RegisterSwaggerRoutes(s *Server) {
	s.Echo.GET("/swagger/*", echoSwagger.WrapHandler)

	s.Echo.GET("/swagger", func(c echo.Context) error {
		return c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
	})

	s.Echo.GET("/api/docs", func(c echo.Context) error {
		return c.File("docs/swagger.json")
	})

	s.Echo.GET("/api/docs.yaml", func(c echo.Context) error {
		return c.File("docs/swagger.yaml")
	})
}

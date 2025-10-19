package web

import (
	"zori/internal/server"
	"zori/internal/server/middlewares"
	"zori/services/analytics/services"
)

func RegisterRoutes(s *server.Server, analyticsService *services.AnalyticsService, jwtMiddleware *middlewares.JwtMiddleware) {
	analyticsRouteGroup := s.Group("/api/v1/analytics")
	analyticsRouteGroup.Use(jwtMiddleware.Middleware())

	// Visitors endpoints
	server.GroupGET(analyticsRouteGroup, "/visitors/device", analyticsService.GetVisitorsByDevice)
	server.GroupGET(analyticsRouteGroup, "/visitors/origin", analyticsService.GetUniqueVisitorsByOrigin)
	server.GroupGET(analyticsRouteGroup, "/visitors/country", analyticsService.GetUniqueVisitorsByCountry)

	// Events endpoints
	server.GroupGET(analyticsRouteGroup, "/events/recent", analyticsService.GetRecentEvents)
}

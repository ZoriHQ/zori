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
	server.GroupGET(analyticsRouteGroup, "/visitors/top", analyticsService.GetTopVisitors)
	server.GroupGET(analyticsRouteGroup, "/visitors/profile", analyticsService.GetVisitorProfile)
	server.GroupGET(analyticsRouteGroup, "/visitors/timeline", analyticsService.GetUniqueVisitorsTimeline)

	// Events endpoints
	server.GroupGET(analyticsRouteGroup, "/events/recent", analyticsService.GetRecentEvents)

	// Session endpoints
	server.GroupGET(analyticsRouteGroup, "/sessions/metrics", analyticsService.GetSessionMetrics)
	server.GroupGET(analyticsRouteGroup, "/sessions/bounce-rate", analyticsService.GetBounceRate)

	// User activity endpoints
	server.GroupGET(analyticsRouteGroup, "/users/active", analyticsService.GetActiveUsers)

	// Retention endpoints
	server.GroupGET(analyticsRouteGroup, "/retention/return-rate", analyticsService.GetReturnRate)
	server.GroupGET(analyticsRouteGroup, "/retention/churn-rate", analyticsService.GetChurnRate)
	server.GroupGET(analyticsRouteGroup, "/retention/cohorts", analyticsService.GetCohortAnalysis)

	// Dashboard endpoint
	server.GroupGET(analyticsRouteGroup, "/dashboard", analyticsService.GetDashboardMetrics)
}

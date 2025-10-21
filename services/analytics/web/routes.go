package web

import (
	"time"
	"zori/internal/cache"
	"zori/internal/server"
	"zori/internal/server/middlewares"
	"zori/services/analytics/services"
)

func RegisterRoutes(
	s *server.Server,
	analyticsService *services.AnalyticsService,
	jwtMiddleware *middlewares.JwtMiddleware,
	cacheMiddleware *middlewares.CacheMiddleware,
) {
	analyticsRouteGroup := s.Group("/api/v1/analytics")
	analyticsRouteGroup.Use(jwtMiddleware.Middleware())

	highFrequencyTTL := 1 * time.Minute
	mediumFrequencyTTL := 2 * time.Minute
	lowFrequencyTTL := 5 * time.Minute

	cachePrefix := string(cache.AnalyticsCacheKey)

	server.GroupGET(analyticsRouteGroup, "/visitors/device",
		analyticsService.GetVisitorsByDevice,
		cacheMiddleware.Middleware(middlewares.CacheConfig{
			TTL:       mediumFrequencyTTL,
			KeyPrefix: cachePrefix,
		}))

	server.GroupGET(analyticsRouteGroup, "/visitors/origin",
		analyticsService.GetUniqueVisitorsByOrigin,
		cacheMiddleware.Middleware(middlewares.CacheConfig{
			TTL:       lowFrequencyTTL,
			KeyPrefix: cachePrefix,
		}))

	server.GroupGET(analyticsRouteGroup, "/visitors/country",
		analyticsService.GetUniqueVisitorsByCountry,
		cacheMiddleware.Middleware(middlewares.CacheConfig{
			TTL:       lowFrequencyTTL,
			KeyPrefix: cachePrefix,
		}))

	server.GroupGET(analyticsRouteGroup, "/visitors/top",
		analyticsService.GetTopVisitors,
		cacheMiddleware.Middleware(middlewares.CacheConfig{
			TTL:       mediumFrequencyTTL,
			KeyPrefix: cachePrefix,
		}))

	server.GroupGET(analyticsRouteGroup, "/visitors/profile", analyticsService.GetVisitorProfile)

	// POST endpoint for manual visitor identification (no cache)
	server.GroupPOST(analyticsRouteGroup, "/visitors/identify", analyticsService.IdentifyVisitor)

	server.GroupGET(analyticsRouteGroup, "/visitors/timeline",
		analyticsService.GetUniqueVisitorsTimeline,
		cacheMiddleware.Middleware(middlewares.CacheConfig{
			TTL:       highFrequencyTTL,
			KeyPrefix: cachePrefix,
		}))

	server.GroupGET(analyticsRouteGroup, "/events/recent", analyticsService.GetRecentEvents)

	server.GroupGET(analyticsRouteGroup, "/sessions/metrics",
		analyticsService.GetSessionMetrics,
		cacheMiddleware.Middleware(middlewares.CacheConfig{
			TTL:       mediumFrequencyTTL,
			KeyPrefix: cachePrefix,
		}))

	server.GroupGET(analyticsRouteGroup, "/sessions/bounce-rate",
		analyticsService.GetBounceRate,
		cacheMiddleware.Middleware(middlewares.CacheConfig{
			TTL:       mediumFrequencyTTL,
			KeyPrefix: cachePrefix,
		}))

	server.GroupGET(analyticsRouteGroup, "/users/active",
		analyticsService.GetActiveUsers,
		cacheMiddleware.Middleware(middlewares.CacheConfig{
			TTL:       mediumFrequencyTTL,
			KeyPrefix: cachePrefix,
		}))

	server.GroupGET(analyticsRouteGroup, "/retention/return-rate",
		analyticsService.GetReturnRate,
		cacheMiddleware.Middleware(middlewares.CacheConfig{
			TTL:       lowFrequencyTTL,
			KeyPrefix: cachePrefix,
		}))

	server.GroupGET(analyticsRouteGroup, "/retention/churn-rate",
		analyticsService.GetChurnRate,
		cacheMiddleware.Middleware(middlewares.CacheConfig{
			TTL:       lowFrequencyTTL,
			KeyPrefix: cachePrefix,
		}))

	server.GroupGET(analyticsRouteGroup, "/retention/cohorts",
		analyticsService.GetCohortAnalysis,
		cacheMiddleware.Middleware(middlewares.CacheConfig{
			TTL:       lowFrequencyTTL,
			KeyPrefix: cachePrefix,
		}))

	server.GroupGET(analyticsRouteGroup, "/dashboard",
		analyticsService.GetDashboardMetrics,
		cacheMiddleware.Middleware(middlewares.CacheConfig{
			TTL:       highFrequencyTTL,
			KeyPrefix: cachePrefix,
		}))
}

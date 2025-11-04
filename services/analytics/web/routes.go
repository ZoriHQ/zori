package web

import (
	"time"
	"zori/internal/cache"
	"zori/internal/server"
	"zori/internal/server/middlewares"
	"zori/services/analytics/services"
)

const (
	highFrequencyTTL   = 1 * time.Minute
	mediumFrequencyTTL = 2 * time.Minute
	lowFrequencyTTL    = 5 * time.Minute

	cachePrefix = string(cache.AnalyticsCacheKey)
)

var (
	lowFreqTTLCache = middlewares.CacheConfig{
		TTL:       lowFrequencyTTL,
		KeyPrefix: cachePrefix,
	}

	mediumFreqTTLCache = middlewares.CacheConfig{
		TTL:       mediumFrequencyTTL,
		KeyPrefix: cachePrefix,
	}

	highFreqTTLCache = middlewares.CacheConfig{
		TTL:       highFrequencyTTL,
		KeyPrefix: cachePrefix,
	}
)

func RegisterRoutes(
	s *server.Server,
	analyticsService *services.AnalyticsService,
	authMiddleware middlewares.AuthMiddleware,
	cacheMiddleware *middlewares.CacheMiddleware,
) {
	analyticsRouteGroup := s.Group("/api/v1/analytics")
	analyticsRouteGroup.Use(authMiddleware.Middleware())

	mfMiddleware := cacheMiddleware.Middleware(mediumFreqTTLCache)
	hfMiddleware := cacheMiddleware.Middleware(highFreqTTLCache)
	lfMiddleware := cacheMiddleware.Middleware(lowFreqTTLCache)

	server.GroupGetWithFilter(analyticsRouteGroup, "/visitors/device", analyticsService.GetVisitorsByDevice, mfMiddleware)
	server.GroupGetWithFilter(analyticsRouteGroup, "/visitors/origin", analyticsService.GetUniqueVisitorsByOrigin, lfMiddleware)
	server.GroupGetWithFilter(analyticsRouteGroup, "/visitors/country", analyticsService.GetUniqueVisitorsByCountry, lfMiddleware)
	server.GroupGetWithFilter(analyticsRouteGroup, "/visitors/top", analyticsService.GetTopVisitors, mfMiddleware)

	server.GroupGetWithFilter(analyticsRouteGroup, "/visitors/profile", analyticsService.GetVisitorProfile)
	server.GroupPOST(analyticsRouteGroup, "/visitors/identify", analyticsService.IdentifyVisitor)

	server.GroupGetWithFilter(analyticsRouteGroup, "/visitors/timeline", analyticsService.GetUniqueVisitorsTimeline, hfMiddleware)

	// Events page endpoints
	server.GroupGetWithFilter(analyticsRouteGroup, "/events/recent", analyticsService.GetRecentEvents)
	server.GroupGetWithFilter(analyticsRouteGroup, "/events/filter-options", analyticsService.GetEventFilterOptions, mfMiddleware)

	server.GroupGetWithFilter(analyticsRouteGroup, "/dashboard", analyticsService.GetDashboardMetrics, hfMiddleware)
}

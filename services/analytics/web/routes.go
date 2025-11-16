package web

import (
	"time"
	"zori/internal/cache"
	"zori/internal/server"
	"zori/internal/server/middlewares"
	"zori/services/analytics/services"
)

const cachePrefix = string(cache.AnalyticsCacheKey)

var (
	lowFreqTTLCache = middlewares.CacheConfig{
		TTL:       5 * time.Minute,
		KeyPrefix: cachePrefix,
	}

	mediumFreqTTLCache = middlewares.CacheConfig{
		TTL:       2 * time.Minute,
		KeyPrefix: cachePrefix,
	}

	highFreqTTLCache = middlewares.CacheConfig{
		TTL:       1 * time.Minute,
		KeyPrefix: cachePrefix,
	}
)

func RegisterRoutes(
	s *server.Server,
	analyticsService *services.AnalyticsService,
	tilesService *services.TilesService,
	visitorsService *services.VisitorsService,
	authMiddleware middlewares.AuthMiddleware,
	cacheMiddleware *middlewares.CacheMiddleware,
) {
	analyticsRouteGroup := s.Group("/api/v1/analytics")
	analyticsRouteGroup.Use(authMiddleware.Middleware())

	mfMiddleware := cacheMiddleware.Middleware(mediumFreqTTLCache)
	hfMiddleware := cacheMiddleware.Middleware(highFreqTTLCache)
	lfMiddleware := cacheMiddleware.Middleware(lowFreqTTLCache)

	// TODO:: move to tiles service
	server.GroupGetWithFilter(analyticsRouteGroup, "/visitors/device", analyticsService.GetVisitorsByDevice, mfMiddleware)

	// Moved to tiles service
	server.GroupGetWithFilter(analyticsRouteGroup, "/visitors/origin", tilesService.GetTrafficSourceRefererTile, lfMiddleware)
	server.GroupGetWithFilter(analyticsRouteGroup, "/visitors/country", tilesService.GetTrafficSourceCountriesTile, lfMiddleware)

	// TODO:: potential refactor
	server.GroupGetWithFilter(analyticsRouteGroup, "/visitors/top", analyticsService.GetTopVisitors, mfMiddleware)

	server.GroupGetWithFilter(analyticsRouteGroup, "/visitors/profile", visitorsService.GetVisitorProfile)
	server.GroupPOST(analyticsRouteGroup, "/visitors/identify", visitorsService.IdentifyVisitor)

	// Tiles endpoints
	server.GroupGetWithFilter(analyticsRouteGroup, "/timeline", tilesService.GetTimelineTile, hfMiddleware)

	// Events page endpoints
	server.GroupGetWithFilter(analyticsRouteGroup, "/events/recent", analyticsService.GetRecentEvents)
	server.GroupGetWithFilter(analyticsRouteGroup, "/events/filter-options", analyticsService.GetEventFilterOptions, mfMiddleware)

	server.GroupGetWithFilter(analyticsRouteGroup, "/dashboard", analyticsService.GetDashboardMetrics, hfMiddleware)
}

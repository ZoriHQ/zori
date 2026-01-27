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

	// TODO:: potential refactor
	server.GroupGetWithFilter(analyticsRouteGroup, "/visitors/top", analyticsService.GetTopVisitors, mfMiddleware)

	server.GroupGetWithFilter(analyticsRouteGroup, "/visitors/profile", visitorsService.GetVisitorProfile)

	// Tiles endpoints
	server.GroupGetWithFilter(analyticsRouteGroup, "/timeline", tilesService.GetTimelineTile, hfMiddleware)
	server.GroupGetWithFilter(analyticsRouteGroup, "/tiles/unique-visitors", tilesService.GetUniqueVisitorsTile, hfMiddleware)
	server.GroupGetWithFilter(analyticsRouteGroup, "/tiles/unique-sessions", tilesService.GetUniqueSessionsTile, hfMiddleware)
	server.GroupGetWithFilter(analyticsRouteGroup, "/tiles/bounce-rate", tilesService.GetBounceRateTile, hfMiddleware)
	server.GroupGetWithFilter(analyticsRouteGroup, "/tiles/session-duration", tilesService.GetSessionDurationTile, hfMiddleware)
	server.GroupGetWithFilter(analyticsRouteGroup, "/tiles/pages-per-session", tilesService.GetPagesPerSessionTile, hfMiddleware)
	server.GroupGetWithFilter(analyticsRouteGroup, "/tiles/dau", tilesService.GetDAUTile, hfMiddleware)
	server.GroupGetWithFilter(analyticsRouteGroup, "/tiles/wau", tilesService.GetWAUTile, hfMiddleware)
	server.GroupGetWithFilter(analyticsRouteGroup, "/tiles/mau", tilesService.GetMAUTile, hfMiddleware)
	server.GroupGetWithFilter(analyticsRouteGroup, "/tiles/return-rate", tilesService.GetReturnRateTile, hfMiddleware)
	server.GroupGetWithFilter(analyticsRouteGroup, "/tiles/time-between-visits", tilesService.GetTimeBetweenVisitsTile, hfMiddleware)
	server.GroupGetWithFilter(analyticsRouteGroup, "/tiles/traffic-by-country", tilesService.GetTrafficSourceCountriesTile, lfMiddleware)
	server.GroupGetWithFilter(analyticsRouteGroup, "/tiles/traffic-by-referer", tilesService.GetTrafficSourceRefererTile, lfMiddleware)
	server.GroupGetWithFilter(analyticsRouteGroup, "/tiles/traffic-by-utm", tilesService.GetTrafficSourceUTMTile, lfMiddleware)
	server.GroupGetWithFilter(analyticsRouteGroup, "/tiles/visitors-by-browser", tilesService.GetVisitorsByBrowserTile, lfMiddleware)
	server.GroupGetWithFilter(analyticsRouteGroup, "/tiles/visitors-by-os", tilesService.GetVisitorsByOSTile, lfMiddleware)
	server.GroupGetWithFilter(analyticsRouteGroup, "/tiles/entry-pages", tilesService.GetEntryPagesTile, mfMiddleware)
	server.GroupGetWithFilter(analyticsRouteGroup, "/tiles/exit-pages", tilesService.GetExitPagesTile, mfMiddleware)

	// Events page endpoints
	server.GroupGetWithFilter(analyticsRouteGroup, "/events/recent", analyticsService.GetRecentEvents)
	server.GroupGetWithFilter(analyticsRouteGroup, "/events/filter-options", analyticsService.GetEventFilterOptions, mfMiddleware)
}

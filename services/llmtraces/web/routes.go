package web

import (
	"time"
	"zori/internal/cache"
	"zori/internal/server"
	"zori/internal/server/middlewares"
	"zori/services/llmtraces/services"
)

const cachePrefix = "llmtraces"

var (
	highFreqTTLCache = middlewares.CacheConfig{
		TTL:       1 * time.Minute,
		KeyPrefix: cachePrefix,
	}
)

func RegisterRoutes(
	s *server.Server,
	tilesService *services.LLMTracesTilesService,
	authMiddleware middlewares.AuthMiddleware,
	cacheMiddleware *middlewares.CacheMiddleware,
) {
	llmTracesRouteGroup := s.Group("/api/v1/llm-traces")
	llmTracesRouteGroup.Use(authMiddleware.Middleware())

	hfMiddleware := cacheMiddleware.Middleware(highFreqTTLCache)
	_ = cache.LLMPriceCacheKey

	server.GroupGetWithFilter(llmTracesRouteGroup, "/tiles/avg-price-per-customer", tilesService.GetAvgPricePerCustomerTile, hfMiddleware)
	server.GroupGetWithFilter(llmTracesRouteGroup, "/tiles/daily-price", tilesService.GetDailyPriceTile, hfMiddleware)
	server.GroupGetWithFilter(llmTracesRouteGroup, "/tiles/timeline", tilesService.GetDailyPriceTimelineTile, hfMiddleware)
}

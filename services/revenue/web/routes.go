package web

import (
	"time"
	"zori/internal/cache"
	"zori/internal/server"
	"zori/internal/server/middlewares"
	"zori/services/revenue/services"
)

func RegisterRoutes(
	s *server.Server,
	revenueService *services.RevenueService,
	stackAuthMiddleware *middlewares.StackAuthMiddleware,
	cacheMiddleware *middlewares.CacheMiddleware,
) {
	revenueRouteGroup := s.Group("/api/v1/revenue")
	revenueRouteGroup.Use(stackAuthMiddleware.Middleware())

	lowFrequencyTTL := 5 * time.Minute
	mediumFrequencyTTL := 2 * time.Minute

	cachePrefix := string(cache.RevenueCacheKey)

	server.GroupGET(revenueRouteGroup, "/dashboard",
		revenueService.GetDashboard,
		cacheMiddleware.Middleware(middlewares.CacheConfig{
			TTL:       mediumFrequencyTTL,
			KeyPrefix: cachePrefix,
		}))

	server.GroupGET(revenueRouteGroup, "/attribution/origin",
		revenueService.GetAttributionByOrigin,
		cacheMiddleware.Middleware(middlewares.CacheConfig{
			TTL:       lowFrequencyTTL,
			KeyPrefix: cachePrefix,
		}))

	server.GroupGET(revenueRouteGroup, "/attribution/utm",
		revenueService.GetAttributionByUTM,
		cacheMiddleware.Middleware(middlewares.CacheConfig{
			TTL:       lowFrequencyTTL,
			KeyPrefix: cachePrefix,
		}))

	server.GroupGET(revenueRouteGroup, "/timeline",
		revenueService.GetTimeline,
		cacheMiddleware.Middleware(middlewares.CacheConfig{
			TTL:       mediumFrequencyTTL,
			KeyPrefix: cachePrefix,
		}))

	server.GroupGET(revenueRouteGroup, "/customers/top",
		revenueService.GetTopCustomers,
		cacheMiddleware.Middleware(middlewares.CacheConfig{
			TTL:       mediumFrequencyTTL,
			KeyPrefix: cachePrefix,
		}))

	server.GroupGET(revenueRouteGroup, "/customers/profile", revenueService.GetCustomerProfile)

	server.GroupGET(revenueRouteGroup, "/conversion/metrics",
		revenueService.GetConversionMetrics,
		cacheMiddleware.Middleware(middlewares.CacheConfig{
			TTL:       lowFrequencyTTL,
			KeyPrefix: cachePrefix,
		}))
}

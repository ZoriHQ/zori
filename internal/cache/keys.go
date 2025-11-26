package cache

type CacheKey string

const (
	ProjectCacheKey          CacheKey = "project"
	AnalyticsCacheKey        CacheKey = "analytics"
	RevenueCacheKey          CacheKey = "revenue"
	EventDedupeCacheKey      CacheKey = "event:dedupe"
	LiveSessionsCacheKey     CacheKey = "live:sessions"
	LiveCleanupLockCacheKey  CacheKey = "live:cleanup:lock"
)

func (k CacheKey) FromValue(value string) string {
	return string(k) + ":" + value
}

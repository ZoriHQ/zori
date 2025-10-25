package cache

type CacheKey string

const (
	ProjectCacheKey   CacheKey = "project"
	AnalyticsCacheKey CacheKey = "analytics"
	RevenueCacheKey   CacheKey = "revenue"
)

func (k CacheKey) FromValue(value string) string {
	return string(k) + ":" + value
}

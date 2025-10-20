package cache

type CacheKey string

const (
	ProjectCacheKey CacheKey = "project"
)

func (k CacheKey) FromValue(value string) string {
	return string(k) + ":" + value
}

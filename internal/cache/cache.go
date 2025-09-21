package cache

import "time"

type CacheService struct{}

func NewCacheService() *CacheService {
	return &CacheService{}
}

func (s *CacheService) Get(key string) (string, error) {
	return "", nil
}

func (s *CacheService) Set(key string, value string) error {
	return nil
}

func (s *CacheService) Delete(key string) error {
	return nil
}

func (s *CacheService) Expire(key string, duration time.Duration) error {
	return nil
}

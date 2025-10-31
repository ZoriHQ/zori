package cache

import (
	"context"
	"encoding/json"
	"time"
	"zori/internal/config"

	"github.com/redis/go-redis/v9"
)

type CacheService struct {
	redisClient redis.UniversalClient
}

func NewCacheService(conf *config.Config) *CacheService {
	rdb := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:    []string{conf.RedisADDS},
		Password: conf.RedisPASS,
		DB:       0,
	})

	pingCmdResult := rdb.Ping(context.Background())
	if err := pingCmdResult.Err(); err != nil {
		panic(err)
	}

	return &CacheService{
		redisClient: rdb,
	}
}

func (s *CacheService) Get(ctx context.Context, key string) (*string, error) {
	var result string
	err := s.redisClient.Get(ctx, key).Scan(&result)
	switch {
	case err == redis.Nil:
		return nil, nil
	case err != nil:
		return nil, err
	default:
		return &result, nil
	}
}

func (s *CacheService) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	jsonValue, err := json.Marshal(value)
	if err != nil {
		return err
	}

	err = s.redisClient.Set(ctx, key, jsonValue, ttl).Err()
	if err != nil {
		return err
	}

	return nil
}

func (s *CacheService) Delete(ctx context.Context, key string) error {
	return nil
}

func (s *CacheService) Expire(key string, duration time.Duration) error {
	return nil
}

// SetNX sets a key-value pair only if the key does not already exist (atomic operation).
// Returns true if the key was set, false if it already existed.
// This is useful for implementing distributed locks or deduplication.
func (s *CacheService) SetNX(ctx context.Context, key string, value any, ttl time.Duration) (bool, error) {
	jsonValue, err := json.Marshal(value)
	if err != nil {
		return false, err
	}

	result, err := s.redisClient.SetNX(ctx, key, jsonValue, ttl).Result()
	if err != nil {
		return false, err
	}

	return result, nil
}

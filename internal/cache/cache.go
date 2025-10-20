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

func NewCacheService(conf config.Config) *CacheService {
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
	if err != nil {
		return nil, err
	}

	return &result, nil
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

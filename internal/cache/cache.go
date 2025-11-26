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

func (s *CacheService) FlushAll(ctx context.Context) error {
	return s.redisClient.FlushAll(ctx).Err()
}

func (s *CacheService) ZAdd(ctx context.Context, key string, score float64, member string) (bool, error) {
	result, err := s.redisClient.ZAdd(ctx, key, redis.Z{
		Score:  score,
		Member: member,
	}).Result()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

func (s *CacheService) ZRem(ctx context.Context, key string, member string) (bool, error) {
	result, err := s.redisClient.ZRem(ctx, key, member).Result()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

func (s *CacheService) ZRemRangeByScore(ctx context.Context, key string, min, max string) (int64, error) {
	return s.redisClient.ZRemRangeByScore(ctx, key, min, max).Result()
}

func (s *CacheService) ZCard(ctx context.Context, key string) (int64, error) {
	return s.redisClient.ZCard(ctx, key).Result()
}

func (s *CacheService) ZScore(ctx context.Context, key string, member string) (*float64, error) {
	result, err := s.redisClient.ZScore(ctx, key, member).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *CacheService) Keys(ctx context.Context, pattern string) ([]string, error) {
	var keys []string
	iter := s.redisClient.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	return keys, nil
}

package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache implements Cache backed by a Redis client.
type RedisCache struct {
	client redis.UniversalClient
}

// NewRedisCache creates a new RedisCache instance wrapping the given redis client.
func NewRedisCache(client redis.UniversalClient) *RedisCache {
	return &RedisCache{client: client}
}

// Client returns the underlying redis client if needed for inspection or tests.
func (r *RedisCache) Client() redis.UniversalClient {
	return r.client
}

func (r *RedisCache) Get(ctx context.Context, key string, dest any) (bool, error) {
	if r == nil || r.client == nil {
		return false, errors.New("redis client not initialized")
	}

	val, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		return false, err
	}

	if err := json.Unmarshal(val, dest); err != nil {
		return false, err
	}

	return true, nil
}

func (r *RedisCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	if r == nil || r.client == nil {
		return errors.New("redis client not initialized")
	}

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return r.client.Set(ctx, key, data, ttl).Err()
}

func (r *RedisCache) Delete(ctx context.Context, keys ...string) error {
	if r == nil || r.client == nil || len(keys) == 0 {
		return nil
	}

	validKeys := make([]string, 0, len(keys))
	for _, k := range keys {
		if k != "" {
			validKeys = append(validKeys, k)
		}
	}
	if len(validKeys) == 0 {
		return nil
	}

	return r.client.Del(ctx, validKeys...).Err()
}

func (r *RedisCache) BumpVersion(ctx context.Context, key string) (int64, error) {
	if r == nil || r.client == nil {
		return 0, errors.New("redis client not initialized")
	}
	return r.client.Incr(ctx, key).Result()
}

func (r *RedisCache) GetVersion(ctx context.Context, key string) (int64, error) {
	if r == nil || r.client == nil {
		return 0, errors.New("redis client not initialized")
	}
	v, err := r.client.Get(ctx, key).Int64()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, nil
		}
		return 0, err
	}
	return v, nil
}

func (r *RedisCache) Ping(ctx context.Context) error {
	if r == nil || r.client == nil {
		return errors.New("redis client not initialized")
	}
	return r.client.Ping(ctx).Err()
}

func (r *RedisCache) Close() error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.Close()
}


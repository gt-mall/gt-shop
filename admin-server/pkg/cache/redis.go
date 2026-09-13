package cache

import (
	"context"
	"encoding/json"
	"errors"
	"log"
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
		log.Printf("[Cache] redis GET error (key=%s): %v", key, err)
		return false, err
	}

	if err := json.Unmarshal(val, dest); err != nil {
		log.Printf("[Cache] redis unmarshal error (key=%s): %v", key, err)
		return false, err
	}

	log.Printf("[Cache] cache hit (key=%s)", key)
	return true, nil
}

func (r *RedisCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	if r == nil || r.client == nil {
		return errors.New("redis client not initialized")
	}

	data, err := json.Marshal(value)
	if err != nil {
		log.Printf("[Cache] redis marshal error (key=%s): %v", key, err)
		return err
	}

	if err := r.client.Set(ctx, key, data, ttl).Err(); err != nil {
		log.Printf("[Cache] redis SET error (key=%s): %v", key, err)
		return err
	}

	return nil
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

	if err := r.client.Del(ctx, validKeys...).Err(); err != nil {
		log.Printf("[Cache] redis DEL error (keys=%v): %v", validKeys, err)
		return err
	}

	return nil
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


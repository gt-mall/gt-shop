package bootstrap

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"mall/admin_core/pkg/cache"
)

// NewCache initializes the cache layer based on configuration.
// If Redis is disabled (RedisEnabled=false), NoopCache is automatically returned.
// If Redis is enabled (RedisEnabled=true) but connection fails, an explicit error
// is logged, and the system gracefully falls back to NoopCache without crashing.
func NewCache(ctx context.Context, config Config) cache.Cache {
	if !config.RedisEnabled {
		return cache.NewNoop()
	}

	client := redis.NewClient(&redis.Options{
		Addr:     config.RedisAddr,
		Password: config.RedisPassword,
		DB:       config.RedisDB,
	})

	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx).Err(); err != nil {
		log.Printf("[Bootstrap] redis connection failed (addr=%s, db=%d): %v; falling back to NoopCache",
			config.RedisAddr, config.RedisDB, err)
		_ = client.Close()
		return cache.NewNoop()
	}

	log.Printf("[Bootstrap] redis cache initialized successfully (addr=%s, db=%d)",
		config.RedisAddr, config.RedisDB)
	return cache.NewRedisCache(client)
}


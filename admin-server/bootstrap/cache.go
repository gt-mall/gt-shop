package bootstrap

import (
	"context"
	"log"

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

	c, err := cache.NewRueidisCache(cache.Config{
		Addr:     config.RedisAddr,
		Password: config.RedisPassword,
		DB:       config.RedisDB,
	})
	if err != nil {
		log.Printf("[Bootstrap] Redis connection failed (addr=%s, db=%d): %v; Redis cache disabled due to startup connection failure",
			config.RedisAddr, config.RedisDB, err)
		return cache.NewNoop()
	}

	log.Printf("[Bootstrap] redis cache initialized successfully (addr=%s, db=%d)",
		config.RedisAddr, config.RedisDB)
	return c
}

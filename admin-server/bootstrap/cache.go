package bootstrap

import (
	"context"

	"mall/admin_core/pkg/cache"
)

// NewCache uses Redis when available and falls back to a no-op cache otherwise.
func NewCache(_ context.Context, config Config) cache.Cache {
	if !config.RedisEnabled {
		return cache.NewNoop()
	}
	c, err := cache.NewRueidisCache(cache.Config{
		Addr:     config.RedisAddr,
		Password: config.RedisPassword,
		DB:       config.RedisDB,
	})
	if err != nil {
		return cache.NewNoop()
	}
	return c
}

package bootstrap_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"mall/admin_core/bootstrap"
	"mall/admin_core/pkg/cache"
)

func TestNewCacheDisabled(t *testing.T) {
	cfg := bootstrap.Config{
		RedisEnabled: false,
	}
	c := bootstrap.NewCache(context.Background(), cfg)
	defer c.Close()

	if _, ok := c.(*cache.NoopCache); !ok {
		t.Fatalf("expected *cache.NoopCache when disabled, got %T", c)
	}
}

func TestNewCacheConnectionFailureFallback(t *testing.T) {
	cfg := bootstrap.Config{
		RedisEnabled: true,
		RedisAddr:    "127.0.0.1:1", // guaranteed unreachable/connection refused
	}
	c := bootstrap.NewCache(context.Background(), cfg)
	defer c.Close()

	if _, ok := c.(*cache.NoopCache); !ok {
		t.Fatalf("expected fallback to *cache.NoopCache when connection fails, got %T", c)
	}
}

func TestNewCacheEnabledSuccess(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	})
	pingCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx).Err(); err != nil {
		client.Close()
		t.Skipf("skipping live redis bootstrap test: local redis not reachable: %v", err)
		return
	}
	client.Close()

	cfg := bootstrap.Config{
		RedisEnabled: true,
		RedisAddr:    "127.0.0.1:6379",
		RedisDB:      15,
	}
	c := bootstrap.NewCache(context.Background(), cfg)
	defer c.Close()

	if reflect.TypeOf(c).String() != "*cache.RedisCache" {
		t.Fatalf("expected *cache.RedisCache when enabled, got %T", c)
	}
}


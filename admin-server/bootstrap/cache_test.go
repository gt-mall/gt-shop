package bootstrap_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/redis/rueidis"
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
	if !c.IsDisabled() {
		t.Fatalf("expected cache.IsDisabled() to be true")
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
	if !c.IsDisabled() {
		t.Fatalf("expected fallback cache to be disabled")
	}
}

func TestNewCacheEnabledSuccess(t *testing.T) {
	client, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress: []string{"127.0.0.1:6379"},
	})
	if err != nil {
		t.Skipf("skipping live redis bootstrap test: rueidis client init failed: %v", err)
		return
	}

	pingCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Do(pingCtx, client.B().Ping().Build()).Error(); err != nil {
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

	if reflect.TypeOf(c).String() != "*cache.RueidisCache" {
		t.Fatalf("expected *cache.RueidisCache when enabled, got %T", c)
	}
}

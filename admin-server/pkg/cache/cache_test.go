package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"mall/admin_core/pkg/cache"
)

type testItem struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func TestNoopCache(t *testing.T) {
	ctx := context.Background()
	c := cache.NewNoop()

	// 1. Get always returns miss (false, nil)
	var item testItem
	hit, err := c.Get(ctx, "any:key", &item)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if hit {
		t.Fatalf("expected cache miss, got hit")
	}

	// 2. Set returns nil
	if err := c.Set(ctx, "any:key", testItem{ID: 1, Name: "test"}, time.Minute); err != nil {
		t.Fatalf("expected nil error on Set, got %v", err)
	}

	// 3. Delete returns nil
	if err := c.Delete(ctx, "any:key", "other:key"); err != nil {
		t.Fatalf("expected nil error on Delete, got %v", err)
	}

	// 4. Ping and Close return nil
	if err := c.Ping(ctx); err != nil {
		t.Fatalf("expected nil error on Ping, got %v", err)
	}
	if err := c.Close(); err != nil {
		t.Fatalf("expected nil error on Close, got %v", err)
	}
}

func TestRedisCacheBasic(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
		DB:   15,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("skipping RedisCache tests: local redis not reachable: %v", err)
		return
	}
	defer client.Close()

	c := cache.NewRedisCache(client)
	testKey := "mall:test:item:1001"
	defer c.Delete(ctx, testKey)

	// Clean before test
	_ = c.Delete(ctx, testKey)

	// 1. Get before set -> miss
	var dest testItem
	hit, err := c.Get(ctx, testKey, &dest)
	if err != nil {
		t.Fatalf("unexpected error on cache get: %v", err)
	}
	if hit {
		t.Fatalf("expected miss before set")
	}

	// 2. Set
	src := testItem{ID: 1001, Name: "iPhone 16"}
	if err := c.Set(ctx, testKey, src, 10*time.Second); err != nil {
		t.Fatalf("failed to set cache: %v", err)
	}

	// 3. Get after set -> hit
	var fetched testItem
	hit, err = c.Get(ctx, testKey, &fetched)
	if err != nil {
		t.Fatalf("unexpected error on cache get: %v", err)
	}
	if !hit {
		t.Fatalf("expected hit after set")
	}
	if fetched.ID != src.ID || fetched.Name != src.Name {
		t.Fatalf("data mismatch: expected %+v, got %+v", src, fetched)
	}

	// 4. Delete
	if err := c.Delete(ctx, testKey); err != nil {
		t.Fatalf("failed to delete cache: %v", err)
	}

	// 5. Get after delete -> miss
	hit, err = c.Get(ctx, testKey, &dest)
	if err != nil {
		t.Fatalf("unexpected error on cache get: %v", err)
	}
	if hit {
		t.Fatalf("expected miss after delete")
	}
}

func TestRedisCacheErrorHandling(t *testing.T) {
	ctx := context.Background()

	// 1. Nil client handling
	nilCache := cache.NewRedisCache(nil)
	var dest testItem
	hit, err := nilCache.Get(ctx, "k", &dest)
	if hit || err == nil {
		t.Fatalf("expected error on nil client Get")
	}
	if err := nilCache.Set(ctx, "k", dest, time.Minute); err == nil {
		t.Fatalf("expected error on nil client Set")
	}
	if err := nilCache.Delete(ctx, "k"); err != nil {
		t.Fatalf("expected nil on delete for nil client")
	}
	if err := nilCache.Ping(ctx); err == nil {
		t.Fatalf("expected error on nil client Ping")
	}
	if err := nilCache.Close(); err != nil {
		t.Fatalf("expected nil on close for nil client")
	}

	// 2. Closed client handling
	client := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
		DB:   15,
	})
	_ = client.Close()
	closedCache := cache.NewRedisCache(client)

	hit, err = closedCache.Get(ctx, "k", &dest)
	if hit || err == nil {
		t.Fatalf("expected error on closed client Get")
	}
	if err := closedCache.Set(ctx, "k", dest, time.Minute); err == nil {
		t.Fatalf("expected error on closed client Set")
	}
	if err := closedCache.Delete(ctx, "k"); err == nil {
		t.Fatalf("expected error on closed client Delete")
	}
}

func TestCacheKeysAndTTL(t *testing.T) {
	if cache.ProductDetailKey(1) != "mall:product:detail:1" {
		t.Fatalf("unexpected product detail key: %s", cache.ProductDetailKey(1))
	}
	if cache.MiniappProductDetailKey(1) != "mall:miniapp:product:detail:1" {
		t.Fatalf("unexpected miniapp product detail key: %s", cache.MiniappProductDetailKey(1))
	}
	if cache.CategoryTreeKey() != "mall:category:tree" {
		t.Fatalf("unexpected category tree key: %s", cache.CategoryTreeKey())
	}
	if cache.CategoryOptionsKey() != "mall:category:options" {
		t.Fatalf("unexpected category options key: %s", cache.CategoryOptionsKey())
	}
	if cache.RegionTreeKey() != "mall:region:tree" {
		t.Fatalf("unexpected region tree key: %s", cache.RegionTreeKey())
	}
	if cache.SystemConfigKey("shop_basic") != "mall:system:config:shop_basic" {
		t.Fatalf("unexpected system config key: %s", cache.SystemConfigKey("shop_basic"))
	}

	listKey1 := cache.ProductListKey("page=1&size=20")
	listKey2 := cache.ProductListKey("page=1&size=20")
	listKey3 := cache.ProductListKey("page=2&size=20")
	if listKey1 != listKey2 {
		t.Fatalf("expected same list key for same params")
	}
	if listKey1 == listKey3 {
		t.Fatalf("expected different list key for different params")
	}

	if cache.TTLProductDetail != 120*time.Second {
		t.Fatalf("expected 120s TTL for product detail")
	}
	if cache.TTLProductList != 60*time.Second {
		t.Fatalf("expected 60s TTL for product list")
	}
	if cache.TTLCategoryTree != 10*time.Minute {
		t.Fatalf("expected 10m TTL for category tree")
	}
	if cache.TTLRegionTree != 24*time.Hour {
		t.Fatalf("expected 24h TTL for region tree")
	}
	if cache.TTLSystemConfig != 10*time.Minute {
		t.Fatalf("expected 10m TTL for system config")
	}
}


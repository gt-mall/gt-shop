package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
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

	// 4. BumpVersion and GetVersion return nil
	v, err := c.BumpVersion(ctx, "any:ver:key")
	if err != nil || v != 1 {
		t.Fatalf("expected 1, nil on BumpVersion, got %d, %v", v, err)
	}
	gv, err := c.GetVersion(ctx, "any:ver:key")
	if err != nil || gv != 0 {
		t.Fatalf("expected 0, nil on GetVersion, got %d, %v", gv, err)
	}

	// 5. Ping and Close return nil
	if err := c.Ping(ctx); err != nil {
		t.Fatalf("expected nil error on Ping, got %v", err)
	}
	if err := c.Close(); err != nil {
		t.Fatalf("expected nil error on Close, got %v", err)
	}
}

func TestRedisCacheBasic(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	t.Cleanup(func() {
		client.Close()
	})

	ctx := context.Background()
	c := cache.NewRedisCache(client)
	testKey := "mall:test:item:1001"

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

	// 6. BumpVersion and GetVersion
	verKey := "mall:version:test:key"
	v0, err := c.GetVersion(ctx, verKey)
	if err != nil || v0 != 0 {
		t.Fatalf("expected v0=0, got %d, err=%v", v0, err)
	}

	v1, err := c.BumpVersion(ctx, verKey)
	if err != nil || v1 != 1 {
		t.Fatalf("expected v1=1, got %d, err=%v", v1, err)
	}

	v1Check, err := c.GetVersion(ctx, verKey)
	if err != nil || v1Check != 1 {
		t.Fatalf("expected v1Check=1, got %d, err=%v", v1Check, err)
	}

	v2, err := c.BumpVersion(ctx, verKey)
	if err != nil || v2 != 2 {
		t.Fatalf("expected v2=2, got %d, err=%v", v2, err)
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
	if _, err := nilCache.BumpVersion(ctx, "k"); err == nil {
		t.Fatalf("expected error on nil client BumpVersion")
	}
	if _, err := nilCache.GetVersion(ctx, "k"); err == nil {
		t.Fatalf("expected error on nil client GetVersion")
	}
	if err := nilCache.Ping(ctx); err == nil {
		t.Fatalf("expected error on nil client Ping")
	}
	if err := nilCache.Close(); err != nil {
		t.Fatalf("expected nil on close for nil client")
	}

	// 2. Closed client handling
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
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
	if _, err := closedCache.BumpVersion(ctx, "k"); err == nil {
		t.Fatalf("expected error on closed client BumpVersion")
	}
	if _, err := closedCache.GetVersion(ctx, "k"); err == nil {
		t.Fatalf("expected error on closed client GetVersion")
	}
}

func TestCacheKeysAndTTL(t *testing.T) {
	// 1. Version keys
	if cache.MiniappProductListVersionKey() != "mall:version:miniapp:product:list" {
		t.Fatalf("unexpected list version key: %s", cache.MiniappProductListVersionKey())
	}
	if cache.MiniappProductDetailVersionKey(123) != "mall:version:miniapp:product:detail:123" {
		t.Fatalf("unexpected detail version key: %s", cache.MiniappProductDetailVersionKey(123))
	}

	// 2. Versioned keys
	if cache.MiniappProductDetailKey(123, 1) != "mall:miniapp:product:detail:123:v1" {
		t.Fatalf("unexpected miniapp product detail key: %s", cache.MiniappProductDetailKey(123, 1))
	}
	listKey1 := cache.MiniappProductListKey(1, "p=1:s=20")
	listKey2 := cache.MiniappProductListKey(1, "p=1:s=20")
	listKey3 := cache.MiniappProductListKey(2, "p=1:s=20")
	if listKey1 != listKey2 {
		t.Fatalf("expected same key for same version and params")
	}
	if listKey1 == listKey3 {
		t.Fatalf("expected different key for different versions")
	}

	// 3. Other entity keys
	if cache.CategoryTreeKey() != "mall:category:tree" {
		t.Fatalf("unexpected category tree key: %s", cache.CategoryTreeKey())
	}
	if cache.CategoryOptionsKey() != "mall:category:options" {
		t.Fatalf("unexpected category options key: %s", cache.CategoryOptionsKey())
	}
	if cache.SystemConfigKey("shop_basic") != "mall:system:config:shop_basic" {
		t.Fatalf("unexpected system config key: %s", cache.SystemConfigKey("shop_basic"))
	}
	if cache.PublicMpConfigKey() != "mall:system:config:public_mp" {
		t.Fatalf("unexpected public mp key: %s", cache.PublicMpConfigKey())
	}

	// 4. TTL constants
	if cache.TTLProductDetail != 120*time.Second {
		t.Fatalf("expected 120s TTL for product detail")
	}
	if cache.TTLProductList != 60*time.Second {
		t.Fatalf("expected 60s TTL for product list")
	}
	if cache.TTLProductNotFound != 20*time.Second {
		t.Fatalf("expected 20s TTL for not found product")
	}
	if cache.TTLEmptyProductList != 20*time.Second {
		t.Fatalf("expected 20s TTL for empty product list")
	}
	if cache.TTLCategoryTree != 10*time.Minute {
		t.Fatalf("expected 10m TTL for category tree")
	}
	if cache.TTLSystemConfig != 10*time.Minute {
		t.Fatalf("expected 10m TTL for system config")
	}

	// 5. TTL With Jitter (+/- 10%)
	base := 100 * time.Second
	for i := 0; i < 20; i++ {
		jittered := cache.TTLWithJitter(base)
		if jittered < 89*time.Second || jittered > 111*time.Second {
			t.Fatalf("jittered TTL %v out of expected range [89s, 111s]", jittered)
		}
	}
}

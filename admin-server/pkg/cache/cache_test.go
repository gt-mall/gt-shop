package cache_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"mall/admin_core/pkg/cache"
)

type testItem struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func getTestRueidisCache(t *testing.T) (cache.Cache, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	c, err := cache.NewRueidisCache(cache.Config{
		Addr:         mr.Addr(),
		DisableCache: true,
		UseLuaLock:   true,
	})
	if err != nil {
		t.Fatalf("failed to create test rueidis cache: %v", err)
	}
	t.Cleanup(func() {
		_ = c.Close()
	})
	return c, mr
}

func TestNoopCache(t *testing.T) {
	ctx := context.Background()
	c := cache.NewNoop()

	if !c.IsDisabled() {
		t.Fatalf("expected NoopCache to be disabled")
	}

	// 1. Get directly executes loader
	loaderCalled := 0
	val, err := cache.Get(ctx, c, "any:key", time.Minute, func(ctx context.Context) (*testItem, error) {
		loaderCalled++
		return &testItem{ID: 10, Name: "NoopItem"}, nil
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if val == nil || val.ID != 10 || loaderCalled != 1 {
		t.Fatalf("expected loader result, got %+v (called=%d)", val, loaderCalled)
	}

	// 2. Delete returns nil
	if err := c.Delete(ctx, "any:key", "other:key"); err != nil {
		t.Fatalf("expected nil error on Delete, got %v", err)
	}

	// 3. Ping and Close return nil
	if err := c.Ping(ctx); err != nil {
		t.Fatalf("expected nil error on Ping, got %v", err)
	}
	if err := c.Close(); err != nil {
		t.Fatalf("expected nil error on Close, got %v", err)
	}
	if c.Client() != nil || c.Aside() != nil {
		t.Fatalf("expected nil client and aside on NoopCache")
	}
}

func TestRueidisCacheBasicAndCacheAside(t *testing.T) {
	c, mr := getTestRueidisCache(t)
	ctx := context.Background()
	testKey := "mall:test:item:1001"

	// 1. First Get -> Cache Miss -> loader executed
	loaderCalls := 0
	val1, err := cache.Get(ctx, c, testKey, 10*time.Second, func(innerCtx context.Context) (*testItem, error) {
		loaderCalls++
		return &testItem{ID: 1001, Name: "iPhone 16"}, nil
	})
	if err != nil {
		t.Fatalf("unexpected error on cache get 1: %v", err)
	}
	if val1 == nil || val1.ID != 1001 || val1.Name != "iPhone 16" {
		t.Fatalf("unexpected value 1: %+v", val1)
	}
	if loaderCalls != 1 {
		t.Fatalf("expected loader called once, got %d", loaderCalls)
	}

	// Verify key exists in miniredis
	if !mr.Exists(testKey) {
		t.Fatalf("expected key %s to exist in redis", testKey)
	}

	// 2. Second Get -> Cache Hit -> loader NOT executed
	val2, err := cache.Get(ctx, c, testKey, 10*time.Second, func(innerCtx context.Context) (*testItem, error) {
		loaderCalls++
		return &testItem{ID: 1001, Name: "Different"}, nil
	})
	if err != nil {
		t.Fatalf("unexpected error on cache get 2: %v", err)
	}
	if val2 == nil || val2.Name != "iPhone 16" {
		t.Fatalf("expected cached 'iPhone 16', got %+v", val2)
	}
	if loaderCalls != 1 {
		t.Fatalf("expected loader NOT called on cache hit, got calls=%d", loaderCalls)
	}

	// 3. Delete key -> next Get misses again
	if err := c.Delete(ctx, testKey); err != nil {
		t.Fatalf("failed to delete key: %v", err)
	}
	if mr.Exists(testKey) {
		t.Fatalf("expected key %s to be deleted", testKey)
	}

	val3, err := cache.Get(ctx, c, testKey, 10*time.Second, func(innerCtx context.Context) (*testItem, error) {
		loaderCalls++
		return &testItem{ID: 1001, Name: "iPhone 16 Refreshed"}, nil
	})
	if err != nil {
		t.Fatalf("unexpected error on cache get 3: %v", err)
	}
	if val3 == nil || val3.Name != "iPhone 16 Refreshed" {
		t.Fatalf("unexpected value 3: %+v", val3)
	}
	if loaderCalls != 2 {
		t.Fatalf("expected loader called on 2nd miss, got calls=%d", loaderCalls)
	}
}

func TestRueidisCacheNegativeCaching(t *testing.T) {
	c, mr := getTestRueidisCache(t)
	ctx := context.Background()
	missingKey := "mall:test:missing:9999"

	loaderCalls := 0
	// 1. First Get returns (nil, nil) -> negative cache sentinel stored
	val1, err := cache.Get(ctx, c, missingKey, time.Minute, func(innerCtx context.Context) (*testItem, error) {
		loaderCalls++
		return nil, nil // not found
	})
	if err != nil {
		t.Fatalf("unexpected error on negative cache get 1: %v", err)
	}
	if val1 != nil {
		t.Fatalf("expected nil result, got %+v", val1)
	}
	if loaderCalls != 1 {
		t.Fatalf("expected loader called once, got %d", loaderCalls)
	}

	// Key should exist in redis storing the sentinel
	if !mr.Exists(missingKey) {
		t.Fatalf("expected negative cache key to exist in redis")
	}

	// 2. Second Get within TTL returns nil directly from cache
	val2, err := cache.Get(ctx, c, missingKey, time.Minute, func(innerCtx context.Context) (*testItem, error) {
		loaderCalls++
		return &testItem{ID: 9999, Name: "Should Not Call"}, nil
	})
	if err != nil {
		t.Fatalf("unexpected error on negative cache get 2: %v", err)
	}
	if val2 != nil {
		t.Fatalf("expected nil result from negative cache, got %+v", val2)
	}
	if loaderCalls != 1 {
		t.Fatalf("expected loader not called on negative cache hit, got %d", loaderCalls)
	}
}

func TestRueidisCacheRuntimeFallback(t *testing.T) {
	mr := miniredis.RunT(t)
	c, err := cache.NewRueidisCache(cache.Config{
		Addr:         mr.Addr(),
		DisableCache: true,
		UseLuaLock:   true,
	})
	if err != nil {
		t.Fatalf("failed to init cache: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	key := "mall:test:fallback:1"

	// Close miniredis to simulate sudden Redis failure during runtime
	mr.Close()

	loaderCalled := false
	val, err := cache.Get(ctx, c, key, 500*time.Millisecond, func(innerCtx context.Context) (*testItem, error) {
		loaderCalled = true
		return &testItem{ID: 1, Name: "DB Fallback"}, nil
	})
	if err != nil {
		t.Fatalf("expected fallback to succeed without error, got %v", err)
	}
	if !loaderCalled || val == nil || val.Name != "DB Fallback" {
		t.Fatalf("expected fallback to loader, got %+v", val)
	}
}

func TestRueidisCacheLoaderErrorPropagates(t *testing.T) {
	c, _ := getTestRueidisCache(t)
	ctx := context.Background()
	key := "mall:test:err:1"

	expectedErr := errors.New("db error")
	val, err := cache.Get(ctx, c, key, time.Minute, func(innerCtx context.Context) (*testItem, error) {
		return nil, expectedErr
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
	if val != nil {
		t.Fatalf("expected nil val on loader error, got %v", val)
	}
}

func TestProductListGeneration(t *testing.T) {
	c, _ := getTestRueidisCache(t)
	ctx := context.Background()

	// Initial generation is 0
	gen0, err := cache.GetProductListGeneration(ctx, c)
	if err != nil || gen0 != 0 {
		t.Fatalf("expected initial generation 0, got %d (err=%v)", gen0, err)
	}

	// Bump generation
	newGen, err := cache.BumpProductListGeneration(ctx, c)
	if err != nil || newGen != 1 {
		t.Fatalf("expected bumped gen 1, got %d (err=%v)", newGen, err)
	}

	// Check generation again
	gen1, err := cache.GetProductListGeneration(ctx, c)
	if err != nil || gen1 != 1 {
		t.Fatalf("expected generation 1, got %d (err=%v)", gen1, err)
	}

	// Bump again
	newGen2, err := cache.BumpProductListGeneration(ctx, c)
	if err != nil || newGen2 != 2 {
		t.Fatalf("expected bumped gen 2, got %d (err=%v)", newGen2, err)
	}

	// ProductDetailGeneration helper
	dGen0, err := cache.GetProductDetailGeneration(ctx, c, 100)
	if err != nil || dGen0 != 0 {
		t.Fatalf("expected initial detail gen 0, got %d (err=%v)", dGen0, err)
	}
	dGen1, err := cache.BumpProductDetailGeneration(ctx, c, 100)
	if err != nil || dGen1 != 1 {
		t.Fatalf("expected bumped detail gen 1, got %d (err=%v)", dGen1, err)
	}

	// SystemConfigGeneration helper
	sGen0, err := cache.GetSystemConfigGeneration(ctx, c, "shop_basic")
	if err != nil || sGen0 != 0 {
		t.Fatalf("expected initial config gen 0, got %d (err=%v)", sGen0, err)
	}
	sGen1, err := cache.BumpSystemConfigGeneration(ctx, c, "shop_basic")
	if err != nil || sGen1 != 1 {
		t.Fatalf("expected bumped config gen 1, got %d (err=%v)", sGen1, err)
	}
}

func TestCorruptedJSONKeyDeletion(t *testing.T) {
	c, mr := getTestRueidisCache(t)
	ctx := context.Background()
	badKey := "mall:test:bad:json"

	// Directly inject corrupted JSON string into redis
	mr.Set(badKey, "{invalid-json-data---")

	loaderCalls := 0
	val, err := cache.Get(ctx, c, badKey, time.Minute, func(innerCtx context.Context) (*testItem, error) {
		loaderCalls++
		return &testItem{ID: 888, Name: "Recovered"}, nil
	})
	if err != nil {
		t.Fatalf("expected fallback to succeed, got error: %v", err)
	}
	if val == nil || val.Name != "Recovered" || loaderCalls != 1 {
		t.Fatalf("expected recovered value from loader, got %+v (calls=%d)", val, loaderCalls)
	}

	// The corrupted key should have been deleted
	if mr.Exists(badKey) {
		t.Fatalf("expected corrupted key to be deleted from redis")
	}
}

func TestCacheKeysAndTTL(t *testing.T) {
	// Keys
	if k := cache.ProductDetailGenerationKey(123); k != "mall:miniapp:product:detail:123:gen" {
		t.Fatalf("unexpected detail gen key: %s", k)
	}
	if k := cache.ProductDetailKey(123, 5); k != "mall:miniapp:product:detail:123:g5" {
		t.Fatalf("unexpected detail key: %s", k)
	}
	if k := cache.ProductListGenerationKey(); k != "mall:miniapp:product:list:gen" {
		t.Fatalf("unexpected list gen key: %s", k)
	}
	if k := cache.SystemConfigGenerationKey("shop_basic"); k != "mall:system:config:shop_basic:gen" {
		t.Fatalf("unexpected system config gen key: %s", k)
	}
	if k := cache.SystemConfigKey("shop_basic", 3); k != "mall:system:config:shop_basic:g3" {
		t.Fatalf("unexpected system config key: %s", k)
	}

	// TTLs
	if cache.TTLProductDetail != 120*time.Second {
		t.Fatalf("expected TTLProductDetail 120s, got %v", cache.TTLProductDetail)
	}
	if cache.TTLProductList != 60*time.Second {
		t.Fatalf("expected TTLProductList 60s, got %v", cache.TTLProductList)
	}
	if cache.TTLProductNotFound != 20*time.Second {
		t.Fatalf("expected TTLProductNotFound 20s, got %v", cache.TTLProductNotFound)
	}
	if cache.TTLEmptyProductList != 20*time.Second {
		t.Fatalf("expected TTLEmptyProductList 20s, got %v", cache.TTLEmptyProductList)
	}
	if cache.TTLSystemConfig != 10*time.Minute {
		t.Fatalf("expected TTLSystemConfig 10m, got %v", cache.TTLSystemConfig)
	}
}

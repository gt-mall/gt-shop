package cache

import (
	"context"
	"encoding/json"
	"log"
	"reflect"
	"sync/atomic"
	"time"

	"github.com/redis/rueidis"
	"github.com/redis/rueidis/rueidisaside"
)

// Sentinel value stored in cache for not found entities (negative caching).
const sentinelNotFound = "__mall_nil__"

// Cache defines the thin project-level interface for caching operations.
type Cache interface {
	// Delete removes one or more keys from the cache.
	Delete(ctx context.Context, keys ...string) error

	// Client returns the underlying rueidis.Client, or nil if noop.
	Client() rueidis.Client

	// Aside returns the underlying rueidisaside.CacheAsideClient, or nil if noop.
	Aside() rueidisaside.CacheAsideClient

	// Ping checks the health of the cache connection.
	Ping(ctx context.Context) error

	// Close releases any resources associated with the cache.
	Close() error

	// IsDisabled returns true if caching is disabled or uninitialized.
	IsDisabled() bool
}

var lastRedisErrLog atomic.Int64

func logRedisRuntimeError(key string, err error) {
	now := time.Now().Unix()
	last := lastRedisErrLog.Load()
	if now-last >= 5 && lastRedisErrLog.CompareAndSwap(last, now) {
		log.Printf("[Cache] Redis runtime error on key %s: %v (rate-limited, max 1 log/5s); falling back to DB loader", key, err)
	}
}

// Get executes a Cache-Aside fetch for an entity of type T.
// If caching is disabled or nil, it directly invokes the loader function.
// If caching is enabled, it uses rueidisaside for client-side caching, singleflight/stampede protection,
// and stale write prevention.
// If loader returns a nil value (e.g. (*T)(nil)), a negative cache sentinel is stored. Specific negative TTL
// can be configured by the loader calling SetTTL(innerCtx, customTTL).
// If Redis encounters a temporary runtime failure, it rate-limits warning logs and gracefully falls back to loader.
func Get[T any](ctx context.Context, c Cache, key string, ttl time.Duration, loader func(ctx context.Context) (T, error)) (T, error) {
	var zero T
	if ctx == nil {
		ctx = context.Background()
	}
	if c == nil || c.IsDisabled() || c.Aside() == nil {
		return loader(ctx)
	}

	var loaderErr error
	asideVal, err := c.Aside().Get(ctx, ttl, key, func(innerCtx context.Context, k string) (string, error) {
		val, lErr := loader(innerCtx)
		if lErr != nil {
			loaderErr = lErr
			return "", lErr
		}
		if isNil(val) {
			return sentinelNotFound, nil
		}
		b, err := json.Marshal(val)
		if err != nil {
			return "", err
		}
		return string(b), nil
	})

	// If loader returned an error during execution, propagate it without fallback
	if loaderErr != nil {
		return zero, loaderErr
	}

	// If Redis runtime error occurred (e.g. connection lost, timeout), gracefully fallback to DB loader
	if err != nil {
		logRedisRuntimeError(key, err)
		return loader(ctx)
	}

	// Handle negative cache hit
	if asideVal == sentinelNotFound {
		return zero, nil
	}

	// Unmarshal cached JSON. If corrupted, best-effort delete the bad key to avoid persistent bad cache.
	var result T
	if err := json.Unmarshal([]byte(asideVal), &result); err != nil {
		log.Printf("[Cache] JSON unmarshal error on key %s: %v; deleting corrupted key and falling back to DB loader", key, err)
		delCtx, delCancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		_ = c.Delete(delCtx, key)
		delCancel()
		return loader(ctx)
	}

	return result, nil
}

// SetTTL overrides the TTL for the current cache entry inside a loader callback.
// For example, an empty list result can call SetTTL(ctx, TTLEmptyProductList).
func SetTTL(ctx context.Context, ttl time.Duration) {
	rueidisaside.OverrideCacheTTL(ctx, ttl)
}

func isNil(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Map, reflect.Interface, reflect.Func:
		return rv.IsNil()
	}
	return false
}

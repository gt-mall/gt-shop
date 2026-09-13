package cache

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"time"
)

// KeyPrefix is the uniform prefix for all cache keys in this application.
const KeyPrefix = "mall:"

// Default TTLs for cached entities.
const (
	// TTLProductDetail is the TTL for miniapp product detail static cache (~120s).
	TTLProductDetail = 120 * time.Second

	// TTLProductList is the TTL for miniapp product list static cache (~60s).
	TTLProductList = 60 * time.Second

	// TTLProductNotFound is the negative cache TTL for non-existent or off-shelf products (20s).
	TTLProductNotFound = 20 * time.Second

	// TTLEmptyProductList is the negative cache TTL for empty list results (20s).
	TTLEmptyProductList = 20 * time.Second

	// TTLCategoryTree is the TTL for product category tree/options cache (10m).
	TTLCategoryTree = 10 * time.Minute

	// TTLSystemConfig is the TTL for public system/shop config cache (10m).
	TTLSystemConfig = 10 * time.Minute
)

// ProductDetailKey returns the direct stable cache key for miniapp product detail.
// Product detail uses direct key invalidation rather than versioning.
func ProductDetailKey(id int64) string {
	return fmt.Sprintf("%sminiapp:product:detail:%d", KeyPrefix, id)
}

// ProductListKey returns the cache key for miniapp product list with generation and query hash.
func ProductListKey(gen int64, paramStr string) string {
	h := md5.Sum([]byte(paramStr))
	return fmt.Sprintf("%sminiapp:product:list:g%d:%s", KeyPrefix, gen, hex.EncodeToString(h[:]))
}

// ProductListGenerationKey returns the key storing the current generation of product list.
func ProductListGenerationKey() string {
	return fmt.Sprintf("%sminiapp:product:list:gen", KeyPrefix)
}

// CategoryTreeKey returns the cache key for category tree list.
func CategoryTreeKey() string {
	return fmt.Sprintf("%scategory:tree", KeyPrefix)
}

// CategoryOptionsKey returns the cache key for category dropdown options.
func CategoryOptionsKey() string {
	return fmt.Sprintf("%scategory:options", KeyPrefix)
}

// SystemConfigKey returns the cache key for a given system config key.
func SystemConfigKey(key string) string {
	return fmt.Sprintf("%ssystem:config:%s", KeyPrefix, key)
}

// PublicMpConfigKey returns the cache key for public miniapp config (non-sensitive fields only).
func PublicMpConfigKey() string {
	return fmt.Sprintf("%ssystem:config:public_mp", KeyPrefix)
}

// GetProductListGeneration fetches the current product list generation counter.
// Returns 0 if cache is unavailable or key does not exist.
func GetProductListGeneration(ctx context.Context, c Cache) int64 {
	if c == nil || c.IsDisabled() || c.Client() == nil {
		return 0
	}
	if ctx == nil {
		ctx = context.Background()
	}
	v, err := c.Client().Do(ctx, c.Client().B().Get().Key(ProductListGenerationKey()).Build()).AsInt64()
	if err != nil {
		return 0
	}
	return v
}

// BumpProductListGeneration atomically increments the product list generation counter,
// causing subsequent list queries to read fresh data without expensive KEYS or SCAN operations.
func BumpProductListGeneration(ctx context.Context, c Cache) (int64, error) {
	if c == nil || c.IsDisabled() || c.Client() == nil {
		return 0, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return c.Client().Do(ctx, c.Client().B().Incr().Key(ProductListGenerationKey()).Build()).AsInt64()
}

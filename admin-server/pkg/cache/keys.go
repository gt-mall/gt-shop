package cache

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/redis/rueidis"
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

	// TTLSystemConfig is the TTL for public system/shop config cache (10m).
	TTLSystemConfig = 10 * time.Minute
)

// GetGeneration retrieves the generation counter for the given generationKey.
// If the key does not exist yet (cache uninitialized), it returns (0, nil).
// If Redis fails, it returns (0, err) so the caller knows NOT to guess 0 and can bypass cache.
func GetGeneration(ctx context.Context, c Cache, generationKey string) (int64, error) {
	if c == nil || c.IsDisabled() || c.Client() == nil {
		return 0, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	v, err := c.Client().Do(ctx, c.Client().B().Get().Key(generationKey).Build()).AsInt64()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			return 0, nil
		}
		return 0, err
	}
	return v, nil
}

// BumpGeneration atomically increments the generation counter for the given generationKey.
func BumpGeneration(ctx context.Context, c Cache, generationKey string) (int64, error) {
	if c == nil || c.IsDisabled() || c.Client() == nil {
		return 0, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return c.Client().Do(ctx, c.Client().B().Incr().Key(generationKey).Build()).AsInt64()
}

// ProductDetailGenerationKey returns the key storing the current generation of product detail.
func ProductDetailGenerationKey(id int64) string {
	return fmt.Sprintf("%sminiapp:product:detail:%d:gen", KeyPrefix, id)
}

// ProductDetailKey returns the generation-scoped cache key for miniapp product detail.
func ProductDetailKey(id int64, generation int64) string {
	return fmt.Sprintf("%sminiapp:product:detail:%d:g%d", KeyPrefix, id, generation)
}

// GetProductDetailGeneration fetches the current product detail generation counter.
func GetProductDetailGeneration(ctx context.Context, c Cache, id int64) (int64, error) {
	return GetGeneration(ctx, c, ProductDetailGenerationKey(id))
}

// BumpProductDetailGeneration atomically increments the product detail generation counter.
func BumpProductDetailGeneration(ctx context.Context, c Cache, id int64) (int64, error) {
	return BumpGeneration(ctx, c, ProductDetailGenerationKey(id))
}

// ProductListGenerationKey returns the key storing the current generation of product list.
func ProductListGenerationKey() string {
	return fmt.Sprintf("%sminiapp:product:list:gen", KeyPrefix)
}

// ProductListKey returns the cache key for miniapp product list with generation and query hash.
func ProductListKey(gen int64, paramStr string) string {
	h := md5.Sum([]byte(paramStr))
	return fmt.Sprintf("%sminiapp:product:list:g%d:%s", KeyPrefix, gen, hex.EncodeToString(h[:]))
}

// GetProductListGeneration fetches the current product list generation counter.
func GetProductListGeneration(ctx context.Context, c Cache) (int64, error) {
	return GetGeneration(ctx, c, ProductListGenerationKey())
}

// BumpProductListGeneration atomically increments the product list generation counter.
func BumpProductListGeneration(ctx context.Context, c Cache) (int64, error) {
	return BumpGeneration(ctx, c, ProductListGenerationKey())
}

// SystemConfigGenerationKey returns the generation key for a given system config name.
func SystemConfigGenerationKey(name string) string {
	return fmt.Sprintf("%ssystem:config:%s:gen", KeyPrefix, name)
}

// SystemConfigKey returns the generation-scoped cache key for a given system config.
func SystemConfigKey(name string, generation int64) string {
	return fmt.Sprintf("%ssystem:config:%s:g%d", KeyPrefix, name, generation)
}

// GetSystemConfigGeneration fetches the generation counter for the specified system config.
func GetSystemConfigGeneration(ctx context.Context, c Cache, name string) (int64, error) {
	return GetGeneration(ctx, c, SystemConfigGenerationKey(name))
}

// BumpSystemConfigGeneration atomically increments the generation counter for the specified system config.
func BumpSystemConfigGeneration(ctx context.Context, c Cache, name string) (int64, error) {
	return BumpGeneration(ctx, c, SystemConfigGenerationKey(name))
}

package cache

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"time"
)

// KeyPrefix is the uniform prefix for all cache keys in this application.
const KeyPrefix = "mall:"

// Default TTLs for the first batch of cached entities.
const (
	// TTLProductDetail is the TTL for product detail cache (120 seconds).
	TTLProductDetail = 120 * time.Second

	// TTLProductList is the TTL for product list query cache (60 seconds).
	TTLProductList = 60 * time.Second

	// TTLCategoryTree is the TTL for product category tree/options cache (10 minutes).
	TTLCategoryTree = 10 * time.Minute

	// TTLRegionTree is the TTL for administrative region tree cache (24 hours).
	TTLRegionTree = 24 * time.Hour

	// TTLSystemConfig is the TTL for system/shop public config cache (10 minutes).
	TTLSystemConfig = 10 * time.Minute
)

// ProductDetailKey returns the cache key for admin product detail.
func ProductDetailKey(id int64) string {
	return fmt.Sprintf("%sproduct:detail:%d", KeyPrefix, id)
}

// MiniappProductDetailKey returns the cache key for miniapp product detail.
func MiniappProductDetailKey(id int64) string {
	return fmt.Sprintf("%sminiapp:product:detail:%d", KeyPrefix, id)
}

// ProductListKey returns a hashed cache key for admin product list query parameters.
func ProductListKey(paramStr string) string {
	h := md5.Sum([]byte(paramStr))
	return fmt.Sprintf("%sproduct:list:%s", KeyPrefix, hex.EncodeToString(h[:]))
}

// MiniappProductListKey returns a hashed cache key for miniapp product list query parameters.
func MiniappProductListKey(paramStr string) string {
	h := md5.Sum([]byte(paramStr))
	return fmt.Sprintf("%sminiapp:product:list:%s", KeyPrefix, hex.EncodeToString(h[:]))
}

// CategoryTreeKey returns the cache key for category tree list.
func CategoryTreeKey() string {
	return fmt.Sprintf("%scategory:tree", KeyPrefix)
}

// CategoryOptionsKey returns the cache key for category dropdown options.
func CategoryOptionsKey() string {
	return fmt.Sprintf("%scategory:options", KeyPrefix)
}

// RegionTreeKey returns the cache key for region hierarchy tree data.
func RegionTreeKey() string {
	return fmt.Sprintf("%sregion:tree", KeyPrefix)
}

// SystemConfigKey returns the cache key for a given system config key.
func SystemConfigKey(key string) string {
	return fmt.Sprintf("%ssystem:config:%s", KeyPrefix, key)
}


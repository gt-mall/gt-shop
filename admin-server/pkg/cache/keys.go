package cache

import (
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

// TTLWithJitter adds +/- 10% jitter to prevent thundering herd / simultaneous expiration.
func TTLWithJitter(base time.Duration) time.Duration {
	if base <= 0 {
		return base
	}
	// Jitter percentage between -10% and +10% using nano timestamp
	deltaPercent := (float64(time.Now().UnixNano()%21) - 10.0) / 100.0
	jitter := time.Duration(float64(base) * deltaPercent)
	return base + jitter
}

// MiniappProductListVersionKey returns the version key for miniapp product list.
func MiniappProductListVersionKey() string {
	return fmt.Sprintf("%sversion:miniapp:product:list", KeyPrefix)
}

// MiniappProductDetailVersionKey returns the version key for a specific product detail.
func MiniappProductDetailVersionKey(id int64) string {
	return fmt.Sprintf("%sversion:miniapp:product:detail:%d", KeyPrefix, id)
}

// MiniappProductListKey returns a versioned cache key for miniapp product list query parameters.
func MiniappProductListKey(version int64, paramStr string) string {
	h := md5.Sum([]byte(paramStr))
	return fmt.Sprintf("%sminiapp:product:list:v%d:%s", KeyPrefix, version, hex.EncodeToString(h[:]))
}

// MiniappProductDetailKey returns the versioned cache key for miniapp product detail.
func MiniappProductDetailKey(id int64, version int64) string {
	return fmt.Sprintf("%sminiapp:product:detail:%d:v%d", KeyPrefix, id, version)
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


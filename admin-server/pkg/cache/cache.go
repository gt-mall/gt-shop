package cache

import (
	"context"
	"time"
)

// Cache defines the unified interface for caching operations across the application.
type Cache interface {
	// Get retrieves the value for the given key and unmarshals it into dest.
	// Returns (true, nil) if found and unmarshaled successfully.
	// Returns (false, nil) on a cache miss.
	// Returns (false, err) if an unexpected error occurs during fetch or unmarshal.
	Get(ctx context.Context, key string, dest any) (bool, error)

	// Set marshals value into JSON and stores it with the given TTL.
	Set(ctx context.Context, key string, value any, ttl time.Duration) error

	// Delete removes one or more keys from the cache.
	Delete(ctx context.Context, keys ...string) error

	// BumpVersion atomically increments the version counter for a version key and returns the new version.
	BumpVersion(ctx context.Context, key string) (int64, error)

	// GetVersion retrieves the current version counter for a version key (0 if key does not exist).
	GetVersion(ctx context.Context, key string) (int64, error)

	// Ping checks the health of the cache connection.
	Ping(ctx context.Context) error

	// Close releases any resources associated with the cache.
	Close() error
}


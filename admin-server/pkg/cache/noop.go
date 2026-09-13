package cache

import (
	"context"
	"time"
)

// NoopCache implements Cache by doing nothing (always cache miss on Get, no-op on Set/Delete).
type NoopCache struct{}

// NewNoop creates a new NoopCache instance.
func NewNoop() *NoopCache {
	return &NoopCache{}
}

func (n *NoopCache) Get(ctx context.Context, key string, dest any) (bool, error) {
	return false, nil
}

func (n *NoopCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	return nil
}

func (n *NoopCache) Delete(ctx context.Context, keys ...string) error {
	return nil
}

func (n *NoopCache) Ping(ctx context.Context) error {
	return nil
}

func (n *NoopCache) Close() error {
	return nil
}


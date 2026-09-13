package cache

import (
	"context"

	"github.com/redis/rueidis"
	"github.com/redis/rueidis/rueidisaside"
)

// NoopCache implements Cache by doing nothing (cache miss fallback, no-op mutations).
type NoopCache struct{}

// NewNoop creates a new NoopCache instance.
func NewNoop() *NoopCache {
	return &NoopCache{}
}

func (n *NoopCache) Delete(ctx context.Context, keys ...string) error {
	return nil
}

func (n *NoopCache) Client() rueidis.Client {
	return nil
}

func (n *NoopCache) Aside() rueidisaside.CacheAsideClient {
	return nil
}

func (n *NoopCache) Ping(ctx context.Context) error {
	return nil
}

func (n *NoopCache) Close() error {
	return nil
}

func (n *NoopCache) IsDisabled() bool {
	return true
}

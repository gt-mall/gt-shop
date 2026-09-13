package cache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/rueidis"
	"github.com/redis/rueidis/rueidisaside"
)

// Config holds settings for creating a RueidisCache instance.
type Config struct {
	Addr         string
	Password     string
	DB           int
	DisableCache bool // Set true in tests with miniredis which does not support RESP3 client-side tracking
	UseLuaLock   bool // Set true if Redis < 7 or miniredis is used
}

// RueidisCache implements Cache backed by rueidis and rueidisaside.
type RueidisCache struct {
	client rueidis.Client
	aside  rueidisaside.CacheAsideClient
}

// NewRueidisCache initializes a new RueidisCache instance.
func NewRueidisCache(cfg Config) (*RueidisCache, error) {
	if cfg.Addr == "" {
		cfg.Addr = "127.0.0.1:6379"
	}

	clientOpt := rueidis.ClientOption{
		InitAddress:  []string{cfg.Addr},
		Password:     cfg.Password,
		SelectDB:     cfg.DB,
		DisableCache: cfg.DisableCache,
	}

	asideOpt := rueidisaside.ClientOption{
		ClientOption: clientOpt,
		UseLuaLock:   cfg.UseLuaLock,
	}

	asideClient, err := rueidisaside.NewClient(asideOpt)
	if err != nil {
		return nil, err
	}

	// Verify connectivity via Ping
	pingCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := asideClient.Client().Do(pingCtx, asideClient.Client().B().Ping().Build()).Error(); err != nil {
		asideClient.Close()
		return nil, err
	}

	return &RueidisCache{
		client: asideClient.Client(),
		aside:  asideClient,
	}, nil
}

// Client returns the underlying rueidis client.
func (r *RueidisCache) Client() rueidis.Client {
	if r == nil {
		return nil
	}
	return r.client
}

// Aside returns the underlying rueidisaside client.
func (r *RueidisCache) Aside() rueidisaside.CacheAsideClient {
	if r == nil {
		return nil
	}
	return r.aside
}

// Delete removes one or more keys from the cache.
func (r *RueidisCache) Delete(ctx context.Context, keys ...string) error {
	if r == nil || r.client == nil || len(keys) == 0 {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	validKeys := make([]string, 0, len(keys))
	for _, k := range keys {
		if k != "" {
			validKeys = append(validKeys, k)
		}
	}
	if len(validKeys) == 0 {
		return nil
	}
	if len(validKeys) == 1 {
		return r.client.Do(ctx, r.client.B().Del().Key(validKeys[0]).Build()).Error()
	}

	cmds := make([]rueidis.Completed, len(validKeys))
	for i, k := range validKeys {
		cmds[i] = r.client.B().Del().Key(k).Build()
	}
	for _, resp := range r.client.DoMulti(ctx, cmds...) {
		if err := resp.Error(); err != nil && !rueidis.IsRedisNil(err) {
			return err
		}
	}
	return nil
}

// Ping checks the health of the Redis connection.
func (r *RueidisCache) Ping(ctx context.Context) error {
	if r == nil || r.client == nil {
		return errors.New("redis client not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return r.client.Do(ctx, r.client.B().Ping().Build()).Error()
}

// Close releases resources associated with the cache.
func (r *RueidisCache) Close() error {
	if r == nil || r.aside == nil {
		return nil
	}
	r.aside.Close()
	return nil
}

// IsDisabled reports whether the cache is disabled.
func (r *RueidisCache) IsDisabled() bool {
	return r == nil || r.aside == nil
}

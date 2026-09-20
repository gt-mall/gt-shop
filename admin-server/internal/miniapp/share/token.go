package share

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sync"
	"time"

	"mall/admin_core/pkg/cache"
)

const tokenCacheKeyPrefix = "wx:access_token:v2:"

type cachedToken struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	RenewAt   time.Time `json:"renew_at"`
}

type memoryToken struct {
	key   string
	value cachedToken
}

// TokenManager serializes refresh/invalidation, including the Redis fallback.
// Credentials only occur in a SHA-256 fingerprint; neither cache keys nor errors contain them.
type TokenManager struct {
	client   WeChatShareClient
	cache    cache.Cache
	mu       sync.Mutex
	memCache map[string]memoryToken
}

func NewTokenManager(client WeChatShareClient, c cache.Cache) *TokenManager {
	return &TokenManager{client: client, cache: c, memCache: make(map[string]memoryToken)}
}

func (m *TokenManager) GetToken(ctx context.Context, appID, appSecret string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%s%x", tokenCacheKeyPrefix, sha256.Sum256([]byte(appID+"\x00"+appSecret)))
	previous, ok := m.memCache[appID]
	if ok && previous.key != key {
		if m.cache != nil {
			_ = m.cache.Delete(ctx, previous.key)
		}
		delete(m.memCache, appID)
	}
	for attempt := 0; attempt < 2; attempt++ {
		record, err := cache.Get[cachedToken](ctx, m.cache, key, time.Hour, func(inner context.Context) (cachedToken, error) {
			now := time.Now()
			if item, ok := m.memCache[appID]; ok && item.key == key && now.Before(item.value.RenewAt) {
				cache.SetTTL(inner, time.Until(item.value.RenewAt))
				return item.value, nil
			}
			token, seconds, err := m.client.GetAccessToken(inner, appID, appSecret)
			if err != nil {
				return cachedToken{}, err
			}
			if token == "" || seconds <= 0 {
				return cachedToken{}, errors.New("微信凭证有效期异常")
			}
			// Count the upstream request time against the lifetime; short-lived credentials
			// still receive an early renewal margin instead of a fixed 6900-second TTL.
			lifetime := time.Duration(seconds) * time.Second
			margin := min(5*time.Minute, lifetime/10)
			value := cachedToken{Token: token, ExpiresAt: now.Add(lifetime), RenewAt: now.Add(lifetime - margin)}
			if !time.Now().Before(value.RenewAt) {
				return cachedToken{}, errors.New("微信凭证已临近失效，请重试")
			}
			cache.SetTTL(inner, time.Until(value.RenewAt))
			m.memCache[appID] = memoryToken{key: key, value: value}
			return value, nil
		})
		if err != nil {
			return "", err
		}
		if record.Token != "" && time.Now().Before(record.RenewAt) && time.Now().Before(record.ExpiresAt) {
			m.memCache[appID] = memoryToken{key: key, value: record}
			return record.Token, nil
		}
		if m.cache != nil {
			_ = m.cache.Delete(ctx, key)
		}
		delete(m.memCache, appID)
	}
	return "", errors.New("微信凭证缓存已失效，请稍后重试")
}

func (m *TokenManager) InvalidateToken(ctx context.Context, appID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if item, ok := m.memCache[appID]; ok && m.cache != nil {
		_ = m.cache.Delete(ctx, item.key)
	}
	delete(m.memCache, appID)
}

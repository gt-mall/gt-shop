package share

import (
	"context"
	"fmt"
	"sync"
	"time"

	"mall/admin_core/pkg/cache"
)

const (
	tokenCacheKeyPrefix = "wx:access_token:"
	earlyRenewalBuffer  = 300 * time.Second // 提前 5 分钟刷新
)

type memoryToken struct {
	token     string
	expiresAt time.Time
}

// TokenManager 管理微信 access_token 的缓存、续期与并发合并
type TokenManager struct {
	client   WeChatShareClient
	cache    cache.Cache
	memMu    sync.RWMutex
	memCache map[string]memoryToken
	fetchMu  sync.Mutex
}

// NewTokenManager 创建微信 access_token 管理器
func NewTokenManager(client WeChatShareClient, c cache.Cache) *TokenManager {
	return &TokenManager{
		client:   client,
		cache:    c,
		memCache: make(map[string]memoryToken),
	}
}

// GetToken 获取或刷新微信 access_token
func (m *TokenManager) GetToken(ctx context.Context, appID, appSecret string) (string, error) {
	cacheKey := tokenCacheKeyPrefix + appID
	defaultTTL := 6900 * time.Second // 提前 5 分钟刷新

	return cache.Get[string](ctx, m.cache, cacheKey, defaultTTL, func(innerCtx context.Context) (string, error) {
		now := time.Now()

		m.memMu.RLock()
		item, ok := m.memCache[appID]
		m.memMu.RUnlock()
		if ok && item.expiresAt.After(now.Add(earlyRenewalBuffer)) {
			return item.token, nil
		}

		m.fetchMu.Lock()
		defer m.fetchMu.Unlock()

		m.memMu.RLock()
		item, ok = m.memCache[appID]
		m.memMu.RUnlock()
		if ok && item.expiresAt.After(now.Add(earlyRenewalBuffer)) {
			return item.token, nil
		}

		token, expiresIn, err := m.client.GetAccessToken(innerCtx, appID, appSecret)
		if err != nil {
			return "", err
		}

		ttl := time.Duration(expiresIn) * time.Second
		expiresAt := now.Add(ttl)

		m.memMu.Lock()
		m.memCache[appID] = memoryToken{
			token:     token,
			expiresAt: expiresAt,
		}
		m.memMu.Unlock()

		return token, nil
	})
}

// InvalidateToken 主动失效某个 appID 的 Token 缓存（例如微信报 40001 无效凭证时）
func (m *TokenManager) InvalidateToken(ctx context.Context, appID string) {
	m.memMu.Lock()
	delete(m.memCache, appID)
	m.memMu.Unlock()

	if m.cache != nil {
		cacheKey := fmt.Sprintf("%s%s", tokenCacheKeyPrefix, appID)
		_ = m.cache.Delete(ctx, cacheKey)
	}
}

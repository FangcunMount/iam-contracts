package wechatapp

import (
	"context"
	"errors"
	"time"
)

type accessTokenCacher struct {
	cache    AccessTokenCache
	provider AppTokenProvider
	// 策略
	refreshSkew time.Duration // 提前刷新窗口，e.g., 120s
	cacheTTLMin time.Duration // 最小缓存TTL保护，避免抖动
}

// 确保 accessTokenCacher 实现了相应的接口
var _ AccessTokenCacher = (*accessTokenCacher)(nil)

// NewAccessTokenCacher 创建访问令牌缓存器实例
func NewAccessTokenCacher(cache AccessTokenCache, provider AppTokenProvider) AccessTokenCacher {
	return &accessTokenCacher{
		cache:       cache,
		provider:    provider,
		refreshSkew: 120 * time.Second,
		cacheTTLMin: 60 * time.Second,
	}
}

// EnsureToken 单飞刷新 + 过期缓冲 获取访问令牌
func (s *accessTokenCacher) EnsureToken(ctx context.Context, app *WechatApp, skew time.Duration) (string, error) {
	if app == nil {
		return "", errors.New("nil app")
	}
	if skew <= 0 {
		skew = s.refreshSkew
	}
	// 1) 读缓存
	if cached, _ := s.cache.Get(ctx, app.AppID); cached != nil && cached.IsValid(time.Now(), skew) {
		return cached.Token, nil
	}

	// 2) 单飞刷新
	ok, unlock, err := s.cache.TryLockRefresh(ctx, app.AppID, 10*time.Second)
	if err != nil {
		return "", err
	}
	if ok {
		defer unlock()
		aat, err := s.provider.Fetch(ctx, app)
		if err != nil {
			return "", err
		}
		ttl := time.Until(aat.ExpiresAt) - skew
		if ttl < s.cacheTTLMin {
			ttl = s.cacheTTLMin
		}
		if err := s.cache.Set(ctx, app.AppID, aat, ttl); err != nil {
			return "", err
		}
		return aat.Token, nil
	}

	// 3) 未拿到锁：读一次缓存（可能被别人刷新了）
	if cached, _ := s.cache.Get(ctx, app.AppID); cached != nil && cached.Token != "" {
		return cached.Token, nil
	}
	return "", errors.New("access_token refresh in progress, please retry")
}

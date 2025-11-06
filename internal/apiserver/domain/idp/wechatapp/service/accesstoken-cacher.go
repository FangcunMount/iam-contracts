package service

import (
	"context"
	"errors"
	"time"

	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/idp/wechatapp"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/idp/wechatapp/port"
)

type AccessTokenCacher struct {
	cache    port.AccessTokenCache
	provider port.AppTokenProvider
	// 策略
	refreshSkew time.Duration // 提前刷新窗口，e.g., 120s
	cacheTTLMin time.Duration // 最小缓存TTL保护，避免抖动
}

// 确保 AccessTokenCacher 实现了相应的接口
var _ port.AccessTokenCacher = (*AccessTokenCacher)(nil)

// NewAccessTokenCacher 创建访问令牌缓存器实例
func NewAccessTokenCacher() *AccessTokenCacher {
	return &AccessTokenCacher{}
}

// EnsureToken 单飞刷新 + 过期缓冲 获取访问令牌
// ctx: 上下文
// app: 微信应用实体
// skew: 过期缓冲时间窗口，<=0 则使用默认值
func (s *AccessTokenCacher) EnsureToken(ctx context.Context, app *domain.WechatApp, skew time.Duration) (string, error) {
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

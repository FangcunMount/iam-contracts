package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	redislease "github.com/FangcunMount/component-base/pkg/redis/lease"
	redisstore "github.com/FangcunMount/component-base/pkg/redis/store"
	"github.com/redis/go-redis/v9"

	"github.com/FangcunMount/component-base/pkg/log"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/idp/wechatapp"
)

// accessTokenCache 访问令牌缓存实现
type accessTokenCache struct {
	tokens *redisstore.ValueStore[wechatapp.AppAccessToken]
	leases *redislease.Service
}

// 确保实现了接口
var _ wechatapp.AccessTokenCache = (*accessTokenCache)(nil)

// NewAccessTokenCache 创建访问令牌缓存实例
func NewAccessTokenCache(client *redis.Client) wechatapp.AccessTokenCache {
	return &accessTokenCache{
		tokens: newJSONStore[wechatapp.AppAccessToken](client),
		leases: newLeaseService(client),
	}
}

// Get 获取访问令牌
func (c *accessTokenCache) Get(ctx context.Context, appID string) (*wechatapp.AppAccessToken, error) {
	if appID == "" {
		return nil, errors.New("appID cannot be empty")
	}

	key := wechatAccessTokenRedisKey(appID)
	storeKey, err := newStoreKey(key)
	if err != nil {
		return nil, err
	}

	aat, found, err := c.tokens.Get(ctx, storeKey)
	if err != nil {
		redisError(ctx, "failed to load access token", log.String("error", err.Error()), log.String("key", key))
		return nil, fmt.Errorf("failed to get access token from cache: %w", err)
	}
	if !found {
		return nil, nil
	}

	return &aat, nil
}

// Set 设置访问令牌
func (c *accessTokenCache) Set(ctx context.Context, appID string, aat *wechatapp.AppAccessToken, ttl time.Duration) error {
	if appID == "" {
		return errors.New("appID cannot be empty")
	}
	if aat == nil {
		return errors.New("access token cannot be nil")
	}

	key := wechatAccessTokenRedisKey(appID)
	storeKey, err := newStoreKey(key)
	if err != nil {
		return err
	}

	if err := c.tokens.Set(ctx, storeKey, *aat, ttl); err != nil {
		return fmt.Errorf("failed to set access token to cache: %w", err)
	}

	redisInfo(ctx, "access token cached",
		log.String("app_id", appID),
		log.Duration("ttl", ttl),
	)
	return nil
}

// TryLockRefresh 尝试获取单飞刷新锁
func (c *accessTokenCache) TryLockRefresh(ctx context.Context, appID string, ttl time.Duration) (ok bool, unlock func(), err error) {
	if appID == "" {
		return false, nil, errors.New("appID cannot be empty")
	}

	lockKey := wechatAccessTokenLockRedisKey(appID)
	leaseKey, err := newLeaseKey(lockKey)
	if err != nil {
		return false, nil, err
	}
	attempt, err := c.leases.Acquire(ctx, leaseKey, ttl, nil)
	if err != nil {
		return false, nil, fmt.Errorf("failed to acquire lock: %w", err)
	}

	if !attempt.Acquired {
		return false, nil, nil
	}

	heldLease := attempt.Lease
	unlock = func() {
		_ = c.leases.Release(context.Background(), heldLease)
	}

	redisInfo(ctx, "access token lock acquired", log.String("lock_key", lockKey), log.Duration("ttl", ttl))
	return true, unlock, nil
}

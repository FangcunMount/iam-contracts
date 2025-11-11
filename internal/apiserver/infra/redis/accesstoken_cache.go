package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/FangcunMount/component-base/pkg/log"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/idp/wechatapp"
)

// accessTokenCache 访问令牌缓存实现
type accessTokenCache struct {
	client *redis.Client
	prefix string // 键前缀
}

// 确保实现了接口
var _ wechatapp.AccessTokenCache = (*accessTokenCache)(nil)

// NewAccessTokenCache 创建访问令牌缓存实例
func NewAccessTokenCache(client *redis.Client) wechatapp.AccessTokenCache {
	return &accessTokenCache{
		client: client,
		prefix: "idp:wechat:token:",
	}
}

// Get 获取访问令牌
func (c *accessTokenCache) Get(ctx context.Context, appID string) (*wechatapp.AppAccessToken, error) {
	if appID == "" {
		return nil, errors.New("appID cannot be empty")
	}

	key := c.tokenKey(appID)
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			redisDebug(ctx, "access token cache miss", log.String("key", key))
			return nil, nil // 缓存未命中
		}
		redisError(ctx, "failed to get access token", log.String("error", err.Error()), log.String("key", key))
		return nil, fmt.Errorf("failed to get access token from cache: %w", err)
	}

	var aat wechatapp.AppAccessToken
	if err := json.Unmarshal(data, &aat); err != nil {
		redisError(ctx, "failed to unmarshal access token", log.String("error", err.Error()), log.String("key", key))
		return nil, fmt.Errorf("failed to unmarshal access token: %w", err)
	}

	redisDebug(ctx, "access token cache hit", log.String("app_id", appID))
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

	key := c.tokenKey(appID)
	data, err := json.Marshal(aat)
	if err != nil {
		redisError(ctx, "failed to marshal access token", log.String("error", err.Error()), log.String("app_id", appID))
		return fmt.Errorf("failed to marshal access token: %w", err)
	}

	if err := c.client.Set(ctx, key, data, ttl).Err(); err != nil {
		redisError(ctx, "failed to write access token", log.String("error", err.Error()), log.String("app_id", appID))
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

	lockKey := c.lockKey(appID)

	// 尝试获取分布式锁（使用 SET NX EX）
	ok, err = c.client.SetNX(ctx, lockKey, "1", ttl).Result()
	if err != nil {
		redisError(ctx, "failed to acquire access token lock", log.String("error", err.Error()), log.String("lock_key", lockKey))
		return false, nil, fmt.Errorf("failed to acquire lock: %w", err)
	}

	if !ok {
		redisDebug(ctx, "access token lock already held", log.String("lock_key", lockKey))
		// 未获取到锁
		return false, nil, nil
	}

	// 获取到锁，返回解锁函数
	unlock = func() {
		c.client.Del(context.Background(), lockKey)
		redisDebug(context.Background(), "access token lock released", log.String("lock_key", lockKey))
	}

	redisInfo(ctx, "access token lock acquired", log.String("lock_key", lockKey), log.Duration("ttl", ttl))
	return true, unlock, nil
}

// tokenKey 生成令牌缓存键
func (c *accessTokenCache) tokenKey(appID string) string {
	return c.prefix + appID
}

// lockKey 生成刷新锁键
func (c *accessTokenCache) lockKey(appID string) string {
	return c.prefix + "lock:" + appID
}

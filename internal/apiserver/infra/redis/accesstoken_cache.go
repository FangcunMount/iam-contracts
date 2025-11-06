package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/idp/wechatapp"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/idp/wechatapp/port"
)

// accessTokenCache 访问令牌缓存实现
type accessTokenCache struct {
	client *redis.Client
	prefix string // 键前缀
}

// 确保实现了接口
var _ port.AccessTokenCache = (*accessTokenCache)(nil)

// NewAccessTokenCache 创建访问令牌缓存实例
func NewAccessTokenCache(client *redis.Client) port.AccessTokenCache {
	return &accessTokenCache{
		client: client,
		prefix: "idp:wechat:token:",
	}
}

// Get 获取访问令牌
func (c *accessTokenCache) Get(ctx context.Context, appID string) (*domain.AppAccessToken, error) {
	if appID == "" {
		return nil, errors.New("appID cannot be empty")
	}

	key := c.tokenKey(appID)
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil // 缓存未命中
		}
		return nil, fmt.Errorf("failed to get access token from cache: %w", err)
	}

	var aat domain.AppAccessToken
	if err := json.Unmarshal(data, &aat); err != nil {
		return nil, fmt.Errorf("failed to unmarshal access token: %w", err)
	}

	return &aat, nil
}

// Set 设置访问令牌
func (c *accessTokenCache) Set(ctx context.Context, appID string, aat *domain.AppAccessToken, ttl time.Duration) error {
	if appID == "" {
		return errors.New("appID cannot be empty")
	}
	if aat == nil {
		return errors.New("access token cannot be nil")
	}

	key := c.tokenKey(appID)
	data, err := json.Marshal(aat)
	if err != nil {
		return fmt.Errorf("failed to marshal access token: %w", err)
	}

	if err := c.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set access token to cache: %w", err)
	}

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
		return false, nil, fmt.Errorf("failed to acquire lock: %w", err)
	}

	if !ok {
		// 未获取到锁
		return false, nil, nil
	}

	// 获取到锁，返回解锁函数
	unlock = func() {
		c.client.Del(context.Background(), lockKey)
	}

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

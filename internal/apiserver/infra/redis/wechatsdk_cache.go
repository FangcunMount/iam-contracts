package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	wechatCache "github.com/silenceper/wechat/v2/cache"

	"github.com/FangcunMount/component-base/pkg/log"
)

// WechatSDKCache 微信 SDK Cache 接口的 Redis 实现
// 适配 github.com/silenceper/wechat/v2/cache.Cache 接口
type WechatSDKCache struct {
	client *redis.Client
	ctx    context.Context
}

// NewWechatSDKCache 创建微信 SDK 缓存实例
func NewWechatSDKCache(client *redis.Client) wechatCache.Cache {
	return &WechatSDKCache{
		client: client,
		ctx:    context.Background(),
	}
}

// Get 从缓存中获取数据
func (c *WechatSDKCache) Get(key string) interface{} {
	val, err := c.client.Get(c.ctx, key).Result()
	if err != nil {
		if err != redis.Nil {
			redisError(nil, "wechat sdk cache get failed", log.String("error", err.Error()), log.String("key", key))
		}
		return nil
	}
	redisDebug(nil, "wechat sdk cache hit", log.String("key", key))
	return val
}

// Set 设置缓存数据，timeout 单位为秒
func (c *WechatSDKCache) Set(key string, val interface{}, timeout time.Duration) error {
	if err := c.client.Set(c.ctx, key, val, timeout).Err(); err != nil {
		redisError(nil, "wechat sdk cache set failed", log.String("error", err.Error()), log.String("key", key))
		return err
	}
	redisInfo(nil, "wechat sdk cache set", log.String("key", key), log.Duration("ttl", timeout))
	return nil
}

// IsExist 判断 key 是否存在
func (c *WechatSDKCache) IsExist(key string) bool {
	result, err := c.client.Exists(c.ctx, key).Result()
	if err != nil {
		redisError(nil, "wechat sdk cache exists failed", log.String("error", err.Error()), log.String("key", key))
		return false
	}
	redisDebug(nil, "wechat sdk cache exists", log.String("key", key), log.Int64("count", result))
	return result > 0
}

// Delete 删除缓存
func (c *WechatSDKCache) Delete(key string) error {
	if err := c.client.Del(c.ctx, key).Err(); err != nil {
		redisError(nil, "wechat sdk cache delete failed", log.String("error", err.Error()), log.String("key", key))
		return err
	}
	redisInfo(nil, "wechat sdk cache deleted", log.String("key", key))
	return nil
}

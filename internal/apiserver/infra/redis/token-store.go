package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/FangcunMount/component-base/pkg/log"
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/token"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

// RedisStore Redis 令牌存储实现
type RedisStore struct {
	client *redis.Client
}

// NewRedisStore 创建 Redis 令牌存储
func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{
		client: client,
	}
}

// refreshTokenData 刷新令牌存储数据结构
type refreshTokenData struct {
	TokenID   string    `json:"token_id"`
	UserID    uint64    `json:"user_id"`
	AccountID uint64    `json:"account_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

// SaveRefreshToken 保存刷新令牌
func (s *RedisStore) SaveRefreshToken(ctx context.Context, token *domain.Token) error {
	if token == nil {
		return fmt.Errorf("token is nil")
	}

	data := refreshTokenData{
		TokenID:   token.ID,
		UserID:    token.UserID.Uint64(),
		AccountID: token.AccountID.Uint64(),
		ExpiresAt: token.ExpiresAt,
	}

	// 序列化
	jsonData, err := json.Marshal(data)
	if err != nil {
		redisError(ctx, "failed to marshal refresh token", log.String("error", err.Error()))
		return fmt.Errorf("failed to marshal refresh token: %w", err)
	}

	// 保存到 Redis，key 格式: refresh_token:{token_value}
	key := fmt.Sprintf("refresh_token:%s", token.Value)
	ttl := token.RemainingDuration()
	if ttl <= 0 {
		redisWarn(ctx, "attempted to save expired refresh token", log.String("token_id", token.ID))
		return fmt.Errorf("token already expired")
	}

	if err := s.client.Set(ctx, key, jsonData, ttl).Err(); err != nil {
		// Redis Hook 已经记录了 SET 命令错误，只返回错误即可
		return fmt.Errorf("failed to save refresh token to redis: %w", err)
	}

	redisInfo(ctx, "refresh token cached",
		log.String("token_id", token.ID),
		log.String("key", key),
		log.Duration("ttl", ttl),
	)
	return nil
}

// GetRefreshToken 获取刷新令牌
func (s *RedisStore) GetRefreshToken(ctx context.Context, tokenValue string) (*domain.Token, error) {
	key := fmt.Sprintf("refresh_token:%s", tokenValue)

	// 从 Redis 获取
	jsonData, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			// Redis Hook 已经记录了 GET 命令，这里不需要再记录 cache miss
			return nil, nil // 令牌不存在
		}
		// Redis Hook 已经记录了命令错误，只返回错误即可
		return nil, fmt.Errorf("failed to get refresh token from redis: %w", err)
	}

	// 反序列化
	var data refreshTokenData
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		redisError(ctx, "failed to unmarshal refresh token", log.String("error", err.Error()))
		return nil, fmt.Errorf("failed to unmarshal refresh token: %w", err)
	}

	// 构造 Token 对象
	ttl := time.Until(data.ExpiresAt)
	userID := meta.FromUint64(data.UserID)
	accountID := meta.FromUint64(data.AccountID)
	token := domain.NewRefreshToken(
		data.TokenID,
		tokenValue,
		userID,
		accountID,
		ttl,
	)

	// Redis Hook 已经记录了 GET 命令成功，这里不需要再记录 cache hit
	return token, nil
}

// DeleteRefreshToken 删除刷新令牌
func (s *RedisStore) DeleteRefreshToken(ctx context.Context, tokenValue string) error {
	key := fmt.Sprintf("refresh_token:%s", tokenValue)

	if err := s.client.Del(ctx, key).Err(); err != nil {
		// Redis Hook 已经记录了 DEL 命令错误，只返回错误即可
		return fmt.Errorf("failed to delete refresh token from redis: %w", err)
	}

	redisInfo(ctx, "refresh token deleted", log.String("key", key))
	return nil
}

// AddToBlacklist 将令牌加入黑名单
func (s *RedisStore) AddToBlacklist(ctx context.Context, tokenID string, expiry time.Duration) error {
	key := fmt.Sprintf("token_blacklist:%s", tokenID)

	// 设置黑名单标记，TTL 为令牌剩余有效期
	if err := s.client.Set(ctx, key, "1", expiry).Err(); err != nil {
		// Redis Hook 已经记录了 SET 命令错误，只返回错误即可
		return fmt.Errorf("failed to add token to blacklist: %w", err)
	}

	redisInfo(ctx, "token blacklisted", log.String("token_id", tokenID), log.Duration("ttl", expiry))
	return nil
}

// IsBlacklisted 检查令牌是否在黑名单中
func (s *RedisStore) IsBlacklisted(ctx context.Context, tokenID string) (bool, error) {
	key := fmt.Sprintf("token_blacklist:%s", tokenID)

	// 检查 key 是否存在
	exists, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		// Redis Hook 已经记录了 EXISTS 命令错误，只返回错误即可
		return false, fmt.Errorf("failed to check token blacklist: %w", err)
	}

	// Redis Hook 已经记录了 EXISTS 命令结果，这里不需要再记录
	return exists > 0, nil
}

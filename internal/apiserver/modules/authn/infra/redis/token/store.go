// Package token Redis 令牌存储实现
package token

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/FangcunMount/component-base/pkg/util/idutil"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/authentication"
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
func (s *RedisStore) SaveRefreshToken(ctx context.Context, token *authentication.Token) error {
	if token == nil {
		return fmt.Errorf("token is nil")
	}

	data := refreshTokenData{
		TokenID:   token.ID,
		UserID:    token.UserID.Uint64(),
		AccountID: idutil.ID(token.AccountID).Uint64(),
		ExpiresAt: token.ExpiresAt,
	}

	// 序列化
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal refresh token: %w", err)
	}

	// 保存到 Redis，key 格式: refresh_token:{token_value}
	key := fmt.Sprintf("refresh_token:%s", token.Value)
	ttl := token.RemainingDuration()
	if ttl <= 0 {
		return fmt.Errorf("token already expired")
	}

	if err := s.client.Set(ctx, key, jsonData, ttl).Err(); err != nil {
		return fmt.Errorf("failed to save refresh token to redis: %w", err)
	}

	return nil
}

// GetRefreshToken 获取刷新令牌
func (s *RedisStore) GetRefreshToken(ctx context.Context, tokenValue string) (*authentication.Token, error) {
	key := fmt.Sprintf("refresh_token:%s", tokenValue)

	// 从 Redis 获取
	jsonData, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // 令牌不存在
		}
		return nil, fmt.Errorf("failed to get refresh token from redis: %w", err)
	}

	// 反序列化
	var data refreshTokenData
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal refresh token: %w", err)
	}

	// 构造 Token 对象
	ttl := time.Until(data.ExpiresAt)
	token := authentication.NewRefreshToken(
		data.TokenID,
		tokenValue,
		account.NewUserID(data.UserID),
		account.AccountID(idutil.NewID(data.AccountID)),
		ttl,
	)

	return token, nil
}

// DeleteRefreshToken 删除刷新令牌
func (s *RedisStore) DeleteRefreshToken(ctx context.Context, tokenValue string) error {
	key := fmt.Sprintf("refresh_token:%s", tokenValue)

	if err := s.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete refresh token from redis: %w", err)
	}

	return nil
}

// AddToBlacklist 将令牌加入黑名单
func (s *RedisStore) AddToBlacklist(ctx context.Context, tokenID string, expiry time.Duration) error {
	key := fmt.Sprintf("token_blacklist:%s", tokenID)

	// 设置黑名单标记，TTL 为令牌剩余有效期
	if err := s.client.Set(ctx, key, "1", expiry).Err(); err != nil {
		return fmt.Errorf("failed to add token to blacklist: %w", err)
	}

	return nil
}

// IsBlacklisted 检查令牌是否在黑名单中
func (s *RedisStore) IsBlacklisted(ctx context.Context, tokenID string) (bool, error) {
	key := fmt.Sprintf("token_blacklist:%s", tokenID)

	// 检查 key 是否存在
	exists, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check token blacklist: %w", err)
	}

	return exists > 0, nil
}

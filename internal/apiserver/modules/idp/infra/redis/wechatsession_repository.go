package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/domain/wechatsession"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/domain/wechatsession/port"
)

// wechatSessionRepository 微信会话仓储实现
type wechatSessionRepository struct {
	client *redis.Client
	prefix string // 键前缀
}

// 确保实现了接口
var _ port.WechatSessionRepository = (*wechatSessionRepository)(nil)

// NewWechatSessionRepository 创建微信会话仓储实例
func NewWechatSessionRepository(client *redis.Client) port.WechatSessionRepository {
	return &wechatSessionRepository{
		client: client,
		prefix: "idp:wechat:session:",
	}
}

// Get 获取微信会话
func (r *wechatSessionRepository) Get(ctx context.Context, appID, openID string) (*domain.WechatSession, error) {
	if appID == "" || openID == "" {
		return nil, errors.New("appID and openID cannot be empty")
	}

	key := r.sessionKey(appID, openID)
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil // 会话不存在
		}
		return nil, fmt.Errorf("failed to get wechat session: %w", err)
	}

	var session domain.WechatSession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal wechat session: %w", err)
	}

	return &session, nil
}

// Set 设置微信会话
func (r *wechatSessionRepository) Set(ctx context.Context, s *domain.WechatSession, ttl time.Duration) error {
	if s == nil {
		return errors.New("session cannot be nil")
	}
	if s.AppID == "" || s.OpenID == "" {
		return errors.New("appID and openID cannot be empty")
	}

	key := r.sessionKey(s.AppID, s.OpenID)
	data, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("failed to marshal wechat session: %w", err)
	}

	if err := r.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set wechat session: %w", err)
	}

	return nil
}

// sessionKey 生成会话缓存键
func (r *wechatSessionRepository) sessionKey(appID, openID string) string {
	return r.prefix + appID + ":" + openID
}

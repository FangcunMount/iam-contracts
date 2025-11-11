package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/redis/go-redis/v9"

	"github.com/FangcunMount/component-base/pkg/log"
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authz/policy"
)

// VersionNotifier Redis 版本通知器实现
type VersionNotifier struct {
	client  *redis.Client
	pubsub  *redis.PubSub
	channel string
	mu      sync.RWMutex
	closed  bool
}

var _ domain.VersionNotifier = (*VersionNotifier)(nil)

// VersionChangeMessage 版本变更消息
type VersionChangeMessage struct {
	TenantID string `json:"tenant_id"`
	Version  int64  `json:"version"`
}

// NewVersionNotifier 创建版本通知器
func NewVersionNotifier(client *redis.Client, channel string) domain.VersionNotifier {
	if channel == "" {
		channel = "authz:policy_changed"
	}

	return &VersionNotifier{
		client:  client,
		channel: channel,
		closed:  false,
	}
}

// Publish 发布策略版本变更通知
func (n *VersionNotifier) Publish(ctx context.Context, tenantID string, version int64) error {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.closed {
		return fmt.Errorf("notifier is closed")
	}

	msg := VersionChangeMessage{
		TenantID: tenantID,
		Version:  version,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		redisError(ctx, "failed to marshal version notifier payload", log.String("error", err.Error()))
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	err = n.client.Publish(ctx, n.channel, data).Err()
	if err != nil {
		redisError(ctx, "failed to publish version change",
			log.String("error", err.Error()),
			log.String("tenant_id", tenantID),
			log.Int64("version", version),
		)
		return fmt.Errorf("failed to publish message: %w", err)
	}

	redisInfo(ctx, "version change published",
		log.String("tenant_id", tenantID),
		log.Int64("version", version),
		log.String("channel", n.channel),
	)
	return nil
}

// Subscribe 订阅策略版本变更通知
func (n *VersionNotifier) Subscribe(ctx context.Context, handler domain.VersionChangeHandler) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.closed {
		return fmt.Errorf("notifier is closed")
	}

	if n.pubsub != nil {
		return fmt.Errorf("already subscribed")
	}

	// 订阅频道
	n.pubsub = n.client.Subscribe(ctx, n.channel)

	// 等待订阅确认
	_, err := n.pubsub.Receive(ctx)
	if err != nil {
		_ = n.pubsub.Close()
		n.pubsub = nil
		redisError(ctx, "failed to subscribe version channel", log.String("error", err.Error()), log.String("channel", n.channel))
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	// 启动消息处理协程
	go n.handleMessages(handler)

	redisInfo(ctx, "subscribed to version channel", log.String("channel", n.channel))
	return nil
}

// handleMessages 处理接收到的消息
func (n *VersionNotifier) handleMessages(handler domain.VersionChangeHandler) {
	ch := n.pubsub.Channel()

	for msg := range ch {
		var changeMsg VersionChangeMessage

		if err := json.Unmarshal([]byte(msg.Payload), &changeMsg); err != nil {
			redisWarn(context.Background(), "failed to unmarshal version message",
				log.String("error", err.Error()),
				log.String("channel", n.channel),
			)
			continue
		}

		redisDebug(context.Background(), "version change received",
			log.String("tenant_id", changeMsg.TenantID),
			log.Int64("version", changeMsg.Version),
		)
		// 调用处理函数
		handler(changeMsg.TenantID, changeMsg.Version)
	}
}

// Close 关闭订阅
func (n *VersionNotifier) Close() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.closed {
		return nil
	}

	n.closed = true

	if n.pubsub != nil {
		err := n.pubsub.Close()
		n.pubsub = nil
		redisInfo(context.Background(), "version notifier closed", log.String("channel", n.channel))
		return err
	}

	return nil
}

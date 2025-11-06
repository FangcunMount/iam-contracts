package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	drivenPort "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authz/policy/port/driven"
	"github.com/redis/go-redis/v9"
)

// VersionNotifier Redis 版本通知器实现
type VersionNotifier struct {
	client  *redis.Client
	pubsub  *redis.PubSub
	channel string
	mu      sync.RWMutex
	closed  bool
}

var _ drivenPort.VersionNotifier = (*VersionNotifier)(nil)

// VersionChangeMessage 版本变更消息
type VersionChangeMessage struct {
	TenantID string `json:"tenant_id"`
	Version  int64  `json:"version"`
}

// NewVersionNotifier 创建版本通知器
func NewVersionNotifier(client *redis.Client, channel string) drivenPort.VersionNotifier {
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
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	err = n.client.Publish(ctx, n.channel, data).Err()
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

// Subscribe 订阅策略版本变更通知
func (n *VersionNotifier) Subscribe(ctx context.Context, handler drivenPort.VersionChangeHandler) error {
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
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	// 启动消息处理协程
	go n.handleMessages(handler)

	return nil
}

// handleMessages 处理接收到的消息
func (n *VersionNotifier) handleMessages(handler drivenPort.VersionChangeHandler) {
	ch := n.pubsub.Channel()

	for msg := range ch {
		var changeMsg VersionChangeMessage

		if err := json.Unmarshal([]byte(msg.Payload), &changeMsg); err != nil {
			// 记录日志但继续处理
			fmt.Printf("failed to unmarshal message: %v\n", err)
			continue
		}

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
		return err
	}

	return nil
}

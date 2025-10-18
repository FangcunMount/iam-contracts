// Package driven 策略版本通知端口定义
package driven

import "context"

// VersionNotifier 策略版本通知接口
type VersionNotifier interface {
	// Publish 发布策略版本变更通知
	Publish(ctx context.Context, tenantID string, version int64) error

	// Subscribe 订阅策略版本变更通知
	// handler 会在接收到版本变更通知时被调用
	Subscribe(ctx context.Context, handler VersionChangeHandler) error

	// Close 关闭订阅
	Close() error
}

// VersionChangeHandler 版本变更处理函数
type VersionChangeHandler func(tenantID string, version int64)

// Package policy 策略领域包
package policy

import (
	"context"
)

// CasbinAdapter Casbin 策略操作接口（Driven Port - 外部服务）
type CasbinAdapter interface {
	// AddPolicy 添加 p 规则
	AddPolicy(ctx context.Context, rules ...PolicyRule) error
	// RemovePolicy 删除 p 规则
	RemovePolicy(ctx context.Context, rules ...PolicyRule) error
	// AddGroupingPolicy 添加 g 规则
	AddGroupingPolicy(ctx context.Context, rules ...GroupingRule) error
	// RemoveGroupingPolicy 删除 g 规则
	RemoveGroupingPolicy(ctx context.Context, rules ...GroupingRule) error
	// GetPoliciesByRole 获取角色的所有 p 规则
	GetPoliciesByRole(ctx context.Context, role, domain string) ([]PolicyRule, error)
	// GetGroupingsBySubject 获取主体的所有 g 规则
	GetGroupingsBySubject(ctx context.Context, subject, domain string) ([]GroupingRule, error)
	// LoadPolicy 重新加载策略（用于缓存刷新）
	LoadPolicy(ctx context.Context) error
}

// VersionNotifier 策略版本通知接口（Driven Port - 外部服务）
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

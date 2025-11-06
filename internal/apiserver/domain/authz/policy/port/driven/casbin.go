// Package driven 策略 Casbin 操作端口定义
package driven

import (
	"context"

	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authz/policy"
)

// CasbinPort Casbin 策略操作接口（抽象适配器）
type CasbinPort interface {
	// AddPolicy 添加 p 规则
	AddPolicy(ctx context.Context, rules ...domain.PolicyRule) error
	// RemovePolicy 删除 p 规则
	RemovePolicy(ctx context.Context, rules ...domain.PolicyRule) error
	// AddGroupingPolicy 添加 g 规则
	AddGroupingPolicy(ctx context.Context, rules ...domain.GroupingRule) error
	// RemoveGroupingPolicy 删除 g 规则
	RemoveGroupingPolicy(ctx context.Context, rules ...domain.GroupingRule) error
	// GetPoliciesByRole 获取角色的所有 p 规则
	GetPoliciesByRole(ctx context.Context, role, domain string) ([]domain.PolicyRule, error)
	// GetGroupingsBySubject 获取主体的所有 g 规则
	GetGroupingsBySubject(ctx context.Context, subject, domain string) ([]domain.GroupingRule, error)
	// LoadPolicy 重新加载策略（用于缓存刷新）
	LoadPolicy(ctx context.Context) error
}

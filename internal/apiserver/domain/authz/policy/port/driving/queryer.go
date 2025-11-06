// Package driving 定义策略模块的 Driving 端口（用例接口）
package driving

import (
	"context"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authz/policy"
)

// PolicyQueryer 策略查询接口（读操作）
// 定义策略查询的用例接口，遵循 CQRS 原则
type PolicyQueryer interface {
	// GetPoliciesByRole 获取角色的所有策略规则
	GetPoliciesByRole(ctx context.Context, query GetPoliciesByRoleQuery) ([]policy.PolicyRule, error)

	// GetCurrentVersion 获取当前策略版本
	GetCurrentVersion(ctx context.Context, query GetCurrentVersionQuery) (*policy.PolicyVersion, error)
}

// GetPoliciesByRoleQuery 获取角色策略查询
type GetPoliciesByRoleQuery struct {
	RoleID   uint64 // 角色ID
	TenantID string // 租户ID
}

// GetCurrentVersionQuery 获取当前版本查询
type GetCurrentVersionQuery struct {
	TenantID string // 租户ID
}

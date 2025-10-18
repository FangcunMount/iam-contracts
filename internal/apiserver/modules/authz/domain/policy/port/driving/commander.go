// Package driving 定义策略模块的 Driving 端口（用例接口）
package driving

import (
	"context"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/policy"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/resource"
)

// PolicyCommander 策略命令接口（写操作）
// 定义策略管理的用例接口，遵循 CQRS 原则
type PolicyCommander interface {
	// AddPolicyRule 添加策略规则
	AddPolicyRule(ctx context.Context, cmd AddPolicyRuleCommand) error

	// RemovePolicyRule 移除策略规则
	RemovePolicyRule(ctx context.Context, cmd RemovePolicyRuleCommand) error
}

// AddPolicyRuleCommand 添加策略规则命令
type AddPolicyRuleCommand struct {
	RoleID     uint64              // 角色ID
	ResourceID resource.ResourceID // 资源ID
	Action     string              // 操作
	TenantID   string              // 租户ID
	ChangedBy  string              // 变更人
	Reason     string              // 变更原因
}

// RemovePolicyRuleCommand 移除策略规则命令
type RemovePolicyRuleCommand struct {
	RoleID     uint64              // 角色ID
	ResourceID resource.ResourceID // 资源ID
	Action     string              // 操作
	TenantID   string              // 租户ID
	ChangedBy  string              // 变更人
	Reason     string              // 变更原因
}

// BuildPolicyRule 构建策略规则（辅助方法）
func BuildPolicyRule(roleKey, tenantID, resourceKey, action string) policy.PolicyRule {
	return policy.PolicyRule{
		Sub: roleKey,
		Dom: tenantID,
		Obj: resourceKey,
		Act: action,
	}
}

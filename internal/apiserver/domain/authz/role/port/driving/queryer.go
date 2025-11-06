// Package driving 角色领域驱动端口定义
package driving

import (
	"context"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authz/role"
)

// RoleQueryer 角色查询服务接口（读操作）
//
// 职责：
// - 处理角色的查询操作
// - 提供不同维度的角色检索
// - 支持租户隔离的查询
//
// 实现者：application/role/RoleQueryService
type RoleQueryer interface {
	// GetRoleByID 根据ID获取角色
	GetRoleByID(ctx context.Context, roleID role.RoleID) (*role.Role, error)

	// GetRoleByName 根据名称获取角色（租户内）
	GetRoleByName(ctx context.Context, tenantID, name string) (*role.Role, error)

	// ListRoles 列出角色（支持分页和租户过滤）
	ListRoles(ctx context.Context, query ListRolesQuery) (*ListRolesResult, error)

	// ListRolesByTenant 列出指定租户的所有角色
	ListRolesByTenant(ctx context.Context, tenantID string) ([]*role.Role, error)
}

// ListRolesQuery 列出角色查询参数
type ListRolesQuery struct {
	// TenantID 租户ID过滤（可选）
	TenantID string

	// Offset 分页偏移量
	Offset int

	// Limit 分页限制（每页数量）
	Limit int
}

// ListRolesResult 列出角色结果
type ListRolesResult struct {
	// Roles 角色列表
	Roles []*role.Role

	// Total 总数量
	Total int64
}

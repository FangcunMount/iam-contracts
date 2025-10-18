// Package driving 角色领域驱动端口定义
//
// 本包定义了角色管理的用例接口（Driving Ports），遵循CQRS原则：
// - RoleCommander: 处理写操作（命令端）
// - RoleQueryer: 处理读操作（查询端）
//
// 这些接口由应用层实现，被接口层调用。
package driving

import (
	"context"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/role"
)

// RoleCommander 角色命令服务接口（写操作）
//
// 职责：
// - 处理角色的创建、更新、删除操作
// - 管理事务边界
// - 协调领域服务和仓储
//
// 实现者：application/role/RoleCommandService
type RoleCommander interface {
	// CreateRole 创建角色
	CreateRole(ctx context.Context, cmd CreateRoleCommand) (*role.Role, error)

	// UpdateRole 更新角色
	UpdateRole(ctx context.Context, cmd UpdateRoleCommand) (*role.Role, error)

	// DeleteRole 删除角色
	DeleteRole(ctx context.Context, roleID role.RoleID) error
}

// CreateRoleCommand 创建角色命令
type CreateRoleCommand struct {
	// Name 角色名称，在租户内唯一
	Name string

	// DisplayName 显示名称
	DisplayName string

	// TenantID 租户ID
	TenantID string

	// Description 角色描述
	Description string
}

// UpdateRoleCommand 更新角色命令
type UpdateRoleCommand struct {
	// ID 角色ID
	ID role.RoleID

	// DisplayName 更新的显示名称（可选）
	DisplayName *string

	// Description 更新的描述（可选）
	Description *string
}

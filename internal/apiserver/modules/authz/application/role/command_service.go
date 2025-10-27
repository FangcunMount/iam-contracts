// Package role 角色应用服务
//
// 本包提供角色管理的应用服务，实现 domain/port/driving 接口。
// 应用服务负责编排领域服务和仓储，管理事务边界。
package role

import (
	"context"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/domain/role"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/domain/role/port/driven"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/domain/role/port/driving"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/domain/role/service"
)

// RoleCommandService 角色命令服务（写操作）
//
// 实现 driving.RoleCommander 接口
// 职责：
// - 编排领域服务和仓储完成角色的创建、更新、删除
// - 管理事务边界
// - 协调不同领域对象之间的交互
type RoleCommandService struct {
	roleManager *service.RoleManager // 领域服务（封装业务规则）
	roleRepo    driven.RoleRepo      // 仓储接口（数据持久化）
}

// NewRoleCommandService 创建角色命令服务
func NewRoleCommandService(
	roleManager *service.RoleManager,
	roleRepo driven.RoleRepo,
) driving.RoleCommander {
	return &RoleCommandService{
		roleManager: roleManager,
		roleRepo:    roleRepo,
	}
}

// CreateRole 创建角色
//
// 实现 driving.RoleCommander.CreateRole
// 编排流程：
// 1. 调用领域服务进行参数验证
// 2. 调用领域服务检查名称唯一性（租户内）
// 3. 创建领域对象
// 4. 持久化到仓储
func (s *RoleCommandService) CreateRole(
	ctx context.Context,
	cmd driving.CreateRoleCommand,
) (*role.Role, error) {
	// 1. 调用领域服务验证参数
	if err := s.roleManager.ValidateCreateParameters(cmd.Name, cmd.DisplayName, cmd.TenantID); err != nil {
		return nil, err
	}

	// 2. 调用领域服务检查名称唯一性（业务规则：租户内唯一）
	if err := s.roleManager.CheckNameUniqueness(ctx, cmd.TenantID, cmd.Name); err != nil {
		return nil, err
	}

	// 3. 创建角色领域对象
	newRole := role.NewRole(
		cmd.Name,
		cmd.DisplayName,
		cmd.TenantID,
		role.WithDescription(cmd.Description),
	)

	// 4. 持久化到仓储
	if err := s.roleRepo.Create(ctx, &newRole); err != nil {
		return nil, err
	}

	return &newRole, nil
}

// UpdateRole 更新角色
//
// 实现 driving.RoleCommander.UpdateRole
// 编排流程：
// 1. 调用领域服务检查角色是否存在
// 2. 更新领域对象属性
// 3. 持久化更新
func (s *RoleCommandService) UpdateRole(
	ctx context.Context,
	cmd driving.UpdateRoleCommand,
) (*role.Role, error) {
	// 1. 调用领域服务检查角色是否存在
	existingRole, err := s.roleManager.CheckRoleExists(ctx, cmd.ID)
	if err != nil {
		return nil, err
	}

	// 2. 更新领域对象属性
	if cmd.DisplayName != nil {
		existingRole.DisplayName = *cmd.DisplayName
	}
	if cmd.Description != nil {
		existingRole.Description = *cmd.Description
	}

	// 3. 持久化更新
	if err := s.roleRepo.Update(ctx, existingRole); err != nil {
		return nil, err
	}

	return existingRole, nil
}

// DeleteRole 删除角色
//
// 实现 driving.RoleCommander.DeleteRole
// 编排流程：
// 1. 调用领域服务检查角色是否存在
// 2. 删除角色
func (s *RoleCommandService) DeleteRole(
	ctx context.Context,
	roleID role.RoleID,
) error {
	// 1. 调用领域服务检查角色是否存在
	if _, err := s.roleManager.CheckRoleExists(ctx, roleID); err != nil {
		return err
	}

	// 2. 删除角色
	return s.roleRepo.Delete(ctx, roleID)
}

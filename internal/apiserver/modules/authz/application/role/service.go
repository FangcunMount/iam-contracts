// Package role 角色应用服务
package role

import (
	"context"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/role"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/role/port/driven"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	"github.com/fangcun-mount/iam-contracts/pkg/errors"
)

// Service 角色应用服务
type Service struct {
	roleRepo driven.RoleRepo
}

// NewService 创建角色应用服务
func NewService(roleRepo driven.RoleRepo) *Service {
	return &Service{
		roleRepo: roleRepo,
	}
}

// CreateRoleCommand 创建角色命令
type CreateRoleCommand struct {
	Name        string
	DisplayName string
	TenantID    string
	Description string
}

// CreateRole 创建角色
func (s *Service) CreateRole(ctx context.Context, cmd CreateRoleCommand) (*role.Role, error) {
	// 1. 验证参数
	if err := s.validateCreateCommand(cmd); err != nil {
		return nil, err
	}

	// 2. 检查角色名称是否已存在
	existingRole, err := s.roleRepo.FindByName(ctx, cmd.TenantID, cmd.Name)
	if err != nil && !errors.IsCode(err, code.ErrRoleNotFound) {
		return nil, errors.Wrap(err, "检查角色名称失败")
	}
	if existingRole != nil {
		return nil, errors.WithCode(code.ErrRoleAlreadyExists, "角色名称 %s 已存在", cmd.Name)
	}

	// 3. 创建角色领域对象
	newRole := role.NewRole(
		cmd.Name,
		cmd.DisplayName,
		cmd.TenantID,
		role.WithDescription(cmd.Description),
	)

	// 4. 保存到仓储
	if err := s.roleRepo.Create(ctx, &newRole); err != nil {
		return nil, errors.Wrap(err, "创建角色失败")
	}

	return &newRole, nil
}

// UpdateRoleCommand 更新角色命令
type UpdateRoleCommand struct {
	ID          role.RoleID
	DisplayName string
	Description string
}

// UpdateRole 更新角色
func (s *Service) UpdateRole(ctx context.Context, cmd UpdateRoleCommand) (*role.Role, error) {
	// 1. 验证参数
	if cmd.ID.Uint64() == 0 {
		return nil, errors.WithCode(code.ErrInvalidArgument, "角色ID不能为空")
	}

	// 2. 获取现有角色
	existingRole, err := s.roleRepo.FindByID(ctx, cmd.ID)
	if err != nil {
		if errors.IsCode(err, code.ErrRoleNotFound) {
			return nil, errors.WithCode(code.ErrRoleNotFound, "角色 %d 不存在", cmd.ID.Uint64())
		}
		return nil, errors.Wrap(err, "获取角色失败")
	}

	// 3. 更新字段
	if cmd.DisplayName != "" {
		existingRole.DisplayName = cmd.DisplayName
	}
	existingRole.Description = cmd.Description

	// 4. 保存更新
	if err := s.roleRepo.Update(ctx, existingRole); err != nil {
		return nil, errors.Wrap(err, "更新角色失败")
	}

	return existingRole, nil
}

// DeleteRole 删除角色
func (s *Service) DeleteRole(ctx context.Context, roleID role.RoleID, tenantID string) error {
	// 1. 验证参数
	if roleID.Uint64() == 0 {
		return errors.WithCode(code.ErrInvalidArgument, "角色ID不能为空")
	}
	if tenantID == "" {
		return errors.WithCode(code.ErrInvalidArgument, "租户ID不能为空")
	}

	// 2. 检查角色是否存在且属于该租户
	existingRole, err := s.roleRepo.FindByID(ctx, roleID)
	if err != nil {
		if errors.IsCode(err, code.ErrRoleNotFound) {
			return errors.WithCode(code.ErrRoleNotFound, "角色 %d 不存在", roleID.Uint64())
		}
		return errors.Wrap(err, "获取角色失败")
	}

	if existingRole.TenantID != tenantID {
		return errors.WithCode(code.ErrPermissionDenied, "无权删除其他租户的角色")
	}

	// 3. 删除角色
	if err := s.roleRepo.Delete(ctx, roleID); err != nil {
		return errors.Wrap(err, "删除角色失败")
	}

	return nil
}

// GetRoleByID 根据ID获取角色
func (s *Service) GetRoleByID(ctx context.Context, roleID role.RoleID, tenantID string) (*role.Role, error) {
	// 1. 验证参数
	if roleID.Uint64() == 0 {
		return nil, errors.WithCode(code.ErrInvalidArgument, "角色ID不能为空")
	}

	// 2. 获取角色
	foundRole, err := s.roleRepo.FindByID(ctx, roleID)
	if err != nil {
		if errors.IsCode(err, code.ErrRoleNotFound) {
			return nil, errors.WithCode(code.ErrRoleNotFound, "角色 %d 不存在", roleID.Uint64())
		}
		return nil, errors.Wrap(err, "获取角色失败")
	}

	// 3. 检查租户隔离
	if foundRole.TenantID != tenantID {
		return nil, errors.WithCode(code.ErrPermissionDenied, "无权访问其他租户的角色")
	}

	return foundRole, nil
}

// GetRoleByName 根据名称获取角色
func (s *Service) GetRoleByName(ctx context.Context, tenantID, name string) (*role.Role, error) {
	// 1. 验证参数
	if tenantID == "" {
		return nil, errors.WithCode(code.ErrInvalidArgument, "租户ID不能为空")
	}
	if name == "" {
		return nil, errors.WithCode(code.ErrInvalidArgument, "角色名称不能为空")
	}

	// 2. 获取角色
	foundRole, err := s.roleRepo.FindByName(ctx, tenantID, name)
	if err != nil {
		if errors.IsCode(err, code.ErrRoleNotFound) {
			return nil, errors.WithCode(code.ErrRoleNotFound, "角色 %s 不存在", name)
		}
		return nil, errors.Wrap(err, "获取角色失败")
	}

	return foundRole, nil
}

// ListRoleQuery 列出角色查询
type ListRoleQuery struct {
	TenantID string
	Offset   int
	Limit    int
}

// ListRoleResult 列出角色结果
type ListRoleResult struct {
	Roles []*role.Role
	Total int64
}

// ListRoles 列出角色
func (s *Service) ListRoles(ctx context.Context, query ListRoleQuery) (*ListRoleResult, error) {
	// 1. 验证参数
	if query.TenantID == "" {
		return nil, errors.WithCode(code.ErrInvalidArgument, "租户ID不能为空")
	}
	if query.Limit <= 0 {
		query.Limit = 10
	}
	if query.Offset < 0 {
		query.Offset = 0
	}

	// 2. 查询角色列表
	roles, total, err := s.roleRepo.List(ctx, query.TenantID, query.Offset, query.Limit)
	if err != nil {
		return nil, errors.Wrap(err, "列出角色失败")
	}

	return &ListRoleResult{
		Roles: roles,
		Total: total,
	}, nil
}

// validateCreateCommand 验证创建命令
func (s *Service) validateCreateCommand(cmd CreateRoleCommand) error {
	if cmd.Name == "" {
		return errors.WithCode(code.ErrInvalidArgument, "角色名称不能为空")
	}
	if cmd.DisplayName == "" {
		return errors.WithCode(code.ErrInvalidArgument, "显示名称不能为空")
	}
	if cmd.TenantID == "" {
		return errors.WithCode(code.ErrInvalidArgument, "租户ID不能为空")
	}
	return nil
}

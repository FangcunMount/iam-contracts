// Package role 角色应用服务
package role

import (
	"context"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/domain/role"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/domain/role/port/driven"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/domain/role/port/driving"
	"github.com/FangcunMount/iam-contracts/pkg/errors"
)

// RoleQueryService 角色查询服务（读操作）
//
// 实现 driving.RoleQueryer 接口
// 职责：
// - 提供角色的各种查询操作
// - 支持分页、过滤、租户隔离等查询场景
// - 可以在这一层加缓存优化
type RoleQueryService struct {
	roleRepo driven.RoleRepo // 仓储接口（数据读取）
}

// NewRoleQueryService 创建角色查询服务
func NewRoleQueryService(
	roleRepo driven.RoleRepo,
) driving.RoleQueryer {
	return &RoleQueryService{
		roleRepo: roleRepo,
	}
}

// GetRoleByID 根据ID获取角色
//
// 实现 driving.RoleQueryer.GetRoleByID
func (s *RoleQueryService) GetRoleByID(
	ctx context.Context,
	roleID role.RoleID,
) (*role.Role, error) {
	return s.roleRepo.FindByID(ctx, roleID)
}

// GetRoleByName 根据名称获取角色（租户内）
//
// 实现 driving.RoleQueryer.GetRoleByName
func (s *RoleQueryService) GetRoleByName(
	ctx context.Context,
	tenantID, name string,
) (*role.Role, error) {
	return s.roleRepo.FindByName(ctx, tenantID, name)
}

// ListRoles 列出角色（支持分页和租户过滤）
//
// 实现 driving.RoleQueryer.ListRoles
func (s *RoleQueryService) ListRoles(
	ctx context.Context,
	query driving.ListRolesQuery,
) (*driving.ListRolesResult, error) {
	// 设置默认分页参数
	if query.Limit <= 0 {
		query.Limit = 10
	}
	if query.Offset < 0 {
		query.Offset = 0
	}

	// 查询角色列表
	roles, total, err := s.roleRepo.List(ctx, query.TenantID, query.Offset, query.Limit)
	if err != nil {
		return nil, errors.Wrap(err, "列出角色失败")
	}

	return &driving.ListRolesResult{
		Roles: roles,
		Total: total,
	}, nil
}

// ListRolesByTenant 列出指定租户的所有角色
//
// 实现 driving.RoleQueryer.ListRolesByTenant
func (s *RoleQueryService) ListRolesByTenant(
	ctx context.Context,
	tenantID string,
) ([]*role.Role, error) {
	// 查询租户的所有角色（不分页）
	roles, _, err := s.roleRepo.List(ctx, tenantID, 0, 1000) // 使用较大的限制
	if err != nil {
		return nil, errors.Wrap(err, "列出租户角色失败")
	}

	return roles, nil
}

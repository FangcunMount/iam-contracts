// Package role 角色应用服务
package role

import (
	"context"

	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/management"

	roleDomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/role"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
)

// RoleQueryService 角色查询服务（读操作）
type RoleQueryService struct {
	guard    management.Guard
	roleRepo roleDomain.Repository
}

// NewRoleQueryService 创建角色查询服务
func NewRoleQueryService(
	roleRepo roleDomain.Repository,
	guards ...management.Guard,
) *RoleQueryService {
	var guard management.Guard
	if len(guards) > 0 {
		guard = guards[0]
	}
	return &RoleQueryService{
		guard:    guard,
		roleRepo: roleRepo,
	}
}

// GetRoleByID 根据ID获取角色
func (s *RoleQueryService) GetRoleByID(
	ctx context.Context,

	roleID meta.ID,
) (*roleDomain.Role, error) {
	target, err := s.roleRepo.FindByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if err = s.guard.RequireVisible(ctx, target); err != nil {
		return nil, err
	}
	return target, nil
}

// GetRoleByName 根据名称获取角色（租户内）
func (s *RoleQueryService) GetRoleByName(
	ctx context.Context,
	name string,
) (*roleDomain.Role, error) {
	target, err := s.roleRepo.FindByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if err = s.guard.RequireVisible(ctx, target); err != nil {
		return nil, err
	}
	return target, nil
}

// ListRoles 列出角色（支持分页和租户过滤）
func (s *RoleQueryService) ListRoles(
	ctx context.Context,
	query ListRolesQuery,
) (*ListRolesResult, error) {
	roles, _, err := s.roleRepo.List(ctx, 0, -1)
	if err != nil {
		return nil, err
	}

	roles, err = s.visible(ctx, roles)
	if err != nil {
		return nil, err
	}
	total := int64(len(roles))
	offset := max(0, query.Offset)
	if offset > len(roles) {
		offset = len(roles)
	}
	roles = roles[offset:]
	if query.Limit >= 0 && query.Limit < len(roles) {
		roles = roles[:query.Limit]
	}
	return &ListRolesResult{
		Roles: roles,
		Total: total,
	}, nil
}

// ListAllRoles 列出指定租户的所有角色
func (s *RoleQueryService) ListAllRoles(
	ctx context.Context,

) ([]*roleDomain.Role, error) {
	// 不限制分页，返回所有角色
	roles, _, err := s.roleRepo.List(ctx, 0, -1)
	if err != nil {
		return nil, err
	}
	return s.visible(ctx, roles)
}

func (s *RoleQueryService) visible(ctx context.Context, roles []*roleDomain.Role) ([]*roleDomain.Role, error) {
	out := make([]*roleDomain.Role, 0, len(roles))
	for _, r := range roles {
		ok, err := s.guard.Visible(ctx, r)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, r)
		}
	}
	return out, nil
}

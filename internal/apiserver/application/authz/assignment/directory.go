// Package assignment 角色赋权查询应用服务。
package assignment

import (
	"context"

	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/management"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/role"

	"github.com/FangcunMount/component-base/pkg/errors"
	assignmentDomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/assignment"
)

// DirectoryService 负责角色赋权读操作。
type DirectoryService struct {
	roles               role.Repository
	guard               management.Guard
	assignmentValidator assignmentDomain.Validator
	assignmentRepo      assignmentDomain.Repository
}

func NewDirectory(
	assignmentValidator assignmentDomain.Validator,
	assignmentRepo assignmentDomain.Repository,
	roles role.Repository, guard management.Guard,
) *DirectoryService {
	return &DirectoryService{roles: roles, guard: guard,
		assignmentValidator: assignmentValidator,
		assignmentRepo:      assignmentRepo,
	}
}

// ListBySubject 根据主体列出赋权
func (s *DirectoryService) ListBySubject(ctx context.Context, query ListBySubjectQuery) ([]*assignmentDomain.Assignment, error) {
	// 1. 直接查询赋权列表（验证由领域层Repository处理）
	assignments, err := s.assignmentRepo.ListBySubject(ctx, query.SubjectType, query.SubjectID)
	if err != nil {
		return nil, errors.Wrap(err, "查询赋权列表失败")
	}

	return s.visible(ctx, assignments)
}

// ListByRole 根据角色列出赋权
func (s *DirectoryService) ListByRole(ctx context.Context, query ListByRoleQuery) ([]*assignmentDomain.Assignment, error) {
	// 1. 检查角色是否存在
	if err := s.assignmentValidator.CheckRoleExists(ctx, query.RoleID); err != nil {
		return nil, err
	}

	target, err := s.roles.FindByID(ctx, query.RoleID)
	if err != nil {
		return nil, err
	}
	if err = s.guard.RequireVisible(ctx, target); err != nil {
		return nil, err
	}
	// 2. 查询赋权列表
	assignments, err := s.assignmentRepo.ListByRole(ctx, query.RoleID)
	if err != nil {
		return nil, errors.Wrap(err, "查询赋权列表失败")
	}

	return s.visible(ctx, assignments)
}

func (s *DirectoryService) visible(ctx context.Context, items []*assignmentDomain.Assignment) ([]*assignmentDomain.Assignment, error) {
	out := make([]*assignmentDomain.Assignment, 0, len(items))
	for _, a := range items {
		r, err := s.roles.FindByID(ctx, a.RoleID)
		if err != nil {
			return nil, err
		}
		ok, err := s.guard.Visible(ctx, r)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, a)
		}
	}
	return out, nil
}

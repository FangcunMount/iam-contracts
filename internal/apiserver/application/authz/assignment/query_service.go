// Package assignment 赋权查询应用服务
package assignment

import (
	"context"

	"github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authz/assignment"
	assignmentDriven "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authz/assignment/port/driven"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authz/assignment/port/driving"
	assignmentService "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authz/assignment/service"
)

// AssignmentQueryService 赋权查询服务（实现 AssignmentQueryer 接口）
// 负责赋权的读操作，遵循 CQRS 原则
type AssignmentQueryService struct {
	assignmentManager *assignmentService.AssignmentManager
	assignmentRepo    assignmentDriven.AssignmentRepo
}

// NewAssignmentQueryService 创建赋权查询服务
func NewAssignmentQueryService(
	assignmentManager *assignmentService.AssignmentManager,
	assignmentRepo assignmentDriven.AssignmentRepo,
) *AssignmentQueryService {
	return &AssignmentQueryService{
		assignmentManager: assignmentManager,
		assignmentRepo:    assignmentRepo,
	}
}

// ListBySubject 根据主体列出赋权
func (s *AssignmentQueryService) ListBySubject(ctx context.Context, query driving.ListBySubjectQuery) ([]*assignment.Assignment, error) {
	// 1. 验证参数
	if err := s.assignmentManager.ValidateListBySubjectQuery(query.SubjectID, query.TenantID); err != nil {
		return nil, err
	}

	// 2. 查询赋权列表
	assignments, err := s.assignmentRepo.ListBySubject(ctx, query.SubjectType, query.SubjectID, query.TenantID)
	if err != nil {
		return nil, errors.Wrap(err, "查询赋权列表失败")
	}

	return assignments, nil
}

// ListByRole 根据角色列出赋权
func (s *AssignmentQueryService) ListByRole(ctx context.Context, query driving.ListByRoleQuery) ([]*assignment.Assignment, error) {
	// 1. 验证参数
	if err := s.assignmentManager.ValidateListByRoleQuery(query.RoleID, query.TenantID); err != nil {
		return nil, err
	}

	// 2. 检查角色是否存在并验证租户隔离
	if _, err := s.assignmentManager.CheckRoleExistsAndTenant(ctx, query.RoleID, query.TenantID); err != nil {
		return nil, err
	}

	// 3. 查询赋权列表
	assignments, err := s.assignmentRepo.ListByRole(ctx, query.RoleID, query.TenantID)
	if err != nil {
		return nil, errors.Wrap(err, "查询赋权列表失败")
	}

	return assignments, nil
}

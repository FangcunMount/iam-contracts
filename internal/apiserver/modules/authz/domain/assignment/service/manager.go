// Package service 赋权领域服务
package service

import (
	"context"

	"github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/domain/assignment"
	assignmentDriven "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/domain/assignment/port/driven"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/domain/role"
	roleDriven "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/domain/role/port/driven"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
)

// AssignmentManager 赋权管理器（领域服务）
// 封装赋权相关的业务规则，包括：
// 1. 赋权参数验证
// 2. 角色存在性检查
// 3. 租户隔离检查
// 4. 赋权记录查找
type AssignmentManager struct {
	assignmentRepo assignmentDriven.AssignmentRepo
	roleRepo       roleDriven.RoleRepo
}

// NewAssignmentManager 创建赋权管理器
func NewAssignmentManager(
	assignmentRepo assignmentDriven.AssignmentRepo,
	roleRepo roleDriven.RoleRepo,
) *AssignmentManager {
	return &AssignmentManager{
		assignmentRepo: assignmentRepo,
		roleRepo:       roleRepo,
	}
}

// ValidateGrantParameters 验证授权参数
func (m *AssignmentManager) ValidateGrantParameters(
	subjectType assignment.SubjectType,
	subjectID string,
	roleID uint64,
	tenantID string,
	grantedBy string,
) error {
	if subjectType == "" {
		return errors.WithCode(code.ErrInvalidArgument, "主体类型不能为空")
	}
	if subjectID == "" {
		return errors.WithCode(code.ErrInvalidArgument, "主体ID不能为空")
	}
	if roleID == 0 {
		return errors.WithCode(code.ErrInvalidArgument, "角色ID不能为空")
	}
	if tenantID == "" {
		return errors.WithCode(code.ErrInvalidArgument, "租户ID不能为空")
	}
	if grantedBy == "" {
		return errors.WithCode(code.ErrInvalidArgument, "授权人不能为空")
	}
	return nil
}

// ValidateRevokeParameters 验证撤销授权参数
func (m *AssignmentManager) ValidateRevokeParameters(
	subjectType assignment.SubjectType,
	subjectID string,
	roleID uint64,
	tenantID string,
) error {
	if subjectType == "" {
		return errors.WithCode(code.ErrInvalidArgument, "主体类型不能为空")
	}
	if subjectID == "" {
		return errors.WithCode(code.ErrInvalidArgument, "主体ID不能为空")
	}
	if roleID == 0 {
		return errors.WithCode(code.ErrInvalidArgument, "角色ID不能为空")
	}
	if tenantID == "" {
		return errors.WithCode(code.ErrInvalidArgument, "租户ID不能为空")
	}
	return nil
}

// ValidateRevokeByIDParameters 验证根据ID撤销授权参数
func (m *AssignmentManager) ValidateRevokeByIDParameters(
	assignmentID assignment.AssignmentID,
	tenantID string,
) error {
	if assignmentID.Uint64() == 0 {
		return errors.WithCode(code.ErrInvalidArgument, "赋权ID不能为空")
	}
	if tenantID == "" {
		return errors.WithCode(code.ErrInvalidArgument, "租户ID不能为空")
	}
	return nil
}

// CheckRoleExistsAndTenant 检查角色是否存在并验证租户隔离
// 返回角色实体用于后续操作
func (m *AssignmentManager) CheckRoleExistsAndTenant(
	ctx context.Context,
	roleID uint64,
	tenantID string,
) (*role.Role, error) {
	roleExists, err := m.roleRepo.FindByID(ctx, role.NewRoleID(roleID))
	if err != nil {
		if errors.IsCode(err, code.ErrRoleNotFound) {
			return nil, errors.WithCode(code.ErrRoleNotFound, "角色 %d 不存在", roleID)
		}
		return nil, errors.Wrap(err, "获取角色失败")
	}

	// 检查租户隔离
	if roleExists.TenantID != tenantID {
		return nil, errors.WithCode(code.ErrPermissionDenied, "无权操作其他租户的角色")
	}

	return roleExists, nil
}

// FindAssignmentBySubjectAndRole 查找主体和角色的赋权记录
func (m *AssignmentManager) FindAssignmentBySubjectAndRole(
	ctx context.Context,
	subjectType assignment.SubjectType,
	subjectID string,
	roleID uint64,
	tenantID string,
) (*assignment.Assignment, error) {
	// 查询赋权列表
	assignments, err := m.assignmentRepo.ListBySubject(ctx, subjectType, subjectID, tenantID)
	if err != nil {
		return nil, errors.Wrap(err, "查询赋权记录失败")
	}

	// 查找匹配的赋权记录
	for _, a := range assignments {
		if a.RoleID == roleID {
			return a, nil
		}
	}

	return nil, errors.WithCode(code.ErrAssignmentNotFound, "赋权记录不存在")
}

// GetAssignmentByIDAndCheckTenant 根据ID获取赋权记录并检查租户隔离
func (m *AssignmentManager) GetAssignmentByIDAndCheckTenant(
	ctx context.Context,
	assignmentID assignment.AssignmentID,
	tenantID string,
) (*assignment.Assignment, error) {
	// 获取赋权记录
	targetAssignment, err := m.assignmentRepo.FindByID(ctx, assignmentID)
	if err != nil {
		if errors.IsCode(err, code.ErrAssignmentNotFound) {
			return nil, errors.WithCode(code.ErrAssignmentNotFound, "赋权记录不存在")
		}
		return nil, errors.Wrap(err, "获取赋权记录失败")
	}

	// 检查租户隔离
	if targetAssignment.TenantID != tenantID {
		return nil, errors.WithCode(code.ErrPermissionDenied, "无权操作其他租户的赋权记录")
	}

	return targetAssignment, nil
}

// ValidateListBySubjectQuery 验证根据主体查询参数
func (m *AssignmentManager) ValidateListBySubjectQuery(subjectID string, tenantID string) error {
	if subjectID == "" {
		return errors.WithCode(code.ErrInvalidArgument, "主体ID不能为空")
	}
	if tenantID == "" {
		return errors.WithCode(code.ErrInvalidArgument, "租户ID不能为空")
	}
	return nil
}

// ValidateListByRoleQuery 验证根据角色查询参数
func (m *AssignmentManager) ValidateListByRoleQuery(roleID uint64, tenantID string) error {
	if roleID == 0 {
		return errors.WithCode(code.ErrInvalidArgument, "角色ID不能为空")
	}
	if tenantID == "" {
		return errors.WithCode(code.ErrInvalidArgument, "租户ID不能为空")
	}
	return nil
}

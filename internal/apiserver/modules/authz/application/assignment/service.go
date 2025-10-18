// Package assignment 赋权应用服务
package assignment

import (
	"context"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/assignment"
	assignmentDriven "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/assignment/port/driven"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/policy"
	policyDriven "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/policy/port/driven"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/role"
	roleDriven "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/role/port/driven"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	"github.com/fangcun-mount/iam-contracts/pkg/errors"
)

// Service 赋权应用服务
type Service struct {
	assignmentRepo assignmentDriven.AssignmentRepo
	roleRepo       roleDriven.RoleRepo
	casbinPort     policyDriven.CasbinPort
}

// NewService 创建赋权应用服务
func NewService(
	assignmentRepo assignmentDriven.AssignmentRepo,
	roleRepo roleDriven.RoleRepo,
	casbinPort policyDriven.CasbinPort,
) *Service {
	return &Service{
		assignmentRepo: assignmentRepo,
		roleRepo:       roleRepo,
		casbinPort:     casbinPort,
	}
}

// GrantCommand 授权命令
type GrantCommand struct {
	SubjectType assignment.SubjectType
	SubjectID   string
	RoleID      uint64
	TenantID    string
	GrantedBy   string
}

// Grant 授权（赋予角色）
func (s *Service) Grant(ctx context.Context, cmd GrantCommand) (*assignment.Assignment, error) {
	// 1. 验证参数
	if err := s.validateGrantCommand(cmd); err != nil {
		return nil, err
	}

	// 2. 检查角色是否存在
	roleExists, err := s.roleRepo.FindByID(ctx, role.NewRoleID(cmd.RoleID))
	if err != nil {
		if errors.IsCode(err, code.ErrRoleNotFound) {
			return nil, errors.WithCode(code.ErrRoleNotFound, "角色 %d 不存在", cmd.RoleID)
		}
		return nil, errors.Wrap(err, "获取角色失败")
	}

	// 3. 检查租户隔离
	if roleExists.TenantID != cmd.TenantID {
		return nil, errors.WithCode(code.ErrPermissionDenied, "无权操作其他租户的角色")
	}

	// 4. 创建赋权领域对象
	newAssignment := assignment.NewAssignment(
		cmd.SubjectType,
		cmd.SubjectID,
		cmd.RoleID,
		cmd.TenantID,
		assignment.WithGrantedBy(cmd.GrantedBy),
	)

	// 5. 保存到数据库
	if err := s.assignmentRepo.Create(ctx, &newAssignment); err != nil {
		return nil, errors.Wrap(err, "创建赋权失败")
	}

	// 6. 添加 Casbin g 规则
	groupingRule := policy.GroupingRule{
		Sub:  newAssignment.SubjectKey(),
		Role: newAssignment.RoleKey(),
		Dom:  cmd.TenantID,
	}
	if err := s.casbinPort.AddGroupingPolicy(ctx, groupingRule); err != nil {
		// 回滚：删除数据库记录
		_ = s.assignmentRepo.Delete(ctx, newAssignment.ID)
		return nil, errors.Wrap(err, "添加 Casbin 规则失败")
	}

	return &newAssignment, nil
}

// RevokeCommand 撤销授权命令
type RevokeCommand struct {
	SubjectType assignment.SubjectType
	SubjectID   string
	RoleID      uint64
	TenantID    string
}

// Revoke 撤销授权（移除角色）
func (s *Service) Revoke(ctx context.Context, cmd RevokeCommand) error {
	// 1. 验证参数
	if err := s.validateRevokeCommand(cmd); err != nil {
		return err
	}

	// 2. 查询赋权记录
	assignments, err := s.assignmentRepo.ListBySubject(ctx, cmd.SubjectType, cmd.SubjectID, cmd.TenantID)
	if err != nil {
		return errors.Wrap(err, "查询赋权记录失败")
	}

	// 3. 查找匹配的赋权记录
	var targetAssignment *assignment.Assignment
	for _, a := range assignments {
		if a.RoleID == cmd.RoleID {
			targetAssignment = a
			break
		}
	}

	if targetAssignment == nil {
		return errors.WithCode(code.ErrAssignmentNotFound, "赋权记录不存在")
	}

	// 4. 构建 Casbin g 规则
	groupingRule := policy.GroupingRule{
		Sub:  targetAssignment.SubjectKey(),
		Role: targetAssignment.RoleKey(),
		Dom:  cmd.TenantID,
	}

	// 5. 删除 Casbin g 规则
	if err := s.casbinPort.RemoveGroupingPolicy(ctx, groupingRule); err != nil {
		return errors.Wrap(err, "删除 Casbin 规则失败")
	}

	// 6. 删除数据库记录
	if err := s.assignmentRepo.Delete(ctx, targetAssignment.ID); err != nil {
		// 尝试回滚：重新添加 Casbin 规则
		_ = s.casbinPort.AddGroupingPolicy(ctx, groupingRule)
		return errors.Wrap(err, "删除赋权记录失败")
	}

	return nil
}

// RevokeByIDCommand 根据ID撤销授权命令
type RevokeByIDCommand struct {
	AssignmentID assignment.AssignmentID
	TenantID     string
}

// RevokeByID 根据ID撤销授权
func (s *Service) RevokeByID(ctx context.Context, cmd RevokeByIDCommand) error {
	// 1. 验证参数
	if cmd.AssignmentID.Uint64() == 0 {
		return errors.WithCode(code.ErrInvalidArgument, "赋权ID不能为空")
	}
	if cmd.TenantID == "" {
		return errors.WithCode(code.ErrInvalidArgument, "租户ID不能为空")
	}

	// 2. 获取赋权记录
	targetAssignment, err := s.assignmentRepo.FindByID(ctx, cmd.AssignmentID)
	if err != nil {
		if errors.IsCode(err, code.ErrAssignmentNotFound) {
			return errors.WithCode(code.ErrAssignmentNotFound, "赋权记录不存在")
		}
		return errors.Wrap(err, "获取赋权记录失败")
	}

	// 3. 检查租户隔离
	if targetAssignment.TenantID != cmd.TenantID {
		return errors.WithCode(code.ErrPermissionDenied, "无权操作其他租户的赋权记录")
	}

	// 4. 构建 Casbin g 规则
	groupingRule := policy.GroupingRule{
		Sub:  targetAssignment.SubjectKey(),
		Role: targetAssignment.RoleKey(),
		Dom:  targetAssignment.TenantID,
	}

	// 5. 删除 Casbin g 规则
	if err := s.casbinPort.RemoveGroupingPolicy(ctx, groupingRule); err != nil {
		return errors.Wrap(err, "删除 Casbin 规则失败")
	}

	// 6. 删除数据库记录
	if err := s.assignmentRepo.Delete(ctx, targetAssignment.ID); err != nil {
		// 尝试回滚：重新添加 Casbin 规则
		_ = s.casbinPort.AddGroupingPolicy(ctx, groupingRule)
		return errors.Wrap(err, "删除赋权记录失败")
	}

	return nil
}

// ListBySubjectQuery 根据主体列出赋权查询
type ListBySubjectQuery struct {
	SubjectType assignment.SubjectType
	SubjectID   string
	TenantID    string
}

// ListBySubject 根据主体列出赋权
func (s *Service) ListBySubject(ctx context.Context, query ListBySubjectQuery) ([]*assignment.Assignment, error) {
	// 1. 验证参数
	if query.SubjectID == "" {
		return nil, errors.WithCode(code.ErrInvalidArgument, "主体ID不能为空")
	}
	if query.TenantID == "" {
		return nil, errors.WithCode(code.ErrInvalidArgument, "租户ID不能为空")
	}

	// 2. 查询赋权列表
	assignments, err := s.assignmentRepo.ListBySubject(ctx, query.SubjectType, query.SubjectID, query.TenantID)
	if err != nil {
		return nil, errors.Wrap(err, "查询赋权列表失败")
	}

	return assignments, nil
}

// ListByRoleQuery 根据角色列出赋权查询
type ListByRoleQuery struct {
	RoleID   uint64
	TenantID string
}

// ListByRole 根据角色列出赋权
func (s *Service) ListByRole(ctx context.Context, query ListByRoleQuery) ([]*assignment.Assignment, error) {
	// 1. 验证参数
	if query.RoleID == 0 {
		return nil, errors.WithCode(code.ErrInvalidArgument, "角色ID不能为空")
	}
	if query.TenantID == "" {
		return nil, errors.WithCode(code.ErrInvalidArgument, "租户ID不能为空")
	}

	// 2. 检查角色是否存在
	roleExists, err := s.roleRepo.FindByID(ctx, role.NewRoleID(query.RoleID))
	if err != nil {
		if errors.IsCode(err, code.ErrRoleNotFound) {
			return nil, errors.WithCode(code.ErrRoleNotFound, "角色 %d 不存在", query.RoleID)
		}
		return nil, errors.Wrap(err, "获取角色失败")
	}

	// 3. 检查租户隔离
	if roleExists.TenantID != query.TenantID {
		return nil, errors.WithCode(code.ErrPermissionDenied, "无权访问其他租户的角色")
	}

	// 4. 查询赋权列表
	assignments, err := s.assignmentRepo.ListByRole(ctx, query.RoleID, query.TenantID)
	if err != nil {
		return nil, errors.Wrap(err, "查询赋权列表失败")
	}

	return assignments, nil
}

// validateGrantCommand 验证授权命令
func (s *Service) validateGrantCommand(cmd GrantCommand) error {
	if cmd.SubjectType == "" {
		return errors.WithCode(code.ErrInvalidArgument, "主体类型不能为空")
	}
	if cmd.SubjectID == "" {
		return errors.WithCode(code.ErrInvalidArgument, "主体ID不能为空")
	}
	if cmd.RoleID == 0 {
		return errors.WithCode(code.ErrInvalidArgument, "角色ID不能为空")
	}
	if cmd.TenantID == "" {
		return errors.WithCode(code.ErrInvalidArgument, "租户ID不能为空")
	}
	if cmd.GrantedBy == "" {
		return errors.WithCode(code.ErrInvalidArgument, "授权人不能为空")
	}
	return nil
}

// validateRevokeCommand 验证撤销授权命令
func (s *Service) validateRevokeCommand(cmd RevokeCommand) error {
	if cmd.SubjectType == "" {
		return errors.WithCode(code.ErrInvalidArgument, "主体类型不能为空")
	}
	if cmd.SubjectID == "" {
		return errors.WithCode(code.ErrInvalidArgument, "主体ID不能为空")
	}
	if cmd.RoleID == 0 {
		return errors.WithCode(code.ErrInvalidArgument, "角色ID不能为空")
	}
	if cmd.TenantID == "" {
		return errors.WithCode(code.ErrInvalidArgument, "租户ID不能为空")
	}
	return nil
}

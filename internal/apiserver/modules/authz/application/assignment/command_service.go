// Package assignment 赋权命令应用服务
package assignment

import (
	"context"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/assignment"
	assignmentDriven "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/assignment/port/driven"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/assignment/port/driving"
	assignmentService "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/assignment/service"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/policy"
	policyDriven "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/policy/port/driven"
	"github.com/fangcun-mount/iam-contracts/pkg/errors"
)

// AssignmentCommandService 赋权命令服务（实现 AssignmentCommander 接口）
// 负责协调领域服务、仓储和外部端口，处理赋权的写操作
// 核心职责：管理数据库和 Casbin 的双写事务一致性
type AssignmentCommandService struct {
	assignmentManager *assignmentService.AssignmentManager
	assignmentRepo    assignmentDriven.AssignmentRepo
	casbinPort        policyDriven.CasbinPort
}

// NewAssignmentCommandService 创建赋权命令服务
func NewAssignmentCommandService(
	assignmentManager *assignmentService.AssignmentManager,
	assignmentRepo assignmentDriven.AssignmentRepo,
	casbinPort policyDriven.CasbinPort,
) *AssignmentCommandService {
	return &AssignmentCommandService{
		assignmentManager: assignmentManager,
		assignmentRepo:    assignmentRepo,
		casbinPort:        casbinPort,
	}
}

// Grant 授权（赋予角色）
func (s *AssignmentCommandService) Grant(ctx context.Context, cmd driving.GrantCommand) (*assignment.Assignment, error) {
	// 1. 验证参数
	if err := s.assignmentManager.ValidateGrantParameters(
		cmd.SubjectType, cmd.SubjectID, cmd.RoleID, cmd.TenantID, cmd.GrantedBy,
	); err != nil {
		return nil, err
	}

	// 2. 检查角色是否存在并验证租户隔离
	roleExists, err := s.assignmentManager.CheckRoleExistsAndTenant(ctx, cmd.RoleID, cmd.TenantID)
	if err != nil {
		return nil, err
	}

	// 3. 创建赋权领域对象
	newAssignment := assignment.NewAssignment(
		cmd.SubjectType,
		cmd.SubjectID,
		cmd.RoleID,
		cmd.TenantID,
		assignment.WithGrantedBy(cmd.GrantedBy),
	)

	// 4. 保存到数据库
	if err := s.assignmentRepo.Create(ctx, &newAssignment); err != nil {
		return nil, errors.Wrap(err, "创建赋权失败")
	}

	// 5. 添加 Casbin 分组规则（g 规则）
	groupingRule := policy.GroupingRule{
		Sub:  newAssignment.SubjectKey(),
		Role: roleExists.Key(),
		Dom:  cmd.TenantID,
	}
	if err := s.casbinPort.AddGroupingPolicy(ctx, groupingRule); err != nil {
		// 回滚：删除数据库记录
		_ = s.assignmentRepo.Delete(ctx, newAssignment.ID)
		return nil, errors.Wrap(err, "添加 Casbin 分组规则失败")
	}

	return &newAssignment, nil
}

// Revoke 撤销授权（移除角色）
func (s *AssignmentCommandService) Revoke(ctx context.Context, cmd driving.RevokeCommand) error {
	// 1. 验证参数
	if err := s.assignmentManager.ValidateRevokeParameters(
		cmd.SubjectType, cmd.SubjectID, cmd.RoleID, cmd.TenantID,
	); err != nil {
		return err
	}

	// 2. 查找赋权记录
	targetAssignment, err := s.assignmentManager.FindAssignmentBySubjectAndRole(
		ctx, cmd.SubjectType, cmd.SubjectID, cmd.RoleID, cmd.TenantID,
	)
	if err != nil {
		return err
	}

	// 3. 执行撤销操作（数据库 + Casbin 双写）
	return s.revokeAssignment(ctx, targetAssignment)
}

// RevokeByID 根据ID撤销授权
func (s *AssignmentCommandService) RevokeByID(ctx context.Context, cmd driving.RevokeByIDCommand) error {
	// 1. 验证参数
	if err := s.assignmentManager.ValidateRevokeByIDParameters(cmd.AssignmentID, cmd.TenantID); err != nil {
		return err
	}

	// 2. 获取赋权记录并检查租户隔离
	targetAssignment, err := s.assignmentManager.GetAssignmentByIDAndCheckTenant(
		ctx, cmd.AssignmentID, cmd.TenantID,
	)
	if err != nil {
		return err
	}

	// 3. 执行撤销操作（数据库 + Casbin 双写）
	return s.revokeAssignment(ctx, targetAssignment)
}

// revokeAssignment 撤销赋权记录（内部辅助方法）
// 负责协调数据库删除和 Casbin 规则删除，保证事务一致性
func (s *AssignmentCommandService) revokeAssignment(
	ctx context.Context,
	targetAssignment *assignment.Assignment,
) error {
	// 1. 构建 Casbin 分组规则
	groupingRule := policy.GroupingRule{
		Sub:  targetAssignment.SubjectKey(),
		Role: targetAssignment.RoleKey(),
		Dom:  targetAssignment.TenantID,
	}

	// 2. 删除 Casbin 分组规则
	if err := s.casbinPort.RemoveGroupingPolicy(ctx, groupingRule); err != nil {
		return errors.Wrap(err, "删除 Casbin 分组规则失败")
	}

	// 3. 删除数据库记录
	if err := s.assignmentRepo.Delete(ctx, targetAssignment.ID); err != nil {
		// 尝试回滚：重新添加 Casbin 规则
		_ = s.casbinPort.AddGroupingPolicy(ctx, groupingRule)
		return errors.Wrap(err, "删除赋权记录失败")
	}

	return nil
}

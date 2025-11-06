// Package assignment 赋权命令应用服务
package assignment

import (
	"context"

	"github.com/FangcunMount/component-base/pkg/errors"
	assignmentDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authz/assignment"
	policyDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authz/policy"
	roleDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authz/role"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

// AssignmentCommandService 赋权命令服务（实现 AssignmentCommander 接口）
// 负责协调领域服务、仓储和外部端口，处理赋权的写操作
// 核心职责：管理数据库和 Casbin 的双写事务一致性
type AssignmentCommandService struct {
	assignmentValidator assignmentDomain.Validator
	assignmentRepo      assignmentDomain.Repository
	roleRepo            roleDomain.Repository
	casbinAdapter       policyDomain.CasbinAdapter
}

// NewAssignmentCommandService 创建赋权命令服务
func NewAssignmentCommandService(
	assignmentValidator assignmentDomain.Validator,
	assignmentRepo assignmentDomain.Repository,
	roleRepo roleDomain.Repository,
	casbinAdapter policyDomain.CasbinAdapter,
) *AssignmentCommandService {
	return &AssignmentCommandService{
		assignmentValidator: assignmentValidator,
		assignmentRepo:      assignmentRepo,
		roleRepo:            roleRepo,
		casbinAdapter:       casbinAdapter,
	}
}

// Grant 授权（赋予角色）
func (s *AssignmentCommandService) Grant(ctx context.Context, cmd assignmentDomain.GrantCommand) (*assignmentDomain.Assignment, error) {
	// 1. 验证授权命令
	if err := s.assignmentValidator.ValidateGrantCommand(cmd); err != nil {
		return nil, err
	}

	// 2. 检查角色是否存在
	if err := s.assignmentValidator.CheckRoleExists(ctx, cmd.RoleID, cmd.TenantID); err != nil {
		return nil, err
	}

	// 3. 获取角色信息以构建 Casbin 规则
	role, err := s.roleRepo.FindByID(ctx, meta.NewID(cmd.RoleID))
	if err != nil {
		return nil, errors.Wrap(err, "获取角色失败")
	}
	if role.TenantID != cmd.TenantID {
		return nil, errors.New("角色不属于当前租户")
	}

	// 4. 创建赋权领域对象
	newAssignment := assignmentDomain.NewAssignment(
		cmd.SubjectType,
		cmd.SubjectID,
		cmd.RoleID,
		cmd.TenantID,
		assignmentDomain.WithGrantedBy(cmd.GrantedBy),
	)

	// 5. 保存到数据库
	if err := s.assignmentRepo.Create(ctx, &newAssignment); err != nil {
		return nil, errors.Wrap(err, "创建赋权失败")
	}

	// 6. 添加 Casbin 分组规则（g 规则）
	groupingRule := policyDomain.GroupingRule{
		Sub:  newAssignment.SubjectKey(),
		Role: role.Key(),
		Dom:  cmd.TenantID,
	}
	if err := s.casbinAdapter.AddGroupingPolicy(ctx, groupingRule); err != nil {
		// 回滚：删除数据库记录
		_ = s.assignmentRepo.Delete(ctx, newAssignment.ID)
		return nil, errors.Wrap(err, "添加 Casbin 分组规则失败")
	}

	return &newAssignment, nil
}

// Revoke 撤销授权（移除角色）
func (s *AssignmentCommandService) Revoke(ctx context.Context, cmd assignmentDomain.RevokeCommand) error {
	// 1. 验证撤销命令
	if err := s.assignmentValidator.ValidateRevokeCommand(cmd); err != nil {
		return err
	}

	// 2. 删除赋权记录（直接使用 Repository 方法）
	if err := s.assignmentRepo.DeleteBySubjectAndRole(ctx, cmd.SubjectType, cmd.SubjectID, cmd.RoleID, cmd.TenantID); err != nil {
		return errors.Wrap(err, "删除赋权记录失败")
	}

	// 3. 获取角色信息以构建 Casbin 规则
	role, err := s.roleRepo.FindByID(ctx, meta.NewID(cmd.RoleID))
	if err != nil {
		return errors.Wrap(err, "获取角色失败")
	}

	// 4. 删除 Casbin 分组规则
	subjectKey := string(cmd.SubjectType) + ":" + cmd.SubjectID
	groupingRule := policyDomain.GroupingRule{
		Sub:  subjectKey,
		Role: role.Key(),
		Dom:  cmd.TenantID,
	}
	if err := s.casbinAdapter.RemoveGroupingPolicy(ctx, groupingRule); err != nil {
		return errors.Wrap(err, "删除 Casbin 分组规则失败")
	}

	return nil
}

// RevokeByID 根据ID撤销授权
func (s *AssignmentCommandService) RevokeByID(ctx context.Context, cmd assignmentDomain.RevokeByIDCommand) error {
	// 1. 获取赋权记录
	targetAssignment, err := s.assignmentRepo.FindByID(ctx, cmd.AssignmentID)
	if err != nil {
		return errors.Wrap(err, "获取赋权记录失败")
	}

	// 2. 验证租户隔离
	if targetAssignment.TenantID != cmd.TenantID {
		return errors.New("赋权记录不属于当前租户")
	}

	// 3. 执行撤销操作（数据库 + Casbin 双写）
	return s.revokeAssignment(ctx, targetAssignment)
}

// revokeAssignment 撤销赋权记录（内部辅助方法）
// 负责协调数据库删除和 Casbin 规则删除，保证事务一致性
func (s *AssignmentCommandService) revokeAssignment(
	ctx context.Context,
	targetAssignment *assignmentDomain.Assignment,
) error {
	// 1. 获取角色信息以构建 Casbin 规则
	role, err := s.roleRepo.FindByID(ctx, meta.NewID(targetAssignment.RoleID))
	if err != nil {
		return errors.Wrap(err, "获取角色失败")
	}

	// 2. 构建 Casbin 分组规则
	groupingRule := policyDomain.GroupingRule{
		Sub:  targetAssignment.SubjectKey(),
		Role: role.Key(),
		Dom:  targetAssignment.TenantID,
	}

	// 3. 删除 Casbin 分组规则
	if err := s.casbinAdapter.RemoveGroupingPolicy(ctx, groupingRule); err != nil {
		return errors.Wrap(err, "删除 Casbin 分组规则失败")
	}

	// 4. 删除数据库记录
	if err := s.assignmentRepo.Delete(ctx, targetAssignment.ID); err != nil {
		// 尝试回滚：重新添加 Casbin 规则
		_ = s.casbinAdapter.AddGroupingPolicy(ctx, groupingRule)
		return errors.Wrap(err, "删除赋权记录失败")
	}

	return nil
}

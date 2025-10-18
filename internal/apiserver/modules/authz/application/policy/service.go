// Package policy 策略应用服务
package policy

import (
	"context"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/policy"
	policyDriven "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/policy/port/driven"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/resource"
	resourceDriven "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/resource/port/driven"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/role"
	roleDriven "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/role/port/driven"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	"github.com/fangcun-mount/iam-contracts/pkg/errors"
	"github.com/fangcun-mount/iam-contracts/pkg/log"
)

// Service 策略应用服务
type Service struct {
	policyVersionRepo policyDriven.PolicyVersionRepo
	roleRepo          roleDriven.RoleRepo
	resourceRepo      resourceDriven.ResourceRepo
	casbinPort        policyDriven.CasbinPort
	versionNotifier   policyDriven.VersionNotifier
}

// NewService 创建策略应用服务
func NewService(
	policyVersionRepo policyDriven.PolicyVersionRepo,
	roleRepo roleDriven.RoleRepo,
	resourceRepo resourceDriven.ResourceRepo,
	casbinPort policyDriven.CasbinPort,
	versionNotifier policyDriven.VersionNotifier,
) *Service {
	return &Service{
		policyVersionRepo: policyVersionRepo,
		roleRepo:          roleRepo,
		resourceRepo:      resourceRepo,
		casbinPort:        casbinPort,
		versionNotifier:   versionNotifier,
	}
}

// AddPolicyRuleCommand 添加策略规则命令
type AddPolicyRuleCommand struct {
	RoleID     uint64
	ResourceID resource.ResourceID
	Action     string
	TenantID   string
	ChangedBy  string
	Reason     string
}

// AddPolicyRule 添加策略规则
func (s *Service) AddPolicyRule(ctx context.Context, cmd AddPolicyRuleCommand) error {
	// 1. 验证参数
	if err := s.validateAddPolicyCommand(cmd); err != nil {
		return err
	}

	// 2. 检查角色是否存在
	roleExists, err := s.roleRepo.FindByID(ctx, role.NewRoleID(cmd.RoleID))
	if err != nil {
		if errors.IsCode(err, code.ErrRoleNotFound) {
			return errors.WithCode(code.ErrRoleNotFound, "角色 %d 不存在", cmd.RoleID)
		}
		return errors.Wrap(err, "获取角色失败")
	}

	// 3. 检查租户隔离
	if roleExists.TenantID != cmd.TenantID {
		return errors.WithCode(code.ErrPermissionDenied, "无权操作其他租户的角色")
	}

	// 4. 检查资源是否存在
	resourceExists, err := s.resourceRepo.FindByID(ctx, cmd.ResourceID)
	if err != nil {
		if errors.IsCode(err, code.ErrResourceNotFound) {
			return errors.WithCode(code.ErrResourceNotFound, "资源 %d 不存在", cmd.ResourceID.Uint64())
		}
		return errors.Wrap(err, "获取资源失败")
	}

	// 5. 验证 Action 是否合法
	valid, err := s.resourceRepo.ValidateAction(ctx, resourceExists.Key, cmd.Action)
	if err != nil {
		return errors.Wrap(err, "验证 Action 失败")
	}
	if !valid {
		return errors.WithCode(code.ErrInvalidAction, "Action %s 不被资源 %s 支持", cmd.Action, resourceExists.Key)
	}

	// 6. 构建策略规则
	policyRule := policy.PolicyRule{
		Sub: roleExists.Key(),
		Dom: cmd.TenantID,
		Obj: resourceExists.Key,
		Act: cmd.Action,
	}

	// 7. 添加到 Casbin
	if err := s.casbinPort.AddPolicy(ctx, policyRule); err != nil {
		return errors.Wrap(err, "添加策略规则失败")
	}

	// 8. 递增版本号
	newVersion, err := s.policyVersionRepo.Increment(ctx, cmd.TenantID, cmd.ChangedBy, cmd.Reason)
	if err != nil {
		log.Errorf("递增策略版本失败: %v", err)
		// 不阻塞主流程，只记录日志
	}

	// 9. 发布版本变更通知
	if newVersion != nil {
		if err := s.versionNotifier.Publish(ctx, cmd.TenantID, newVersion.Version); err != nil {
			log.Errorf("发布版本变更通知失败: %v", err)
			// 不阻塞主流程，只记录日志
		}
	}

	return nil
}

// RemovePolicyRuleCommand 移除策略规则命令
type RemovePolicyRuleCommand struct {
	RoleID     uint64
	ResourceID resource.ResourceID
	Action     string
	TenantID   string
	ChangedBy  string
	Reason     string
}

// RemovePolicyRule 移除策略规则
func (s *Service) RemovePolicyRule(ctx context.Context, cmd RemovePolicyRuleCommand) error {
	// 1. 验证参数
	if err := s.validateRemovePolicyCommand(cmd); err != nil {
		return err
	}

	// 2. 检查角色是否存在
	roleExists, err := s.roleRepo.FindByID(ctx, role.NewRoleID(cmd.RoleID))
	if err != nil {
		if errors.IsCode(err, code.ErrRoleNotFound) {
			return errors.WithCode(code.ErrRoleNotFound, "角色 %d 不存在", cmd.RoleID)
		}
		return errors.Wrap(err, "获取角色失败")
	}

	// 3. 检查租户隔离
	if roleExists.TenantID != cmd.TenantID {
		return errors.WithCode(code.ErrPermissionDenied, "无权操作其他租户的角色")
	}

	// 4. 检查资源是否存在
	resourceExists, err := s.resourceRepo.FindByID(ctx, cmd.ResourceID)
	if err != nil {
		if errors.IsCode(err, code.ErrResourceNotFound) {
			return errors.WithCode(code.ErrResourceNotFound, "资源 %d 不存在", cmd.ResourceID.Uint64())
		}
		return errors.Wrap(err, "获取资源失败")
	}

	// 5. 构建策略规则
	policyRule := policy.PolicyRule{
		Sub: roleExists.Key(),
		Dom: cmd.TenantID,
		Obj: resourceExists.Key,
		Act: cmd.Action,
	}

	// 6. 从 Casbin 移除
	if err := s.casbinPort.RemovePolicy(ctx, policyRule); err != nil {
		return errors.Wrap(err, "移除策略规则失败")
	}

	// 7. 递增版本号
	newVersion, err := s.policyVersionRepo.Increment(ctx, cmd.TenantID, cmd.ChangedBy, cmd.Reason)
	if err != nil {
		log.Errorf("递增策略版本失败: %v", err)
		// 不阻塞主流程，只记录日志
	}

	// 8. 发布版本变更通知
	if newVersion != nil {
		if err := s.versionNotifier.Publish(ctx, cmd.TenantID, newVersion.Version); err != nil {
			log.Errorf("发布版本变更通知失败: %v", err)
			// 不阻塞主流程，只记录日志
		}
	}

	return nil
}

// GetPoliciesByRoleQuery 获取角色策略查询
type GetPoliciesByRoleQuery struct {
	RoleID   uint64
	TenantID string
}

// GetPoliciesByRole 获取角色的所有策略规则
func (s *Service) GetPoliciesByRole(ctx context.Context, query GetPoliciesByRoleQuery) ([]policy.PolicyRule, error) {
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

	// 4. 从 Casbin 获取策略规则
	policies, err := s.casbinPort.GetPoliciesByRole(ctx, roleExists.Key(), query.TenantID)
	if err != nil {
		return nil, errors.Wrap(err, "获取策略规则失败")
	}

	return policies, nil
}

// GetCurrentVersionQuery 获取当前版本查询
type GetCurrentVersionQuery struct {
	TenantID string
}

// GetCurrentVersion 获取当前策略版本
func (s *Service) GetCurrentVersion(ctx context.Context, query GetCurrentVersionQuery) (*policy.PolicyVersion, error) {
	// 1. 验证参数
	if query.TenantID == "" {
		return nil, errors.WithCode(code.ErrInvalidArgument, "租户ID不能为空")
	}

	// 2. 获取当前版本
	version, err := s.policyVersionRepo.GetCurrent(ctx, query.TenantID)
	if err != nil {
		if errors.IsCode(err, code.ErrPolicyVersionNotFound) {
			// 如果没有版本记录，创建初始版本
			version, err = s.policyVersionRepo.GetOrCreate(ctx, query.TenantID)
			if err != nil {
				return nil, errors.Wrap(err, "创建初始版本失败")
			}
		} else {
			return nil, errors.Wrap(err, "获取当前版本失败")
		}
	}

	return version, nil
}

// validateAddPolicyCommand 验证添加策略命令
func (s *Service) validateAddPolicyCommand(cmd AddPolicyRuleCommand) error {
	if cmd.RoleID == 0 {
		return errors.WithCode(code.ErrInvalidArgument, "角色ID不能为空")
	}
	if cmd.ResourceID.Uint64() == 0 {
		return errors.WithCode(code.ErrInvalidArgument, "资源ID不能为空")
	}
	if cmd.Action == "" {
		return errors.WithCode(code.ErrInvalidArgument, "Action 不能为空")
	}
	if cmd.TenantID == "" {
		return errors.WithCode(code.ErrInvalidArgument, "租户ID不能为空")
	}
	if cmd.ChangedBy == "" {
		return errors.WithCode(code.ErrInvalidArgument, "变更人不能为空")
	}
	return nil
}

// validateRemovePolicyCommand 验证移除策略命令
func (s *Service) validateRemovePolicyCommand(cmd RemovePolicyRuleCommand) error {
	if cmd.RoleID == 0 {
		return errors.WithCode(code.ErrInvalidArgument, "角色ID不能为空")
	}
	if cmd.ResourceID.Uint64() == 0 {
		return errors.WithCode(code.ErrInvalidArgument, "资源ID不能为空")
	}
	if cmd.Action == "" {
		return errors.WithCode(code.ErrInvalidArgument, "Action 不能为空")
	}
	if cmd.TenantID == "" {
		return errors.WithCode(code.ErrInvalidArgument, "租户ID不能为空")
	}
	if cmd.ChangedBy == "" {
		return errors.WithCode(code.ErrInvalidArgument, "变更人不能为空")
	}
	return nil
}

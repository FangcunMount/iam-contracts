// Package policy 策略命令应用服务
package policy

import (
	"context"

	"github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/component-base/pkg/log"
	policyDriven "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/domain/policy/port/driven"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/domain/policy/port/driving"
	policyService "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/domain/policy/service"
)

// PolicyCommandService 策略命令服务（实现 PolicyCommander 接口）
// 负责协调领域服务、仓储和外部端口，处理策略的写操作
type PolicyCommandService struct {
	policyManager     *policyService.PolicyManager
	policyVersionRepo policyDriven.PolicyVersionRepo
	casbinPort        policyDriven.CasbinPort
	versionNotifier   policyDriven.VersionNotifier
}

// NewPolicyCommandService 创建策略命令服务
func NewPolicyCommandService(
	policyManager *policyService.PolicyManager,
	policyVersionRepo policyDriven.PolicyVersionRepo,
	casbinPort policyDriven.CasbinPort,
	versionNotifier policyDriven.VersionNotifier,
) *PolicyCommandService {
	return &PolicyCommandService{
		policyManager:     policyManager,
		policyVersionRepo: policyVersionRepo,
		casbinPort:        casbinPort,
		versionNotifier:   versionNotifier,
	}
}

// AddPolicyRule 添加策略规则
func (s *PolicyCommandService) AddPolicyRule(ctx context.Context, cmd driving.AddPolicyRuleCommand) error {
	// 1. 验证参数
	if err := s.policyManager.ValidateAddPolicyParameters(
		cmd.RoleID, cmd.ResourceID, cmd.Action, cmd.TenantID, cmd.ChangedBy,
	); err != nil {
		return err
	}

	// 2. 检查角色是否存在并验证租户隔离
	roleExists, err := s.policyManager.CheckRoleExistsAndTenant(ctx, cmd.RoleID, cmd.TenantID)
	if err != nil {
		return err
	}

	// 3. 检查资源是否存在并验证 Action 合法性
	resourceExists, err := s.policyManager.CheckResourceExistsAndValidateAction(ctx, cmd.ResourceID, cmd.Action)
	if err != nil {
		return err
	}

	// 4. 构建策略规则
	policyRule := driving.BuildPolicyRule(
		roleExists.Key(),
		cmd.TenantID,
		resourceExists.Key,
		cmd.Action,
	)

	// 5. 添加到 Casbin
	if err := s.casbinPort.AddPolicy(ctx, policyRule); err != nil {
		return errors.Wrap(err, "添加策略规则失败")
	}

	// 6. 递增版本号
	if err := s.incrementVersionAndNotify(ctx, cmd.TenantID, cmd.ChangedBy, cmd.Reason); err != nil {
		// 版本变更失败不阻塞主流程，只记录日志
		log.Errorf("策略版本变更失败: %v", err)
	}

	return nil
}

// RemovePolicyRule 移除策略规则
func (s *PolicyCommandService) RemovePolicyRule(ctx context.Context, cmd driving.RemovePolicyRuleCommand) error {
	// 1. 验证参数
	if err := s.policyManager.ValidateRemovePolicyParameters(
		cmd.RoleID, cmd.ResourceID, cmd.Action, cmd.TenantID, cmd.ChangedBy,
	); err != nil {
		return err
	}

	// 2. 检查角色是否存在并验证租户隔离
	roleExists, err := s.policyManager.CheckRoleExistsAndTenant(ctx, cmd.RoleID, cmd.TenantID)
	if err != nil {
		return err
	}

	// 3. 检查资源是否存在
	resourceExists, err := s.policyManager.CheckResourceExistsAndValidateAction(ctx, cmd.ResourceID, cmd.Action)
	if err != nil {
		return err
	}

	// 4. 构建策略规则
	policyRule := driving.BuildPolicyRule(
		roleExists.Key(),
		cmd.TenantID,
		resourceExists.Key,
		cmd.Action,
	)

	// 5. 从 Casbin 移除
	if err := s.casbinPort.RemovePolicy(ctx, policyRule); err != nil {
		return errors.Wrap(err, "移除策略规则失败")
	}

	// 6. 递增版本号
	if err := s.incrementVersionAndNotify(ctx, cmd.TenantID, cmd.ChangedBy, cmd.Reason); err != nil {
		// 版本变更失败不阻塞主流程，只记录日志
		log.Errorf("策略版本变更失败: %v", err)
	}

	return nil
}

// incrementVersionAndNotify 递增版本号并发布通知（内部辅助方法）
func (s *PolicyCommandService) incrementVersionAndNotify(
	ctx context.Context,
	tenantID string,
	changedBy string,
	reason string,
) error {
	// 1. 递增版本号
	newVersion, err := s.policyVersionRepo.Increment(ctx, tenantID, changedBy, reason)
	if err != nil {
		return errors.Wrap(err, "递增策略版本失败")
	}

	// 2. 发布版本变更通知
	if newVersion != nil {
		if err := s.versionNotifier.Publish(ctx, tenantID, newVersion.Version); err != nil {
			log.Errorf("发布版本变更通知失败: %v", err)
			// 通知失败不返回错误，只记录日志
		}
	}

	return nil
}

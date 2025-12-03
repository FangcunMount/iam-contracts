package policy

import (
	"context"

	policyDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authz/policy"
	resourceDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authz/resource"
	roleDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authz/role"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

type PolicyCommandService struct {
	policyValidator policyDomain.Validator
	policyRepo      policyDomain.Repository
	casbinAdapter   policyDomain.CasbinAdapter
	versionNotifier policyDomain.VersionNotifier
	roleRepo        roleDomain.Repository
	resourceRepo    resourceDomain.Repository
}

func NewPolicyCommandService(
	policyValidator policyDomain.Validator,
	policyRepo policyDomain.Repository,
	casbinAdapter policyDomain.CasbinAdapter,
	versionNotifier policyDomain.VersionNotifier,
	roleRepo roleDomain.Repository,
	resourceRepo resourceDomain.Repository,
) *PolicyCommandService {
	return &PolicyCommandService{
		policyValidator: policyValidator,
		policyRepo:      policyRepo,
		casbinAdapter:   casbinAdapter,
		versionNotifier: versionNotifier,
		roleRepo:        roleRepo,
		resourceRepo:    resourceRepo,
	}
}

func (s *PolicyCommandService) AddPolicyRule(
	ctx context.Context,
	cmd policyDomain.AddPolicyRuleCommand,
) error {
	// 1. 获取角色和资源信息
	role, err := s.roleRepo.FindByID(ctx, meta.FromUint64(cmd.RoleID))
	if err != nil {
		return err
	}

	resource, err := s.resourceRepo.FindByID(ctx, cmd.ResourceID)
	if err != nil {
		return err
	}

	// 2. 构建策略规则（resource.Key 是字段，不是方法）
	rule := policyDomain.BuildPolicyRule(role.Key(), cmd.TenantID, resource.Key, cmd.Action)

	// 3. 添加到 Casbin
	if err := s.casbinAdapter.AddPolicy(ctx, rule); err != nil {
		return err
	}

	// 4. 递增版本
	version, err := s.policyRepo.Increment(ctx, cmd.TenantID, cmd.ChangedBy, cmd.Reason)
	if err != nil {
		// 回滚 Casbin
		_ = s.casbinAdapter.RemovePolicy(ctx, rule)
		return err
	}

	// 5. 发送版本变更通知（如果启用了消息队列）
	if s.versionNotifier != nil {
		if err := s.versionNotifier.Publish(ctx, cmd.TenantID, version.Version); err != nil {
			// 日志记录错误，但不影响主流程
			// TODO: 添加日志
		}
	}

	return nil
}

func (s *PolicyCommandService) RemovePolicyRule(
	ctx context.Context,
	cmd policyDomain.RemovePolicyRuleCommand,
) error {
	// 1. 获取角色和资源信息
	role, err := s.roleRepo.FindByID(ctx, meta.FromUint64(cmd.RoleID))
	if err != nil {
		return err
	}

	resource, err := s.resourceRepo.FindByID(ctx, cmd.ResourceID)
	if err != nil {
		return err
	}

	// 2. 构建策略规则（resource.Key 是字段，不是方法）
	rule := policyDomain.BuildPolicyRule(role.Key(), cmd.TenantID, resource.Key, cmd.Action)

	// 3. 从 Casbin 移除
	if err := s.casbinAdapter.RemovePolicy(ctx, rule); err != nil {
		return err
	}

	// 4. 递增版本
	version, err := s.policyRepo.Increment(ctx, cmd.TenantID, cmd.ChangedBy, cmd.Reason)
	if err != nil {
		// 回滚 Casbin
		_ = s.casbinAdapter.AddPolicy(ctx, rule)
		return err
	}

	// 5. 发送版本变更通知（如果启用了消息队列）
	if s.versionNotifier != nil {
		if err := s.versionNotifier.Publish(ctx, cmd.TenantID, version.Version); err != nil {
			// 日志记录错误，但不影响主流程
			// TODO: 添加日志
		}
	}

	return nil
}

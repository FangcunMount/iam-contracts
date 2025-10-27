// Package policy 策略查询应用服务
package policy

import (
	"context"

	"github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/domain/policy"
	policyDriven "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/domain/policy/port/driven"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/domain/policy/port/driving"
	policyService "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/domain/policy/service"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
)

// PolicyQueryService 策略查询服务（实现 PolicyQueryer 接口）
// 负责策略的读操作，遵循 CQRS 原则
type PolicyQueryService struct {
	policyManager     *policyService.PolicyManager
	policyVersionRepo policyDriven.PolicyVersionRepo
	casbinPort        policyDriven.CasbinPort
}

// NewPolicyQueryService 创建策略查询服务
func NewPolicyQueryService(
	policyManager *policyService.PolicyManager,
	policyVersionRepo policyDriven.PolicyVersionRepo,
	casbinPort policyDriven.CasbinPort,
) *PolicyQueryService {
	return &PolicyQueryService{
		policyManager:     policyManager,
		policyVersionRepo: policyVersionRepo,
		casbinPort:        casbinPort,
	}
}

// GetPoliciesByRole 获取角色的所有策略规则
func (s *PolicyQueryService) GetPoliciesByRole(ctx context.Context, query driving.GetPoliciesByRoleQuery) ([]policy.PolicyRule, error) {
	// 1. 验证参数
	if err := s.policyManager.ValidateGetPoliciesQuery(query.RoleID, query.TenantID); err != nil {
		return nil, err
	}

	// 2. 检查角色是否存在并验证租户隔离
	roleExists, err := s.policyManager.CheckRoleExistsAndTenant(ctx, query.RoleID, query.TenantID)
	if err != nil {
		return nil, err
	}

	// 3. 从 Casbin 获取策略规则
	policies, err := s.casbinPort.GetPoliciesByRole(ctx, roleExists.Key(), query.TenantID)
	if err != nil {
		return nil, errors.Wrap(err, "获取策略规则失败")
	}

	return policies, nil
}

// GetCurrentVersion 获取当前策略版本
func (s *PolicyQueryService) GetCurrentVersion(ctx context.Context, query driving.GetCurrentVersionQuery) (*policy.PolicyVersion, error) {
	// 1. 验证参数
	if err := s.policyManager.ValidateGetVersionQuery(query.TenantID); err != nil {
		return nil, err
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

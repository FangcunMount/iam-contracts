// Package version 版本应用服务
package version

import (
	"context"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/policy"
	policyDriven "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/policy/port/driven"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	"github.com/fangcun-mount/iam-contracts/pkg/errors"
)

// Service 版本应用服务
type Service struct {
	policyVersionRepo policyDriven.PolicyVersionRepo
}

// NewService 创建版本应用服务
func NewService(policyVersionRepo policyDriven.PolicyVersionRepo) *Service {
	return &Service{
		policyVersionRepo: policyVersionRepo,
	}
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

// GetOrCreateVersionQuery 获取或创建版本查询
type GetOrCreateVersionQuery struct {
	TenantID string
}

// GetOrCreateVersion 获取或创建策略版本
func (s *Service) GetOrCreateVersion(ctx context.Context, query GetOrCreateVersionQuery) (*policy.PolicyVersion, error) {
	// 1. 验证参数
	if query.TenantID == "" {
		return nil, errors.WithCode(code.ErrInvalidArgument, "租户ID不能为空")
	}

	// 2. 获取或创建版本
	version, err := s.policyVersionRepo.GetOrCreate(ctx, query.TenantID)
	if err != nil {
		return nil, errors.Wrap(err, "获取或创建版本失败")
	}

	return version, nil
}

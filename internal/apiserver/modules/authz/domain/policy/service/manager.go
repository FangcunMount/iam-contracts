// Package service 策略领域服务
package service

import (
	"context"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/domain/resource"
	resourceDriven "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/domain/resource/port/driven"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/domain/role"
	roleDriven "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/domain/role/port/driven"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	"github.com/FangcunMount/iam-contracts/pkg/errors"
)

// PolicyManager 策略管理器（领域服务）
// 封装策略相关的业务规则，包括：
// 1. 策略参数验证
// 2. 角色和资源的存在性检查
// 3. 租户隔离检查
// 4. Action 合法性验证
type PolicyManager struct {
	roleRepo     roleDriven.RoleRepo
	resourceRepo resourceDriven.ResourceRepo
}

// NewPolicyManager 创建策略管理器
func NewPolicyManager(
	roleRepo roleDriven.RoleRepo,
	resourceRepo resourceDriven.ResourceRepo,
) *PolicyManager {
	return &PolicyManager{
		roleRepo:     roleRepo,
		resourceRepo: resourceRepo,
	}
}

// ValidateAddPolicyParameters 验证添加策略参数
func (m *PolicyManager) ValidateAddPolicyParameters(
	roleID uint64,
	resourceID resource.ResourceID,
	action string,
	tenantID string,
	changedBy string,
) error {
	if roleID == 0 {
		return errors.WithCode(code.ErrInvalidArgument, "角色ID不能为空")
	}
	if resourceID.Uint64() == 0 {
		return errors.WithCode(code.ErrInvalidArgument, "资源ID不能为空")
	}
	if action == "" {
		return errors.WithCode(code.ErrInvalidArgument, "Action 不能为空")
	}
	if tenantID == "" {
		return errors.WithCode(code.ErrInvalidArgument, "租户ID不能为空")
	}
	if changedBy == "" {
		return errors.WithCode(code.ErrInvalidArgument, "变更人不能为空")
	}
	return nil
}

// ValidateRemovePolicyParameters 验证移除策略参数
func (m *PolicyManager) ValidateRemovePolicyParameters(
	roleID uint64,
	resourceID resource.ResourceID,
	action string,
	tenantID string,
	changedBy string,
) error {
	// 移除策略的参数验证与添加策略相同
	return m.ValidateAddPolicyParameters(roleID, resourceID, action, tenantID, changedBy)
}

// CheckRoleExistsAndTenant 检查角色是否存在并验证租户隔离
// 返回角色实体用于后续操作
func (m *PolicyManager) CheckRoleExistsAndTenant(
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

// CheckResourceExistsAndValidateAction 检查资源是否存在并验证 Action 合法性
// 返回资源实体用于后续操作
func (m *PolicyManager) CheckResourceExistsAndValidateAction(
	ctx context.Context,
	resourceID resource.ResourceID,
	action string,
) (*resource.Resource, error) {
	resourceExists, err := m.resourceRepo.FindByID(ctx, resourceID)
	if err != nil {
		if errors.IsCode(err, code.ErrResourceNotFound) {
			return nil, errors.WithCode(code.ErrResourceNotFound, "资源 %d 不存在", resourceID.Uint64())
		}
		return nil, errors.Wrap(err, "获取资源失败")
	}

	// 验证 Action 是否合法
	valid, err := m.resourceRepo.ValidateAction(ctx, resourceExists.Key, action)
	if err != nil {
		return nil, errors.Wrap(err, "验证 Action 失败")
	}
	if !valid {
		return nil, errors.WithCode(code.ErrInvalidAction, "Action %s 不被资源 %s 支持", action, resourceExists.Key)
	}

	return resourceExists, nil
}

// ValidateGetPoliciesQuery 验证获取策略查询参数
func (m *PolicyManager) ValidateGetPoliciesQuery(roleID uint64, tenantID string) error {
	if roleID == 0 {
		return errors.WithCode(code.ErrInvalidArgument, "角色ID不能为空")
	}
	if tenantID == "" {
		return errors.WithCode(code.ErrInvalidArgument, "租户ID不能为空")
	}
	return nil
}

// ValidateGetVersionQuery 验证获取版本查询参数
func (m *PolicyManager) ValidateGetVersionQuery(tenantID string) error {
	if tenantID == "" {
		return errors.WithCode(code.ErrInvalidArgument, "租户ID不能为空")
	}
	return nil
}

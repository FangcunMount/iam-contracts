// Package resource 资源应用服务
//
// 本包提供资源管理的应用服务，实现 domain/port/driving 接口。
// 应用服务负责编排领域服务和仓储，管理事务边界。
package resource

import (
	"context"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/resource"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/resource/port/driven"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/resource/port/driving"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/resource/service"
)

// ResourceCommandService 资源命令服务（写操作）
//
// 实现 driving.ResourceCommander 接口
// 职责：
// - 编排领域服务和仓储完成资源的创建、更新、删除
// - 管理事务边界
// - 协调不同领域对象之间的交互
type ResourceCommandService struct {
	resourceManager *service.ResourceManager // 领域服务（封装业务规则）
	resourceRepo    driven.ResourceRepo      // 仓储接口（数据持久化）
}

// NewResourceCommandService 创建资源命令服务
func NewResourceCommandService(
	resourceManager *service.ResourceManager,
	resourceRepo driven.ResourceRepo,
) driving.ResourceCommander {
	return &ResourceCommandService{
		resourceManager: resourceManager,
		resourceRepo:    resourceRepo,
	}
}

// CreateResource 创建资源
//
// 实现 driving.ResourceCommander.CreateResource
// 编排流程：
// 1. 调用领域服务进行参数验证
// 2. 调用领域服务检查唯一性
// 3. 创建领域对象
// 4. 持久化到仓储
func (s *ResourceCommandService) CreateResource(
	ctx context.Context,
	cmd driving.CreateResourceCommand,
) (*resource.Resource, error) {
	// 1. 调用领域服务验证参数
	if err := s.resourceManager.ValidateCreateParameters(
		cmd.Key,
		cmd.DisplayName,
		cmd.AppName,
		cmd.Domain,
		cmd.Type,
		cmd.Actions,
	); err != nil {
		return nil, err
	}

	// 2. 调用领域服务检查键的唯一性（业务规则）
	if err := s.resourceManager.CheckKeyUniqueness(ctx, cmd.Key); err != nil {
		return nil, err
	}

	// 3. 创建资源领域对象
	newResource := resource.NewResource(
		cmd.Key,
		cmd.Actions,
		resource.WithDisplayName(cmd.DisplayName),
		resource.WithAppName(cmd.AppName),
		resource.WithDomain(cmd.Domain),
		resource.WithType(cmd.Type),
		resource.WithDescription(cmd.Description),
	)

	// 4. 持久化到仓储
	if err := s.resourceRepo.Create(ctx, &newResource); err != nil {
		return nil, err
	}

	return &newResource, nil
}

// UpdateResource 更新资源
//
// 实现 driving.ResourceCommander.UpdateResource
// 编排流程：
// 1. 调用领域服务检查资源是否存在
// 2. 调用领域服务验证更新参数
// 3. 更新领域对象属性
// 4. 持久化更新
func (s *ResourceCommandService) UpdateResource(
	ctx context.Context,
	cmd driving.UpdateResourceCommand,
) (*resource.Resource, error) {
	// 1. 调用领域服务检查资源是否存在
	existingResource, err := s.resourceManager.CheckResourceExists(ctx, cmd.ID)
	if err != nil {
		return nil, err
	}

	// 2. 调用领域服务验证更新参数
	if err := s.resourceManager.ValidateUpdateParameters(cmd.Actions); err != nil {
		return nil, err
	}

	// 3. 更新领域对象属性
	if cmd.DisplayName != nil {
		existingResource.DisplayName = *cmd.DisplayName
	}
	if len(cmd.Actions) > 0 {
		existingResource.Actions = cmd.Actions
	}
	if cmd.Description != nil {
		existingResource.Description = *cmd.Description
	}

	// 4. 持久化更新
	if err := s.resourceRepo.Update(ctx, existingResource); err != nil {
		return nil, err
	}

	return existingResource, nil
}

// DeleteResource 删除资源
//
// 实现 driving.ResourceCommander.DeleteResource
// 编排流程：
// 1. 调用领域服务检查资源是否存在
// 2. 删除资源
func (s *ResourceCommandService) DeleteResource(
	ctx context.Context,
	resourceID resource.ResourceID,
) error {
	// 1. 调用领域服务检查资源是否存在
	if _, err := s.resourceManager.CheckResourceExists(ctx, resourceID); err != nil {
		return err
	}

	// 2. 删除资源
	return s.resourceRepo.Delete(ctx, resourceID)
}

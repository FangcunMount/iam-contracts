// Package resource 资源应用服务
package resource

import (
	"context"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/resource"
	resourceDriven "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/resource/port/driven"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	"github.com/fangcun-mount/iam-contracts/pkg/errors"
)

// Service 资源应用服务
type Service struct {
	resourceRepo resourceDriven.ResourceRepo
}

// NewService 创建资源应用服务
func NewService(resourceRepo resourceDriven.ResourceRepo) *Service {
	return &Service{
		resourceRepo: resourceRepo,
	}
}

// CreateResourceCommand 创建资源命令
type CreateResourceCommand struct {
	Key         string
	DisplayName string
	AppName     string
	Domain      string
	Type        string
	Actions     []string
	Description string
}

// CreateResource 创建资源
func (s *Service) CreateResource(ctx context.Context, cmd CreateResourceCommand) (*resource.Resource, error) {
	// 1. 验证参数
	if err := s.validateCreateCommand(cmd); err != nil {
		return nil, err
	}

	// 2. 检查资源键是否已存在
	existingResource, err := s.resourceRepo.FindByKey(ctx, cmd.Key)
	if err != nil && !errors.IsCode(err, code.ErrResourceNotFound) {
		return nil, errors.Wrap(err, "检查资源键失败")
	}
	if existingResource != nil {
		return nil, errors.WithCode(code.ErrResourceAlreadyExists, "资源键 %s 已存在", cmd.Key)
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

	// 4. 保存到仓储
	if err := s.resourceRepo.Create(ctx, &newResource); err != nil {
		return nil, errors.Wrap(err, "创建资源失败")
	}

	return &newResource, nil
}

// UpdateResourceCommand 更新资源命令
type UpdateResourceCommand struct {
	ID          resource.ResourceID
	DisplayName string
	Actions     []string
	Description string
}

// UpdateResource 更新资源
func (s *Service) UpdateResource(ctx context.Context, cmd UpdateResourceCommand) (*resource.Resource, error) {
	// 1. 验证参数
	if cmd.ID.Uint64() == 0 {
		return nil, errors.WithCode(code.ErrInvalidArgument, "资源ID不能为空")
	}

	// 2. 获取现有资源
	existingResource, err := s.resourceRepo.FindByID(ctx, cmd.ID)
	if err != nil {
		if errors.IsCode(err, code.ErrResourceNotFound) {
			return nil, errors.WithCode(code.ErrResourceNotFound, "资源 %d 不存在", cmd.ID.Uint64())
		}
		return nil, errors.Wrap(err, "获取资源失败")
	}

	// 3. 更新字段
	if cmd.DisplayName != "" {
		existingResource.DisplayName = cmd.DisplayName
	}
	if len(cmd.Actions) > 0 {
		existingResource.Actions = cmd.Actions
	}
	existingResource.Description = cmd.Description

	// 4. 保存更新
	if err := s.resourceRepo.Update(ctx, existingResource); err != nil {
		return nil, errors.Wrap(err, "更新资源失败")
	}

	return existingResource, nil
}

// DeleteResource 删除资源
func (s *Service) DeleteResource(ctx context.Context, resourceID resource.ResourceID) error {
	// 1. 验证参数
	if resourceID.Uint64() == 0 {
		return errors.WithCode(code.ErrInvalidArgument, "资源ID不能为空")
	}

	// 2. 检查资源是否存在
	_, err := s.resourceRepo.FindByID(ctx, resourceID)
	if err != nil {
		if errors.IsCode(err, code.ErrResourceNotFound) {
			return errors.WithCode(code.ErrResourceNotFound, "资源 %d 不存在", resourceID.Uint64())
		}
		return errors.Wrap(err, "获取资源失败")
	}

	// 3. 删除资源
	if err := s.resourceRepo.Delete(ctx, resourceID); err != nil {
		return errors.Wrap(err, "删除资源失败")
	}

	return nil
}

// GetResourceByID 根据ID获取资源
func (s *Service) GetResourceByID(ctx context.Context, resourceID resource.ResourceID) (*resource.Resource, error) {
	// 1. 验证参数
	if resourceID.Uint64() == 0 {
		return nil, errors.WithCode(code.ErrInvalidArgument, "资源ID不能为空")
	}

	// 2. 获取资源
	foundResource, err := s.resourceRepo.FindByID(ctx, resourceID)
	if err != nil {
		if errors.IsCode(err, code.ErrResourceNotFound) {
			return nil, errors.WithCode(code.ErrResourceNotFound, "资源 %d 不存在", resourceID.Uint64())
		}
		return nil, errors.Wrap(err, "获取资源失败")
	}

	return foundResource, nil
}

// GetResourceByKey 根据键获取资源
func (s *Service) GetResourceByKey(ctx context.Context, key string) (*resource.Resource, error) {
	// 1. 验证参数
	if key == "" {
		return nil, errors.WithCode(code.ErrInvalidArgument, "资源键不能为空")
	}

	// 2. 获取资源
	foundResource, err := s.resourceRepo.FindByKey(ctx, key)
	if err != nil {
		if errors.IsCode(err, code.ErrResourceNotFound) {
			return nil, errors.WithCode(code.ErrResourceNotFound, "资源 %s 不存在", key)
		}
		return nil, errors.Wrap(err, "获取资源失败")
	}

	return foundResource, nil
}

// ListResourceQuery 列出资源查询
type ListResourceQuery struct {
	Offset int
	Limit  int
}

// ListResourceResult 列出资源结果
type ListResourceResult struct {
	Resources []*resource.Resource
	Total     int64
}

// ListResources 列出资源
func (s *Service) ListResources(ctx context.Context, query ListResourceQuery) (*ListResourceResult, error) {
	// 1. 设置默认值
	if query.Limit <= 0 {
		query.Limit = 20
	}
	if query.Offset < 0 {
		query.Offset = 0
	}

	// 2. 查询资源列表
	resources, total, err := s.resourceRepo.List(ctx, query.Offset, query.Limit)
	if err != nil {
		return nil, errors.Wrap(err, "列出资源失败")
	}

	return &ListResourceResult{
		Resources: resources,
		Total:     total,
	}, nil
}

// ValidateActionQuery 验证动作查询
type ValidateActionQuery struct {
	ResourceKey string
	Action      string
}

// ValidateAction 验证动作是否被资源支持
func (s *Service) ValidateAction(ctx context.Context, query ValidateActionQuery) (bool, error) {
	// 1. 验证参数
	if query.ResourceKey == "" {
		return false, errors.WithCode(code.ErrInvalidArgument, "资源键不能为空")
	}
	if query.Action == "" {
		return false, errors.WithCode(code.ErrInvalidArgument, "动作不能为空")
	}

	// 2. 验证动作
	valid, err := s.resourceRepo.ValidateAction(ctx, query.ResourceKey, query.Action)
	if err != nil {
		return false, errors.Wrap(err, "验证动作失败")
	}

	return valid, nil
}

// validateCreateCommand 验证创建命令
func (s *Service) validateCreateCommand(cmd CreateResourceCommand) error {
	if cmd.Key == "" {
		return errors.WithCode(code.ErrInvalidArgument, "资源键不能为空")
	}
	if cmd.DisplayName == "" {
		return errors.WithCode(code.ErrInvalidArgument, "显示名称不能为空")
	}
	if len(cmd.Actions) == 0 {
		return errors.WithCode(code.ErrInvalidArgument, "动作列表不能为空")
	}
	if cmd.AppName == "" {
		return errors.WithCode(code.ErrInvalidArgument, "应用名称不能为空")
	}
	if cmd.Domain == "" {
		return errors.WithCode(code.ErrInvalidArgument, "业务域不能为空")
	}
	if cmd.Type == "" {
		return errors.WithCode(code.ErrInvalidArgument, "对象类型不能为空")
	}
	return nil
}

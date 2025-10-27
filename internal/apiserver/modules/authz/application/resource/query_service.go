// Package resource 资源应用服务
package resource

import (
	"context"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/domain/resource"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/domain/resource/port/driven"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/domain/resource/port/driving"
	"github.com/FangcunMount/iam-contracts/pkg/errors"
)

// ResourceQueryService 资源查询服务（读操作）
//
// 实现 driving.ResourceQueryer 接口
// 职责：
// - 提供资源的各种查询操作
// - 支持分页、过滤等查询场景
// - 可以在这一层加缓存优化
type ResourceQueryService struct {
	resourceRepo driven.ResourceRepo // 仓储接口（数据读取）
}

// NewResourceQueryService 创建资源查询服务
func NewResourceQueryService(
	resourceRepo driven.ResourceRepo,
) driving.ResourceQueryer {
	return &ResourceQueryService{
		resourceRepo: resourceRepo,
	}
}

// GetResourceByID 根据ID获取资源
//
// 实现 driving.ResourceQueryer.GetResourceByID
func (s *ResourceQueryService) GetResourceByID(
	ctx context.Context,
	resourceID resource.ResourceID,
) (*resource.Resource, error) {
	return s.resourceRepo.FindByID(ctx, resourceID)
}

// GetResourceByKey 根据键获取资源
//
// 实现 driving.ResourceQueryer.GetResourceByKey
func (s *ResourceQueryService) GetResourceByKey(
	ctx context.Context,
	key string,
) (*resource.Resource, error) {
	return s.resourceRepo.FindByKey(ctx, key)
}

// ListResources 列出资源（支持分页）
//
// 实现 driving.ResourceQueryer.ListResources
func (s *ResourceQueryService) ListResources(
	ctx context.Context,
	query driving.ListResourcesQuery,
) (*driving.ListResourcesResult, error) {
	// 设置默认分页参数
	if query.Limit <= 0 {
		query.Limit = 20
	}
	if query.Offset < 0 {
		query.Offset = 0
	}

	// 查询资源列表
	resources, total, err := s.resourceRepo.List(ctx, query.Offset, query.Limit)
	if err != nil {
		return nil, errors.Wrap(err, "列出资源失败")
	}

	return &driving.ListResourcesResult{
		Resources: resources,
		Total:     total,
	}, nil
}

// ValidateAction 验证动作是否被资源支持
//
// 实现 driving.ResourceQueryer.ValidateAction
func (s *ResourceQueryService) ValidateAction(
	ctx context.Context,
	resourceKey, action string,
) (bool, error) {
	// 委托给仓储层验证（仓储层可能有缓存优化）
	return s.resourceRepo.ValidateAction(ctx, resourceKey, action)
}

// Package driving 资源领域驱动端口定义
package driving

import (
	"context"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authz/resource"
)

// ResourceQueryer 资源查询服务接口（读操作）
//
// 职责：
// - 处理资源的查询操作
// - 提供不同维度的资源检索
// - 支持资源动作验证
//
// 实现者：application/resource/ResourceQueryService
type ResourceQueryer interface {
	// GetResourceByID 根据ID获取资源
	GetResourceByID(ctx context.Context, resourceID resource.ResourceID) (*resource.Resource, error)

	// GetResourceByKey 根据键获取资源
	GetResourceByKey(ctx context.Context, key string) (*resource.Resource, error)

	// ListResources 列出资源（支持分页）
	ListResources(ctx context.Context, query ListResourcesQuery) (*ListResourcesResult, error)

	// ValidateAction 验证动作是否被资源支持
	ValidateAction(ctx context.Context, resourceKey, action string) (bool, error)
}

// ListResourcesQuery 列出资源查询参数
type ListResourcesQuery struct {
	// Offset 分页偏移量
	Offset int

	// Limit 分页限制（每页数量）
	Limit int
}

// ListResourcesResult 列出资源结果
type ListResourcesResult struct {
	// Resources 资源列表
	Resources []*resource.Resource

	// Total 总数量
	Total int64
}

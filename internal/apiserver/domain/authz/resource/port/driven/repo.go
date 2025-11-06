// Package driven 资源领域被驱动端口定义
package driven

import (
	"context"

	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authz/resource"
)

// ResourceRepo 资源目录仓储接口
type ResourceRepo interface {
	// Create 创建资源
	Create(ctx context.Context, resource *domain.Resource) error
	// Update 更新资源
	Update(ctx context.Context, resource *domain.Resource) error
	// Delete 删除资源
	Delete(ctx context.Context, id domain.ResourceID) error
	// FindByID 根据ID获取资源
	FindByID(ctx context.Context, id domain.ResourceID) (*domain.Resource, error)
	// FindByKey 根据键获取资源
	FindByKey(ctx context.Context, key string) (*domain.Resource, error)
	// List 列出所有资源
	List(ctx context.Context, offset, limit int) ([]*domain.Resource, int64, error)
	// ValidateAction 校验动作是否在资源的允许列表中
	ValidateAction(ctx context.Context, resourceKey, action string) (bool, error)
}

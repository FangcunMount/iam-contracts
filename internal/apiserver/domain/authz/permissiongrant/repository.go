package permissiongrant

import (
	"context"

	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/resource"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
)

// Repository 权限授予仓储接口
type Repository interface {
	// Create 创建权限授予实体
	Create(ctx context.Context, grant *Grant) error
	// AtomicRevoke 原子撤销权限授予实体
	AtomicRevoke(ctx context.Context, id meta.ID) (RevokeOutcome, error)
	// FindRoleID reads ownership for guarded revocation, including historical grants.
	FindRoleID(ctx context.Context, id meta.ID) (meta.ID, error)
	// FindByID 根据ID查找权限授予实体
	FindByID(ctx context.Context, id meta.ID) (*Grant, error)
	// ListByRole 根据角色ID查找权限授予实体列表
	ListByRole(ctx context.Context, roleID meta.ID) ([]*Grant, error)
	// ListActiveByResource 根据资源ID查找活动权限授予实体列表
	ListActiveByResource(ctx context.Context, resourceID resource.ResourceID) ([]*Grant, error)
	// ListActive 查找所有活动权限授予实体列表
	ListActive(ctx context.Context) ([]*Grant, error)
}

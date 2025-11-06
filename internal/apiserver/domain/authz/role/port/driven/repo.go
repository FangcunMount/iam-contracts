// Package driven 角色领域被驱动端口定义
package driven

import (
	"context"

	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authz/role"
)

// RoleRepo 角色仓储接口
type RoleRepo interface {
	// Create 创建角色
	Create(ctx context.Context, role *domain.Role) error
	// Update 更新角色
	Update(ctx context.Context, role *domain.Role) error
	// Delete 删除角色
	Delete(ctx context.Context, id domain.RoleID) error
	// FindByID 根据ID获取角色
	FindByID(ctx context.Context, id domain.RoleID) (*domain.Role, error)
	// FindByName 根据名称和租户获取角色
	FindByName(ctx context.Context, tenantID, name string) (*domain.Role, error)
	// List 列出角色
	List(ctx context.Context, tenantID string, offset, limit int) ([]*domain.Role, int64, error)
}

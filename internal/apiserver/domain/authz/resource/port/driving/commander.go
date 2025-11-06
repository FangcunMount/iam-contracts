// Package driving 资源领域驱动端口定义
//
// 本包定义了资源管理的用例接口（Driving Ports），遵循CQRS原则：
// - ResourceCommander: 处理写操作（命令端）
// - ResourceQueryer: 处理读操作（查询端）
//
// 这些接口由应用层实现，被接口层调用。
package driving

import (
	"context"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authz/resource"
)

// ResourceCommander 资源命令服务接口（写操作）
//
// 职责：
// - 处理资源的创建、更新、删除操作
// - 管理事务边界
// - 协调领域服务和仓储
//
// 实现者：application/resource/ResourceCommandService
type ResourceCommander interface {
	// CreateResource 创建资源
	CreateResource(ctx context.Context, cmd CreateResourceCommand) (*resource.Resource, error)

	// UpdateResource 更新资源
	UpdateResource(ctx context.Context, cmd UpdateResourceCommand) (*resource.Resource, error)

	// DeleteResource 删除资源
	DeleteResource(ctx context.Context, resourceID resource.ResourceID) error
}

// CreateResourceCommand 创建资源命令
type CreateResourceCommand struct {
	// Key 资源键，全局唯一标识符
	Key string

	// DisplayName 资源显示名称
	DisplayName string

	// AppName 所属应用名称
	AppName string

	// Domain 业务域
	Domain string

	// Type 资源对象类型
	Type string

	// Actions 资源支持的动作列表
	Actions []string

	// Description 资源描述
	Description string
}

// UpdateResourceCommand 更新资源命令
type UpdateResourceCommand struct {
	// ID 资源ID
	ID resource.ResourceID

	// DisplayName 更新的显示名称（可选）
	DisplayName *string

	// Actions 更新的动作列表（可选）
	Actions []string

	// Description 更新的描述（可选）
	Description *string
}

// Package resource 资源领域包
package resource

import (
	"context"
)

// Commander 资源命令服务接口（Driving Port - 写操作）
//
// 职责：
// - 处理资源的创建、更新、删除操作
// - 管理事务边界
// - 协调领域服务和仓储
type Commander interface {
	// CreateResource 创建资源
	CreateResource(ctx context.Context, cmd CreateResourceCommand) (*Resource, error)

	// UpdateResource 更新资源
	UpdateResource(ctx context.Context, cmd UpdateResourceCommand) (*Resource, error)

	// DeleteResource 删除资源
	DeleteResource(ctx context.Context, resourceID ResourceID) error
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
	ID ResourceID

	// DisplayName 更新的显示名称（可选）
	DisplayName *string

	// Actions 更新的动作列表（可选）
	Actions []string

	// Description 更新的描述（可选）
	Description *string
}

// Queryer 资源查询服务接口（Driving Port - 读操作）
//
// 职责：
// - 处理资源的查询操作
// - 提供不同维度的资源检索
// - 支持资源动作验证
type Queryer interface {
	// GetResourceByID 根据ID获取资源
	GetResourceByID(ctx context.Context, resourceID ResourceID) (*Resource, error)

	// GetResourceByKey 根据键获取资源
	GetResourceByKey(ctx context.Context, key string) (*Resource, error)

	// ListResources 列出资源（支持分页）
	ListResources(ctx context.Context, query ListResourcesQuery) (*ListResourcesResult, error)

	// ValidateAction 验证动作是否被资源支持
	ValidateAction(ctx context.Context, resourceKey, action string) (bool, error)
}

// ListResourcesQuery 列出资源查询参数
type ListResourcesQuery struct {
	// AppName 应用名称过滤
	AppName string

	// Domain 业务域过滤
	Domain string

	// Type 资源类型过滤
	Type string

	// Offset 分页偏移量
	Offset int

	// Limit 分页限制（每页数量）
	Limit int
}

// ListResourcesResult 列出资源结果
type ListResourcesResult struct {
	// Resources 资源列表
	Resources []*Resource

	// Total 总数量
	Total int64
}

// Validator 资源验证器接口（Driving Port - 领域服务）
// 封装资源相关的验证规则
type Validator interface {
	// ValidateCreateCommand 验证创建命令
	ValidateCreateCommand(cmd CreateResourceCommand) error

	// ValidateUpdateCommand 验证更新命令
	ValidateUpdateCommand(cmd UpdateResourceCommand) error

	// CheckKeyUnique 检查键唯一性
	CheckKeyUnique(ctx context.Context, key string) error
}

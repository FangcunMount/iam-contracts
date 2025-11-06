// Package assignment 赋权领域包
package assignment

import (
	"context"
)

// Commander 赋权命令接口（Driving Port - 写操作）
// 定义赋权管理的用例接口，遵循 CQRS 原则
type Commander interface {
	// Grant 授权（赋予角色）
	Grant(ctx context.Context, cmd GrantCommand) (*Assignment, error)

	// Revoke 撤销授权（移除角色）
	Revoke(ctx context.Context, cmd RevokeCommand) error

	// RevokeByID 根据ID撤销授权
	RevokeByID(ctx context.Context, cmd RevokeByIDCommand) error
}

// GrantCommand 授权命令
type GrantCommand struct {
	SubjectType SubjectType // 主体类型（user/service）
	SubjectID   string      // 主体ID
	RoleID      uint64      // 角色ID
	TenantID    string      // 租户ID
	GrantedBy   string      // 授权人
}

// RevokeCommand 撤销授权命令
type RevokeCommand struct {
	SubjectType SubjectType // 主体类型
	SubjectID   string      // 主体ID
	RoleID      uint64      // 角色ID
	TenantID    string      // 租户ID
}

// RevokeByIDCommand 根据ID撤销授权命令
type RevokeByIDCommand struct {
	AssignmentID AssignmentID // 赋权ID
	TenantID     string       // 租户ID
}

// Queryer 赋权查询接口（Driving Port - 读操作）
// 定义赋权查询的用例接口，遵循 CQRS 原则
type Queryer interface {
	// ListBySubject 根据主体列出赋权
	ListBySubject(ctx context.Context, query ListBySubjectQuery) ([]*Assignment, error)

	// ListByRole 根据角色列出赋权
	ListByRole(ctx context.Context, query ListByRoleQuery) ([]*Assignment, error)
}

// ListBySubjectQuery 根据主体列出赋权查询
type ListBySubjectQuery struct {
	SubjectType SubjectType // 主体类型
	SubjectID   string      // 主体ID
	TenantID    string      // 租户ID
}

// ListByRoleQuery 根据角色列出赋权查询
type ListByRoleQuery struct {
	RoleID   uint64 // 角色ID
	TenantID string // 租户ID
}

// Validator 赋权验证器接口（Driving Port - 领域服务）
// 封装赋权相关的验证规则
type Validator interface {
	// ValidateGrantCommand 验证授权命令
	ValidateGrantCommand(cmd GrantCommand) error

	// ValidateRevokeCommand 验证撤销命令
	ValidateRevokeCommand(cmd RevokeCommand) error

	// CheckRoleExists 检查角色是否存在
	CheckRoleExists(ctx context.Context, roleID uint64, tenantID string) error

	// CheckSubjectExists 检查主体是否存在
	CheckSubjectExists(ctx context.Context, subjectType SubjectType, subjectID, tenantID string) error
}

// Package driving 定义赋权模块的 Driving 端口（用例接口）
package driving

import (
	"context"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authz/assignment"
)

// AssignmentQueryer 赋权查询接口（读操作）
// 定义赋权查询的用例接口，遵循 CQRS 原则
type AssignmentQueryer interface {
	// ListBySubject 根据主体列出赋权
	ListBySubject(ctx context.Context, query ListBySubjectQuery) ([]*assignment.Assignment, error)

	// ListByRole 根据角色列出赋权
	ListByRole(ctx context.Context, query ListByRoleQuery) ([]*assignment.Assignment, error)
}

// ListBySubjectQuery 根据主体列出赋权查询
type ListBySubjectQuery struct {
	SubjectType assignment.SubjectType // 主体类型
	SubjectID   string                 // 主体ID
	TenantID    string                 // 租户ID
}

// ListByRoleQuery 根据角色列出赋权查询
type ListByRoleQuery struct {
	RoleID   uint64 // 角色ID
	TenantID string // 租户ID
}

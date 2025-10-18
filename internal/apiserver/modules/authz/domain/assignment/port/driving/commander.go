// Package driving 定义赋权模块的 Driving 端口（用例接口）
package driving

import (
	"context"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/assignment"
)

// AssignmentCommander 赋权命令接口（写操作）
// 定义赋权管理的用例接口，遵循 CQRS 原则
type AssignmentCommander interface {
	// Grant 授权（赋予角色）
	Grant(ctx context.Context, cmd GrantCommand) (*assignment.Assignment, error)

	// Revoke 撤销授权（移除角色）
	Revoke(ctx context.Context, cmd RevokeCommand) error

	// RevokeByID 根据ID撤销授权
	RevokeByID(ctx context.Context, cmd RevokeByIDCommand) error
}

// GrantCommand 授权命令
type GrantCommand struct {
	SubjectType assignment.SubjectType // 主体类型（user/service）
	SubjectID   string                 // 主体ID
	RoleID      uint64                 // 角色ID
	TenantID    string                 // 租户ID
	GrantedBy   string                 // 授权人
}

// RevokeCommand 撤销授权命令
type RevokeCommand struct {
	SubjectType assignment.SubjectType // 主体类型
	SubjectID   string                 // 主体ID
	RoleID      uint64                 // 角色ID
	TenantID    string                 // 租户ID
}

// RevokeByIDCommand 根据ID撤销授权命令
type RevokeByIDCommand struct {
	AssignmentID assignment.AssignmentID // 赋权ID
	TenantID     string                  // 租户ID
}

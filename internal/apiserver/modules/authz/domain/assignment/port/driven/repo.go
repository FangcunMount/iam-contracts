// Package driven 赋权领域被驱动端口定义
package driven

import (
	"context"

	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/domain/assignment"
)

// AssignmentRepo 赋权仓储接口
type AssignmentRepo interface {
	// Create 创建赋权
	Create(ctx context.Context, assignment *domain.Assignment) error
	// Delete 删除赋权（撤销）
	Delete(ctx context.Context, id domain.AssignmentID) error
	// DeleteBySubjectAndRole 根据主体和角色删除赋权
	DeleteBySubjectAndRole(ctx context.Context, subjectType domain.SubjectType, subjectID string, roleID uint64, tenantID string) error
	// FindByID 根据ID获取赋权
	FindByID(ctx context.Context, id domain.AssignmentID) (*domain.Assignment, error)
	// ListBySubject 根据主体列出赋权
	ListBySubject(ctx context.Context, subjectType domain.SubjectType, subjectID, tenantID string) ([]*domain.Assignment, error)
	// ListByRole 根据角色列出赋权
	ListByRole(ctx context.Context, roleID uint64, tenantID string) ([]*domain.Assignment, error)
}

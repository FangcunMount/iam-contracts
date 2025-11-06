package guardianship

import (
	"context"
)

// ============= 应用服务接口（Driving Ports）=============

// GuardianshipApplicationService 监护关系应用服务
type GuardianshipApplicationService interface {
	// AddGuardian 添加监护人
	AddGuardian(ctx context.Context, dto AddGuardianDTO) error
	// RemoveGuardian 移除监护人
	RemoveGuardian(ctx context.Context, dto RemoveGuardianDTO) error
}

// GuardianshipQueryApplicationService 监护关系查询应用服务（只读）
type GuardianshipQueryApplicationService interface {
	// IsGuardian 检查是否为监护人
	IsGuardian(ctx context.Context, userID string, childID string) (bool, error)
	// GetByUserIDAndChildID 查询监护关系
	GetByUserIDAndChildID(ctx context.Context, userID string, childID string) (*GuardianshipResult, error)
	// ListChildrenByUserID 列出用户监护的所有儿童
	ListChildrenByUserID(ctx context.Context, userID string) ([]*GuardianshipResult, error)
	// ListGuardiansByChildID 列出儿童的所有监护人
	ListGuardiansByChildID(ctx context.Context, childID string) ([]*GuardianshipResult, error)
}

// ============= DTOs =============

// AddGuardianDTO 添加监护人 DTO
type AddGuardianDTO struct {
	UserID   string // 用户 ID
	ChildID  string // 儿童 ID
	Relation string // 关系（parent/grandparents/other）
}

// RemoveGuardianDTO 移除监护人 DTO
type RemoveGuardianDTO struct {
	UserID  string // 用户 ID
	ChildID string // 儿童 ID
}

// GuardianshipResult 监护关系结果 DTO
type GuardianshipResult struct {
	ID            uint64 // 监护关系 ID
	UserID        string // 用户 ID
	ChildID       string // 儿童 ID
	Relation      string // 关系
	EstablishedAt string // 建立时间
	// 可选：包含儿童信息
	ChildName     string // 儿童姓名
	ChildGender   string // 儿童性别
	ChildBirthday string // 儿童生日
}

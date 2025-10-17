package user

import (
	"context"

	domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/user"
)

// ============= 应用服务接口（Driving Ports）=============

// UserApplicationService 用户应用服务 - 基本管理（命令）
type UserApplicationService interface {
	// Register 注册新用户
	Register(ctx context.Context, dto RegisterUserDTO) (*UserResult, error)
}

// UserProfileApplicationService 用户资料应用服务
type UserProfileApplicationService interface {
	// Rename 修改用户名称
	Rename(ctx context.Context, userID string, newName string) error
	// UpdateContact 更新联系方式
	UpdateContact(ctx context.Context, dto UpdateContactDTO) error
	// UpdateIDCard 更新身份证
	UpdateIDCard(ctx context.Context, userID string, idCard string) error
}

// UserStatusApplicationService 用户状态应用服务
type UserStatusApplicationService interface {
	// Activate 激活用户
	Activate(ctx context.Context, userID string) error
	// Deactivate 停用用户
	Deactivate(ctx context.Context, userID string) error
	// Block 封禁用户
	Block(ctx context.Context, userID string) error
}

// UserQueryApplicationService 用户查询应用服务（只读）
type UserQueryApplicationService interface {
	// GetByID 根据 ID 查询用户
	GetByID(ctx context.Context, userID string) (*UserResult, error)
	// GetByPhone 根据手机号查询用户
	GetByPhone(ctx context.Context, phone string) (*UserResult, error)
}

// ============= DTOs =============

// RegisterUserDTO 注册用户 DTO
type RegisterUserDTO struct {
	Name  string // 用户名
	Phone string // 手机号（必填）
	Email string // 邮箱（可选）
}

// UpdateContactDTO 更新联系方式 DTO
type UpdateContactDTO struct {
	UserID string // 用户 ID
	Phone  string // 手机号（可选）
	Email  string // 邮箱（可选）
}

// UserResult 用户结果 DTO
type UserResult struct {
	ID     string            // 用户 ID
	Name   string            // 用户名
	Phone  string            // 手机号
	Email  string            // 邮箱
	IDCard string            // 身份证号
	Status domain.UserStatus // 用户状态
}

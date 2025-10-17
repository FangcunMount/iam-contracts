package child

import (
	"context"
)

// ============= 应用服务接口（Driving Ports）=============

// ChildApplicationService 儿童应用服务 - 基本管理（命令）
type ChildApplicationService interface {
	// Register 注册新儿童档案
	Register(ctx context.Context, dto RegisterChildDTO) (*ChildResult, error)
}

// ChildProfileApplicationService 儿童资料应用服务
type ChildProfileApplicationService interface {
	// Rename 修改儿童姓名
	Rename(ctx context.Context, childID string, newName string) error
	// UpdateIDCard 更新身份证
	UpdateIDCard(ctx context.Context, childID string, name string, idCard string) error
	// UpdateProfile 更新基本信息（性别、生日）
	UpdateProfile(ctx context.Context, dto UpdateChildProfileDTO) error
	// UpdateHeightWeight 更新身高体重
	UpdateHeightWeight(ctx context.Context, dto UpdateHeightWeightDTO) error
}

// ChildQueryApplicationService 儿童查询应用服务（只读）
type ChildQueryApplicationService interface {
	// GetByID 根据 ID 查询儿童
	GetByID(ctx context.Context, childID string) (*ChildResult, error)
	// GetByIDCard 根据身份证查询儿童
	GetByIDCard(ctx context.Context, idCard string) (*ChildResult, error)
	// FindSimilar 查找相似儿童（姓名、性别、生日）
	FindSimilar(ctx context.Context, name string, gender string, birthday string) ([]*ChildResult, error)
}

// ============= DTOs =============

// RegisterChildDTO 注册儿童 DTO
type RegisterChildDTO struct {
	Name     string  // 姓名（必填）
	Gender   string  // 性别（可选：male/female）
	Birthday string  // 生日（格式：YYYY-MM-DD）
	IDCard   string  // 身份证号（可选）
	Height   *uint32 // 身高（厘米，可选）
	Weight   *uint32 // 体重（克，可选）
}

// UpdateChildProfileDTO 更新儿童资料 DTO
type UpdateChildProfileDTO struct {
	ChildID  string // 儿童 ID
	Gender   string // 性别
	Birthday string // 生日
}

// UpdateHeightWeightDTO 更新身高体重 DTO
type UpdateHeightWeightDTO struct {
	ChildID string // 儿童 ID
	Height  uint32 // 身高（厘米）
	Weight  uint32 // 体重（克）
}

// ChildResult 儿童结果 DTO
type ChildResult struct {
	ID       string // 儿童 ID
	Name     string // 姓名
	IDCard   string // 身份证号
	Gender   string // 性别
	Birthday string // 生日
	Height   uint32 // 身高（厘米）
	Weight   uint32 // 体重（克）
}

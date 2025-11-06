package user

import (
	"context"
)

// UserStatusChanger 用户状态管理领域服务
type userStatusChanger struct {
	repo Repository
}

// 确保 UserStatusChanger 实现了 UserStatusChanger 接口
var _ StatusChanger = (*userStatusChanger)(nil)

// NewStatusService 创建用户状态服务
func NewStatusService(repo Repository) *userStatusChanger {
	return &userStatusChanger{repo: repo}
}

// Activate 激活用户
// 领域逻辑：验证 + 修改实体状态
// 注意：不包括持久化，返回修改后的实体供应用层持久化
func (s *userStatusChanger) Activate(ctx context.Context, userID UserID) (*User, error) {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 领域逻辑：激活用户
	user.Activate()

	// 返回修改后的实体，由应用层持久化
	return user, nil
}

// Deactivate 停用用户
// 领域逻辑：验证 + 修改实体状态
// 注意：不包括持久化，返回修改后的实体供应用层持久化
func (s *userStatusChanger) Deactivate(ctx context.Context, userID UserID) (*User, error) {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 领域逻辑：停用用户
	user.Deactivate()

	// 返回修改后的实体，由应用层持久化
	return user, nil
}

// Block 封禁用户
// 领域逻辑：验证 + 修改实体状态
// 注意：不包括持久化，返回修改后的实体供应用层持久化
func (s *userStatusChanger) Block(ctx context.Context, userID UserID) (*User, error) {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 领域逻辑：封禁用户
	user.Block()

	// 返回修改后的实体，由应用层持久化
	return user, nil
}

package user

import (
	"context"
)

// statusChanger 用户状态管理领域服务
type statusChanger struct {
	repo Repository
}

// 确保 statusChanger 实现了 statusChanger 接口
var _ StatusChanger = (*statusChanger)(nil)

// NewStatusService 创建用户状态服务
func NewStatusService(repo Repository) *statusChanger {
	return &statusChanger{repo: repo}
}

// Activate 激活用户
// 领域逻辑：验证 + 修改实体状态
// 注意：不包括持久化，返回修改后的实体供应用层持久化
func (s *statusChanger) Activate(ctx context.Context, userID UserID) (*User, error) {
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
func (s *statusChanger) Deactivate(ctx context.Context, userID UserID) (*User, error) {
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
func (s *statusChanger) Block(ctx context.Context, userID UserID) (*User, error) {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 领域逻辑：封禁用户
	user.Block()

	// 返回修改后的实体，由应用层持久化
	return user, nil
}

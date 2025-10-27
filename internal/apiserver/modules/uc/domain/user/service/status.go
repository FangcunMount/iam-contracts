package service

import (
	"context"

	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/domain/user"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/domain/user/port"
)

// UserStatusChanger 用户状态管理领域服务
type UserStatusChanger struct {
	repo port.UserRepository
}

// 确保 UserStatusChanger 实现了 port.UserStatusChanger 接口
var _ port.UserStatusChanger = (*UserStatusChanger)(nil)

// NewStatusService 创建用户状态服务
func NewStatusService(repo port.UserRepository) *UserStatusChanger {
	return &UserStatusChanger{repo: repo}
}

// Activate 激活用户
// 领域逻辑：验证 + 修改实体状态
// 注意：不包括持久化，返回修改后的实体供应用层持久化
func (s *UserStatusChanger) Activate(ctx context.Context, userID domain.UserID) (*domain.User, error) {
	user, err := NewQueryService(s.repo).FindByID(ctx, userID)
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
func (s *UserStatusChanger) Deactivate(ctx context.Context, userID domain.UserID) (*domain.User, error) {
	user, err := NewQueryService(s.repo).FindByID(ctx, userID)
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
func (s *UserStatusChanger) Block(ctx context.Context, userID domain.UserID) (*domain.User, error) {
	user, err := NewQueryService(s.repo).FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 领域逻辑：封禁用户
	user.Block()

	// 返回修改后的实体，由应用层持久化
	return user, nil
}

package user

import (
	"context"

	domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/user"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/user/port"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	perrors "github.com/fangcun-mount/iam-contracts/pkg/errors"
)

// UserStatusChanger 用户状态变更应用服务
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
func (s *UserStatusChanger) Activate(ctx context.Context, userID domain.UserID) error {
	u, err := NewQueryService(s.repo).FindByID(ctx, userID)
	if err != nil {
		return err
	}

	u.Activate()

	if err := s.repo.Update(ctx, *u); err != nil {
		return perrors.WrapC(err, code.ErrDatabase, "activate user(%s) failed", userID.String())
	}

	return nil
}

// Deactivate 停用用户
func (s *UserStatusChanger) Deactivate(ctx context.Context, userID domain.UserID) error {
	u, err := NewQueryService(s.repo).FindByID(ctx, userID)
	if err != nil {
		return err
	}

	u.Deactivate()

	if err := s.repo.Update(ctx, *u); err != nil {
		return perrors.WrapC(err, code.ErrDatabase, "deactivate user(%s) failed", userID.String())
	}

	return nil
}

// Block 封禁用户
func (s *UserStatusChanger) Block(ctx context.Context, userID domain.UserID) error {
	u, err := NewQueryService(s.repo).FindByID(ctx, userID)
	if err != nil {
		return err
	}

	u.Block()

	if err := s.repo.Update(ctx, *u); err != nil {
		return perrors.WrapC(err, code.ErrDatabase, "block user(%s) failed", userID.String())
	}

	return nil
}

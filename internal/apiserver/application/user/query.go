package user

import (
	"context"
	"errors"

	domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/user"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/user/port"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/meta"
	perrors "github.com/fangcun-mount/iam-contracts/pkg/errors"
	"gorm.io/gorm"
)

// UserQueryer 用户查询应用服务
type UserQueryer struct {
	repo port.UserRepository
}

// 确保 UserQueryer 实现了 port.UserQueryer 接口
var _ port.UserQueryer = (*UserQueryer)(nil)

// NewQueryService 创建用户查询服务
func NewQueryService(repo port.UserRepository) *UserQueryer {
	return &UserQueryer{repo: repo}
}

// FindByID 根据用户ID查询用户
func (s *UserQueryer) FindByID(ctx context.Context, userID domain.UserID) (*domain.User, error) {
	u, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, perrors.WithCode(code.ErrUserNotFound, "user(%s) not found", userID.String())
		}
		return nil, perrors.WrapC(err, code.ErrDatabase, "find user by id(%s) failed", userID.String())
	}
	return &u, nil
}

// FindByPhone 根据手机号查询用户
func (s *UserQueryer) FindByPhone(ctx context.Context, phone meta.Phone) (*domain.User, error) {
	u, err := s.repo.FindByPhone(ctx, phone)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, perrors.WithCode(code.ErrUserNotFound, "user with phone(%s) not found", phone.String())
		}
		return nil, perrors.WrapC(err, code.ErrDatabase, "find user by phone(%s) failed", phone.String())
	}
	return &u, nil
}

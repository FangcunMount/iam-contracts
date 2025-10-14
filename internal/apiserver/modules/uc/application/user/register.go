package user

import (
	"context"

	domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/user"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/user/port"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/meta"
	perrors "github.com/fangcun-mount/iam-contracts/pkg/errors"
)

// UserRegister 用户注册应用服务
type UserRegister struct {
	repo port.UserRepository
}

// 确保 UserRegister 实现了 port.UserRegister 接口
var _ port.UserRegister = (*UserRegister)(nil)

// NewRegisterService 创建用户注册服务
func NewRegisterService(repo port.UserRepository) *UserRegister {
	return &UserRegister{repo: repo}
}

// Register 注册新用户
func (s *UserRegister) Register(ctx context.Context, name string, phone meta.Phone) (*domain.User, error) {
	// 确保手机号唯一
	if err := ensurePhoneUnique(ctx, s.repo, phone); err != nil {
		return nil, err
	}

	u, err := domain.NewUser(name, phone)
	if err != nil {
		return nil, err
	}

	if err := s.repo.Create(ctx, u); err != nil {
		return nil, perrors.WrapC(err, code.ErrDatabase, "create user(%s) failed", phone.String())
	}

	created, err := s.repo.FindByPhone(ctx, phone)
	if err != nil {
		return nil, perrors.WrapC(err, code.ErrDatabase, "load user(%s) after creation failed", phone.String())
	}

	return created, nil
}

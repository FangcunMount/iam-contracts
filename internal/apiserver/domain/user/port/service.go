package port

import (
	"context"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/user"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/meta"
)

// UserRegister 用户注册服务接口
type UserRegister interface {
	Register(ctx context.Context, name string, phone meta.Phone) (u *user.User, err error)
}

// UserStatusChanger 用户状态变更服务接口
type UserStatusChanger interface {
	Activate(ctx context.Context, userID user.UserID) error
	Deactivate(ctx context.Context, userID user.UserID) error
	Block(ctx context.Context, userID user.UserID) error
}

// UserProfileEditor 用户资料编辑服务接口
type UserProfileEditor interface {
	Rename(ctx context.Context, userID user.UserID, name string) error
	UpdateContact(ctx context.Context, userID user.UserID, phone meta.Phone, email meta.Email) error
	UpdateIDCard(ctx context.Context, userID user.UserID, idCard meta.IDCard) error
}

// UserQueryer 用户查询服务接口
type UserQueryer interface {
	FindByID(ctx context.Context, userID user.UserID) (*user.User, error)
	FindByPhone(ctx context.Context, phone meta.Phone) (*user.User, error)
}

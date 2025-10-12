package repository

import (
	"context"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/user"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/meta"
)

// UserRepository 用户存储接口
type UserRepository interface {
	Create(ctx context.Context, user user.User) error
	FindByID(ctx context.Context, id user.UserID) (user.User, error)
	FindByPhone(ctx context.Context, phone meta.Phone) (user.User, error)
	Update(ctx context.Context, user user.User) error
}

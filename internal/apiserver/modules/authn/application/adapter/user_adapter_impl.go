package adapter

import (
	"context"
	"errors"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	userdomain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/domain/user"
	userport "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/domain/user/port"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	"gorm.io/gorm"
)

// userAdapterImpl 用户适配器实现
type userAdapterImpl struct {
	userRepo userport.UserRepository
}

var _ UserAdapter = (*userAdapterImpl)(nil)

// NewUserAdapter 创建用户适配器
func NewUserAdapter(userRepo userport.UserRepository) UserAdapter {
	return &userAdapterImpl{
		userRepo: userRepo,
	}
}

// ExistsUser 检查用户是否存在
func (a *userAdapterImpl) ExistsUser(ctx context.Context, userID account.UserID) (bool, error) {
	// 将 authn 的 UserID 转换为 uc 的 UserID
	ucUserID := userdomain.NewUserID(userID.Uint64())

	user, err := a.userRepo.FindByID(ctx, ucUserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, perrors.WrapC(err, code.ErrDatabase, "check user existence failed")
	}

	return user != nil, nil
}

// GetUserStatus 获取用户状态
func (a *userAdapterImpl) GetUserStatus(ctx context.Context, userID account.UserID) (string, error) {
	ucUserID := userdomain.NewUserID(userID.Uint64())

	user, err := a.userRepo.FindByID(ctx, ucUserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", perrors.WithCode(code.ErrUserNotFound, "user not found")
		}
		return "", perrors.WrapC(err, code.ErrDatabase, "get user status failed")
	}

	// 将 uc 的 UserStatus 转换为字符串
	return string(user.Status), nil
}

// IsUserActive 检查用户是否活跃
func (a *userAdapterImpl) IsUserActive(ctx context.Context, userID account.UserID) (bool, error) {
	ucUserID := userdomain.NewUserID(userID.Uint64())

	user, err := a.userRepo.FindByID(ctx, ucUserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, perrors.WrapC(err, code.ErrDatabase, "check user active status failed")
	}

	return user != nil && user.IsUsable(), nil
}

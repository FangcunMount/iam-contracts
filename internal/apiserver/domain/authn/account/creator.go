package account

import (
	"context"

	"github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/component-base/pkg/util/idutil"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
)

// creator 账号创建器实现
type creator struct {
	repo Repository
}

// 确保实现了 Creator 接口
var _ Creator = (*creator)(nil)

// NewCreator 创建账号创建器实例
func NewCreator(repo Repository) Creator {
	return &creator{
		repo: repo,
	}
}

// Create 创建账户
func (c *creator) Create(ctx context.Context, dto CreateAccountDTO) (*Account, error) {
	// 参数校验，验证输入数据的合法性
	if !idutil.ValidateIntID(dto.UserID.Uint64()) {
		return nil, errors.WithCode(code.ErrInvalidArgument, "invalid user ID")
	}
	if !dto.AccountType.Validate() {
		return nil, errors.WithCode(code.ErrInvalidArgument, "invalid account type")
	}
	if dto.AppID.Len() == 0 {
		return nil, errors.WithCode(code.ErrInvalidArgument, "app ID cannot be empty")
	}
	if dto.ExternalID.Len() == 0 {
		return nil, errors.WithCode(code.ErrInvalidArgument, "external ID cannot be empty")
	}

	// 幂等性检查，确保同一用户、同一应用、同一外部ID的账号不会重复创建
	if existingAccount, err := c.repo.GetByExternalIDAppId(ctx, dto.ExternalID, dto.AppID); err != nil {
		return nil, err
	} else if existingAccount != nil && existingAccount.UserID == dto.UserID {
		return nil, errors.WithCode(code.ErrExternalExists, "create account failed, external ID belongs to the other user")
	} else if existingAccount != nil {
		return nil, errors.WithCode(code.ErrAccountExists, "create account failed, external ID already exists")
	}

	// 创建新的账号实体
	account := NewAccount(
		dto.UserID,
		dto.AccountType,
		dto.ExternalID,
		WithAppID(dto.AppID),
	)

	return account, nil
}

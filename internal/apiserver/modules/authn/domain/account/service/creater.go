package service

import (
	"context"

	"github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/component-base/pkg/util/idutil"
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account/port"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
)

// AccountCreater 账号创建器
type AccountCreater struct {
	repo port.AccountRepo
}

// Ensure AccountCreater implements the port.AccountCreater interface
var _ port.AccountCreater = (*AccountCreater)(nil)

// NewAccountCreater creates a new AccountCreater instance
func NewAccountCreater(repo port.AccountRepo) *AccountCreater {
	return &AccountCreater{
		repo: repo,
	}
}

// Create 创建账户
func (ac *AccountCreater) Create(ctx context.Context, dto port.CreateAccountDTO) (*domain.Account, error) {
	// 参数校验，验证输入数据的合法性
	if !idutil.ValidateIntID(dto.UserID.ToUint64()) {
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
	if existingAccount, err := ac.repo.GetByExternalIDAppId(ctx, dto.ExternalID, dto.AppID); err != nil {
		return nil, err
	} else if existingAccount != nil && existingAccount.UserID == dto.UserID {
		return nil, errors.WithCode(code.ErrExternalExists, "create account failed, external ID belongs to the other user")
	} else if existingAccount != nil {
		return nil, errors.WithCode(code.ErrAccountExists, "create account failed, external ID already exists")
	}

	// 创建新的账号实体
	account := domain.NewAccount(
		dto.UserID,
		dto.AccountType,
		dto.ExternalID,
		domain.WithAppID(dto.AppID),
	)

	return account, nil
}

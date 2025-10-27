package service

import (
	"context"
	"errors"
	"strings"

	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	drivenPort "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account/port/driven"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	perrors "github.com/FangcunMount/iam-contracts/pkg/errors"
	"gorm.io/gorm"
)

// 账号注册领域服务
// 职责：提供账号创建的工厂方法和业务规则验证
// 所有方法都是无状态的，不持有仓储引用

// CreateAccountEntity 创建账号实体（工厂方法）
// 封装账号创建的业务规则
func CreateAccountEntity(
	userID domain.UserID,
	provider domain.Provider,
	externalID string,
	appID *string,
) (*domain.Account, error) {
	if userID.IsZero() {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "user id cannot be empty")
	}

	providerValue := strings.TrimSpace(string(provider))
	if providerValue == "" {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "provider cannot be empty")
	}

	externalID = strings.TrimSpace(externalID)
	if externalID == "" {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "external_id cannot be empty")
	}

	var accountOpts []domain.AccountOption
	accountOpts = append(accountOpts, domain.WithExternalID(externalID))

	if appID != nil {
		value := strings.TrimSpace(*appID)
		if value != "" {
			accountOpts = append(accountOpts, domain.WithAppID(value))
		}
	}

	acc := domain.NewAccount(userID, domain.Provider(providerValue), accountOpts...)
	return &acc, nil
}

// CreateOperationAccountEntity 创建运营账号实体（工厂方法）
func CreateOperationAccountEntity(
	accountID domain.AccountID,
	username string,
) (*domain.OperationAccount, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "username cannot be empty")
	}

	return CreateOperationAccount(accountID, username, defaultOperationAccountAlgo)
}

// CreateWeChatAccountEntity 创建微信账号实体（工厂方法）
func CreateWeChatAccountEntity(
	accountID domain.AccountID,
	externalID string,
	appID string,
) (*domain.WeChatAccount, error) {
	externalID = strings.TrimSpace(externalID)
	if externalID == "" {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "external_id (openid) cannot be empty")
	}

	appID = strings.TrimSpace(appID)
	if appID == "" {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "app_id cannot be empty")
	}

	return CreateWeChatAccount(accountID, externalID, appID)
}

// ValidateAccountNotExists 验证账号不存在（业务规则验证）
func ValidateAccountNotExists(
	ctx context.Context,
	repo drivenPort.AccountRepo,
	provider domain.Provider,
	externalID string,
	appID *string,
) error {
	_, err := repo.FindByRef(ctx, provider, externalID, appID)
	if err == nil {
		return perrors.WithCode(code.ErrInvalidArgument, "account already exists")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return perrors.WrapC(err, code.ErrDatabase, "check account existence failed")
	}
	return nil
}

// ValidateOperationNotExists 验证运营账号不存在
func ValidateOperationNotExists(
	ctx context.Context,
	repo drivenPort.OperationRepo,
	username string,
) error {
	_, err := repo.FindByUsername(ctx, username)
	if err == nil {
		return perrors.WithCode(code.ErrInvalidArgument, "operation account already exists")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return perrors.WrapC(err, code.ErrDatabase, "check operation account failed")
	}
	return nil
}

// ValidateWeChatNotExists 验证微信账号不存在
func ValidateWeChatNotExists(
	ctx context.Context,
	repo drivenPort.WeChatRepo,
	appID string,
	openID string,
) error {
	_, err := repo.FindByAppOpenID(ctx, appID, openID)
	if err == nil {
		return perrors.WithCode(code.ErrInvalidArgument, "wechat account already exists")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return perrors.WrapC(err, code.ErrDatabase, "check wechat account failed")
	}
	return nil
}

// EnsureAccountExists 确保账号存在，如果不存在则创建
// 返回：(账号, 是否新创建, 错误)
func EnsureAccountExists(
	ctx context.Context,
	repo drivenPort.AccountRepo,
	userID domain.UserID,
	provider domain.Provider,
	externalID string,
	appID *string,
) (*domain.Account, bool, error) {
	// 查找已存在的账号
	existing, err := repo.FindByRef(ctx, provider, externalID, appID)
	if err == nil {
		// 检查用户ID是否匹配
		if existing.UserID != userID {
			return nil, false, perrors.WithCode(
				code.ErrInvalidArgument,
				"account already bound to another user",
			)
		}
		return existing, false, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, perrors.WrapC(err, code.ErrDatabase, "check account existence failed")
	}

	// 创建新账号
	account, err := CreateAccountEntity(userID, provider, externalID, appID)
	if err != nil {
		return nil, false, err
	}

	if err := repo.Create(ctx, account); err != nil {
		return nil, false, perrors.WrapC(err, code.ErrDatabase, "create account failed")
	}

	return account, true, nil
}

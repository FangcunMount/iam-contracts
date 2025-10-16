// Package driven 账号领域被驱动端口定义
package driven

import (
	"context"

	domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/account"
)

// 统一锚点
type AccountRepo interface {
	Create(ctx context.Context, a *domain.Account) error
	FindByID(ctx context.Context, id domain.AccountID) (*domain.Account, error)
	FindByRef(ctx context.Context, provider domain.Provider, externalID string, appID *string) (*domain.Account, error)
	UpdateStatus(ctx context.Context, id domain.AccountID, status domain.AccountStatus) error
	UpdateUserID(ctx context.Context, id domain.AccountID, userID domain.UserID) error
	UpdateExternalRef(ctx context.Context, id domain.AccountID, externalID string, appID *string) error
}

// 子实体：WeChat
type WeChatRepo interface {
	Create(ctx context.Context, wx *domain.WeChatAccount) error
	FindByAccountID(ctx context.Context, accountID domain.AccountID) (*domain.WeChatAccount, error)
	FindByAppOpenID(ctx context.Context, appID, openid string) (*domain.WeChatAccount, error)
	UpdateProfile(ctx context.Context, accountID domain.AccountID, nickname, avatar *string, meta []byte) error
	UpdateUnionID(ctx context.Context, accountID domain.AccountID, unionID string) error
}

// 子实体：Operation
type OperationRepo interface {
	Create(ctx context.Context, cred *domain.OperationAccount) error
	FindByAccountID(ctx context.Context, accountID domain.AccountID) (*domain.OperationAccount, error)
	FindByUsername(ctx context.Context, username string) (*domain.OperationAccount, error)
	UpdateHash(ctx context.Context, username string, hash []byte, algo string, params []byte) error
	UpdateUsername(ctx context.Context, accountID domain.AccountID, newUsername string) error
	ResetFailures(ctx context.Context, username string) error
	Unlock(ctx context.Context, username string) error
}

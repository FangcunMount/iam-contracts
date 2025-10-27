// Package driving 账号领域驱动端口定义
package driving

import (
	"context"

	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account"
)

// AccountRegisterer —— 账号注册服务接口，统一创建 Account + 子实体
type AccountRegisterer interface {
	// CreateAccount 创建账号
	CreateAccount(ctx context.Context, userID domain.UserID, externalID string) (*domain.Account, error)
	// CreateOperationAccount 创建运营账号
	CreateOperationAccount(ctx context.Context, userID domain.UserID, externalID string) (*domain.Account, *domain.OperationAccount, error)
	// CreateWeChatAccount 创建微信相关账号
	CreateWeChatAccount(ctx context.Context, userID domain.UserID, externalID, appID string) (*domain.Account, *domain.WeChatAccount, error)
}

// AccountEditor —— 账号编辑服务接口
type AccountEditor interface {
	UpdateWeChatProfile(ctx context.Context, accountID domain.AccountID, nickname, avatar *string, meta []byte) error
	SetWeChatUnionID(ctx context.Context, accountID domain.AccountID, unionID string) error
	UpdateOperationCredential(ctx context.Context, username string, newHash []byte, algo string, params []byte) error
	ChangeOperationUsername(ctx context.Context, oldUsername, newUsername string) error
	ResetOperationFailures(ctx context.Context, username string) error
	UnlockOperationAccount(ctx context.Context, username string) error
}

// AccountStatusUpdater —— 账号状态更新服务接口
type AccountStatusUpdater interface {
	DisableAccount(ctx context.Context, accountID domain.AccountID) error
	EnableAccount(ctx context.Context, accountID domain.AccountID) error
}

// AccountQueryer —— 账号查询服务接口
type AccountQueryer interface {
	FindAccountByID(ctx context.Context, accountID domain.AccountID) (*domain.Account, error)
	FindByUsername(ctx context.Context, username string) (*domain.Account, *domain.OperationAccount, error)
	FindByWeChatRef(ctx context.Context, externalID, appID string) (*domain.Account, *domain.WeChatAccount, error)
	FindByRef(ctx context.Context, provider domain.Provider, externalID string, appID *string) (*domain.Account, error)
	FindAccountListByUserID(ctx context.Context, userID domain.UserID) ([]*domain.Account, error)
}

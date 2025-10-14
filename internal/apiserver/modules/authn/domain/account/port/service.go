package port

import (
	"context"

	domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/account"
)

// AccountRegisterer —— 账号注册服务接口，统一创建 Account + 子实体
type AccountRegisterer interface {
	CreateWeChatAccount(ctx context.Context, provider domain.Provider, externalID string, appID *string, unionID *string, nickname string, avatar string) (*domain.Account, error)
	CreateOperationAccount(ctx context.Context, accountID string, username string, passwordHash []byte, algo string, params []byte) error
}

// AccountEditor —— 账号编辑服务接口
type AccountEditor interface {
	UpdateOperationCredential(ctx context.Context, username string, newHash []byte, algo string, params []byte) error
	ChangeUsername(ctx context.Context, oldUsername, newUsername string) error
}

// AccountStatusUpdater —— 账号状态更新服务接口
type AccountStatusUpdater interface {
	DisableAccount(ctx context.Context, accountID string) error
	EnableAccount(ctx context.Context, accountID string) error
}

// AccountQueryer —— 账号查询服务接口
type AccountQueryer interface {
	FindByID(ctx context.Context, accountID string) (*domain.Account, error)
	FindByUsername(ctx context.Context, username string) (*domain.Account, error)
}

// Package account 账号应用服务层
package account

import (
	"context"

	domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	req "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/interface/restful/request"
)

// AccountManagementService 账号管理应用服务
type AccountManagementService interface {
	// CreateOperationAccount 创建运营账号
	CreateOperationAccount(ctx context.Context, userID domain.UserID, req *req.CreateOperationAccountReq) (*domain.Account, *domain.OperationAccount, error)

	// GetAccount 获取账号详情
	GetAccount(ctx context.Context, accountID domain.AccountID) (*domain.Account, error)

	// ListAccountsByUser 列出用户的账号列表
	ListAccountsByUser(ctx context.Context, userID domain.UserID, offset, limit int) ([]*domain.Account, error)

	// EnableAccount 启用账号
	EnableAccount(ctx context.Context, accountID domain.AccountID) error

	// DisableAccount 禁用账号
	DisableAccount(ctx context.Context, accountID domain.AccountID) error
}

// OperationAccountService 运营账号管理应用服务
type OperationAccountService interface {
	// UpdateOperationCredential 更新运营账号凭据
	UpdateOperationCredential(ctx context.Context, username string, req *req.UpdateOperationCredentialReq) error

	// ChangeOperationUsername 修改运营账号用户名
	ChangeOperationUsername(ctx context.Context, oldUsername string, req *req.ChangeOperationUsernameReq) error

	// GetOperationAccountByUsername 根据用户名获取运营账号
	GetOperationAccountByUsername(ctx context.Context, username string) (*domain.Account, *domain.OperationAccount, error)
}

// WeChatAccountService 微信账号管理应用服务
type WeChatAccountService interface {
	// BindWeChatAccount 绑定微信账号
	BindWeChatAccount(ctx context.Context, accountID domain.AccountID, req *req.BindWeChatAccountReq) error

	// UpsertWeChatProfile 更新或插入微信资料
	UpsertWeChatProfile(ctx context.Context, accountID domain.AccountID, req *req.UpsertWeChatProfileReq) error

	// SetWeChatUnionID 设置微信UnionID
	SetWeChatUnionID(ctx context.Context, accountID domain.AccountID, unionID string) error
}

// AccountLookupService 账号查找应用服务
type AccountLookupService interface {
	// FindAccountByRef 根据引用查找账号
	FindAccountByRef(ctx context.Context, provider domain.Provider, externalID string, appID *string) (*domain.Account, error)
}

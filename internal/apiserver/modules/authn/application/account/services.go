// Package account 账号应用服务层
//
// 应用服务层职责：
// 1. 编排业务用例流程
// 2. 协调领域服务和领域对象
// 3. 处理事务边界
// 4. 转换API请求到领域对象
// 5. 处理跨聚合的业务逻辑
//
// 应用服务不包含业务规则，仅负责流程编排
package account

import (
	"context"

	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account"
)

// DTO定义 - 应用层的数据传输对象

// CreateOperationAccountDTO 创建运营账号DTO
type CreateOperationAccountDTO struct {
	UserID   domain.UserID
	Username string
	Password string
	HashAlgo string
}

// UpdateOperationCredentialDTO 更新运营账号凭据DTO
type UpdateOperationCredentialDTO struct {
	Username string
	Password string
	HashAlgo string
}

// ChangeUsernameDTO 修改用户名DTO
type ChangeUsernameDTO struct {
	OldUsername string
	NewUsername string
}

// BindWeChatAccountDTO 绑定微信账号DTO
type BindWeChatAccountDTO struct {
	AccountID  domain.AccountID
	ExternalID string
	AppID      string
	Nickname   *string
	Avatar     *string
	Meta       []byte
}

// UpdateWeChatProfileDTO 更新微信资料DTO
type UpdateWeChatProfileDTO struct {
	AccountID domain.AccountID
	Nickname  *string
	Avatar    *string
	Meta      []byte
}

// AccountResult 账号查询结果
type AccountResult struct {
	Account       *domain.Account
	OperationData *domain.OperationAccount
	WeChatData    *domain.WeChatAccount
}

// 应用服务接口定义

// AccountApplicationService 账号应用服务 - 负责账号管理相关的用例编排
type AccountApplicationService interface {
	// CreateOperationAccount 创建运营账号用例
	// 流程：1.创建账号聚合根 2.创建运营账号子实体 3.设置密码
	CreateOperationAccount(ctx context.Context, dto CreateOperationAccountDTO) (*AccountResult, error)

	// GetAccountByID 根据ID获取账号用例
	GetAccountByID(ctx context.Context, accountID domain.AccountID) (*AccountResult, error)

	// ListAccountsByUserID 列出用户的所有账号用例
	ListAccountsByUserID(ctx context.Context, userID domain.UserID) ([]*domain.Account, error)

	// EnableAccount 启用账号用例
	EnableAccount(ctx context.Context, accountID domain.AccountID) error

	// DisableAccount 禁用账号用例
	DisableAccount(ctx context.Context, accountID domain.AccountID) error
}

// OperationAccountApplicationService 运营账号应用服务 - 负责运营账号管理用例
type OperationAccountApplicationService interface {
	// UpdateCredential 更新运营账号凭据用例
	// 流程：1.查找账号 2.验证权限 3.更新密码 4.重置失败次数
	UpdateCredential(ctx context.Context, dto UpdateOperationCredentialDTO) error

	// ChangeUsername 修改运营账号用户名用例
	// 流程：1.验证新用户名唯一性 2.更新用户名 3.解锁账号
	ChangeUsername(ctx context.Context, dto ChangeUsernameDTO) error

	// GetByUsername 根据用户名获取运营账号用例
	GetByUsername(ctx context.Context, username string) (*AccountResult, error)

	// ResetFailures 重置失败次数用例
	ResetFailures(ctx context.Context, username string) error

	// UnlockAccount 解锁运营账号用例
	UnlockAccount(ctx context.Context, username string) error
}

// WeChatAccountApplicationService 微信账号应用服务 - 负责微信账号管理用例
type WeChatAccountApplicationService interface {
	// BindWeChatAccount 绑定微信账号用例
	// 流程：1.验证账号存在 2.创建微信子实体 3.更新资料
	BindWeChatAccount(ctx context.Context, dto BindWeChatAccountDTO) error

	// UpdateProfile 更新微信资料用例
	// 流程：1.查找微信账号 2.更新资料字段
	UpdateProfile(ctx context.Context, dto UpdateWeChatProfileDTO) error

	// SetUnionID 设置微信UnionID用例
	// 流程：1.查找微信账号 2.设置UnionID
	SetUnionID(ctx context.Context, accountID domain.AccountID, unionID string) error

	// GetByWeChatRef 根据微信引用查找账号用例
	GetByWeChatRef(ctx context.Context, externalID, appID string) (*AccountResult, error)
}

// AccountLookupApplicationService 账号查找应用服务 - 负责账号查询用例
type AccountLookupApplicationService interface {
	// FindByProvider 根据提供商查找账号用例
	FindByProvider(ctx context.Context, provider domain.Provider, externalID string, appID *string) (*domain.Account, error)
}

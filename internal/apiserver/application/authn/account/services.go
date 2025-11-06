package account

import (
	"context"

	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/account"
	credDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/credential"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

// ============= 应用服务接口（Driving Ports）=============

// AccountApplicationService 账户应用服务 - 已存在账户的管理
type AccountApplicationService interface {
	// GetAccountByID 根据ID获取账户
	GetAccountByID(ctx context.Context, accountID meta.ID) (*AccountResult, error)

	// FindByExternalRef 根据外部引用查找账户
	// 用于幂等性检查和已有账户查询
	FindByExternalRef(ctx context.Context, accountType domain.AccountType, appID domain.AppId, externalID domain.ExternalID) (*AccountResult, error)

	// FindByUniqueID 根据全局唯一标识查找账户
	FindByUniqueID(ctx context.Context, uniqueID domain.UnionID) (*AccountResult, error)

	// SetUniqueID 设置全局唯一标识（如 UnionID）
	SetUniqueID(ctx context.Context, accountID meta.ID, uniqueID domain.UnionID) error

	// UpdateProfile 更新账户资料
	UpdateProfile(ctx context.Context, accountID meta.ID, profile map[string]string) error

	// UpdateMeta 更新账户元数据
	UpdateMeta(ctx context.Context, accountID meta.ID, meta map[string]string) error

	// EnableAccount 启用账户
	EnableAccount(ctx context.Context, accountID meta.ID) error

	// DisableAccount 禁用账户
	DisableAccount(ctx context.Context, accountID meta.ID) error

	// ArchiveAccount 归档账户
	ArchiveAccount(ctx context.Context, accountID meta.ID) error

	// DeleteAccount 删除账户（软删除）
	DeleteAccount(ctx context.Context, accountID meta.ID) error
}

// CredentialApplicationService 凭据应用服务 - 凭据管理
type CredentialApplicationService interface {
	// BindCredential 绑定凭据到账户
	// 支持：password, phone_otp, oauth_wx_minip, oauth_wecom
	BindCredential(ctx context.Context, dto BindCredentialDTO) error

	// UnbindCredential 解绑凭据
	UnbindCredential(ctx context.Context, credentialID int64) error

	// RotatePassword 轮换密码（密码类型凭据）
	RotatePassword(ctx context.Context, accountID meta.ID, oldPassword, newPassword string) error

	// GetCredentialsByAccountID 获取账户的所有凭据
	GetCredentialsByAccountID(ctx context.Context, accountID meta.ID) ([]*CredentialResult, error)

	// DisableCredential 禁用凭据
	DisableCredential(ctx context.Context, credentialID int64) error

	// EnableCredential 启用凭据
	EnableCredential(ctx context.Context, credentialID int64) error
}

// ============= DTOs =============

// BindCredentialDTO 绑定凭据DTO
type BindCredentialDTO struct {
	AccountID     meta.ID                   // 账户ID（必须）
	Type          credDomain.CredentialType // 凭据类型
	IDP           *string                   // IDP类型："wechat" | "wecom" | "phone"
	IDPIdentifier string                    // IDP标识符：unionid | openid@appid | userid | +E164
	AppID         *string                   // 应用ID
	Material      []byte                    // 凭据材料（仅 password）
	Algo          *string                   // 算法（仅 password）
	ParamsJSON    []byte                    // 参数JSON
}

// AccountResult 账户结果DTO
type AccountResult struct {
	AccountID  meta.ID              // 账户ID
	UserID     meta.ID              // 用户ID
	Type       domain.AccountType   // 账户类型
	AppID      domain.AppId         // 应用ID
	ExternalID domain.ExternalID    // 外部标识
	UniqueID   domain.UnionID       // 全局唯一标识
	Profile    map[string]string    // 用户资料
	Meta       map[string]string    // 元数据
	Status     domain.AccountStatus // 账户状态
}

// CredentialResult 凭据结果DTO
type CredentialResult struct {
	ID            uint64                      // 凭据ID
	AccountID     uint64                      // 账户ID
	Type          credDomain.CredentialType   // 凭据类型
	IDP           *string                     // IDP类型
	IDPIdentifier string                      // IDP标识符
	AppID         *string                     // 应用ID
	Status        credDomain.CredentialStatus // 凭据状态
}

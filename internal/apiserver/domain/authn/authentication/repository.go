package authentication

import "context"

// ================== Repository Interfaces (Driven Ports) ==================
// 定义领域模型所依赖的仓储接口，由基础设施层提供实现

// CredentialRepository 凭据仓储（查询认证凭据）
// 职责：提供各类凭据的查询能力
type CredentialRepository interface {
	// FindPasswordCredential 根据账户ID查找密码凭据
	// 返回：凭据ID、密码哈希值（PHC格式）
	FindPasswordCredential(ctx context.Context, accountID int64) (credentialID int64, passwordHash string, err error)

	// FindPhoneOTPCredential 根据手机号查找OTP凭据绑定
	// 返回：账户ID、用户ID、凭据ID
	FindPhoneOTPCredential(ctx context.Context, phoneE164 string) (accountID, userID, credentialID int64, err error)

	// FindOAuthCredential 根据身份提供商标识查找OAuth凭据绑定
	// idpType: "wx_minip" | "wecom" | ...
	// idpIdentifier: OpenID/UnionID/UserID
	// 返回：账户ID、用户ID、凭据ID
	FindOAuthCredential(ctx context.Context, idpType, appID, idpIdentifier string) (accountID, userID, credentialID int64, err error)
}

// AccountRepository 账户仓储（查询账户信息）
// 职责：提供账户主体信息的查询能力
type AccountRepository interface {
	// FindAccountByUsername 根据用户名查找账户
	// 返回：账户ID、用户ID
	FindAccountByUsername(ctx context.Context, tenantID *int64, username string) (accountID, userID int64, err error)

	// GetAccountStatus 获取账户状态（用于检查是否锁定/禁用）
	// 返回：是否启用、是否锁定
	GetAccountStatus(ctx context.Context, accountID int64) (enabled, locked bool, err error)
}

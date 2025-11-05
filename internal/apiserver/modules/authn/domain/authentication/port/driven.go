package port

import "context"

// ================== 按职责划分的 Driven 端口 ==================

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

// PasswordHasher 密码哈希服务（密码加密算法）
// 职责：提供密码哈希验证和rehash能力
type PasswordHasher interface {
	// Verify 验证明文密码与存储的哈希值是否匹配
	// storedHash: PHC格式的哈希值，如 $argon2id$v=19$m=65536,t=3,p=4$...
	// plaintext: 明文密码
	Verify(storedHash, plaintext string) bool

	// NeedRehash 检查哈希值是否需要重新哈希（算法升级）
	NeedRehash(storedHash string) bool

	// Hash 对明文密码进行哈希
	Hash(plaintext string) (string, error)

	// Pepper 获取全局pepper（用于加盐）
	Pepper() string
}

// OTPVerifier OTP验证服务（一次性密码验证）
// 职责：验证OTP并消费（防止重放）
type OTPVerifier interface {
	// VerifyAndConsume 验证OTP并标记为已使用
	// scene: "login" | "register" | ...
	// 返回：是否验证通过
	VerifyAndConsume(ctx context.Context, phoneE164, scene, code string) bool
}

// IdentityProvider 身份提供商服务（OAuth/OIDC）
// 职责：与外部IdP交互，换取用户身份标识
type IdentityProvider interface {
	// ExchangeWxMinipCode 微信小程序 code 换 session
	// 返回：OpenID、UnionID（可选）
	ExchangeWxMinipCode(ctx context.Context, appID, jsCode string) (openID, unionID string, err error)

	// ExchangeWecomCode 企业微信 code 换 用户信息
	// 返回：OpenUserID、UserID
	ExchangeWecomCode(ctx context.Context, corpID, code string) (openUserID, userID string, err error)
}

// TokenVerifier JWT令牌验证服务
// 职责：验证JWT访问令牌的有效性
type TokenVerifier interface {
	// VerifyAccessToken 验证访问令牌
	// 返回：用户ID、账户ID、租户ID（可选）、错误信息
	// 如果令牌无效/过期/被撤销，返回错误
	VerifyAccessToken(ctx context.Context, tokenValue string) (userID, accountID int64, tenantID *int64, err error)
}

// ================== 可选：审计与安全 ==================

// AuditLogger 审计日志（可选）
// 职责：记录认证事件（成功/失败/锁定）
type AuditLogger interface {
	LogAuthAttempt(ctx context.Context, event AuthAuditEvent)
}

type AuthAuditEvent struct {
	AccountID    int64
	CredentialID int64
	Scenario     string
	Success      bool
	ErrCode      string
	RemoteIP     string
	UserAgent    string
}

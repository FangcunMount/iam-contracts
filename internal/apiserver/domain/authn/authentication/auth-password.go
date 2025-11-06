package authentication

import (
	"context"
	"fmt"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
)

// Register the password credential builder
func init() {
	RegisterCredentialBuilder(AuthPassword, newPasswordCredential)
}

// ====================== 认证凭据（认证所需的数据） ========================

// PasswordCredential 认证凭据（用户名+密码）
type PasswordCredential struct {
	TenantID  *int64
	RemoteIP  string
	UserAgent string
	Username  string
	Password  string
}

// Scenario 返回认证场景
func (c *PasswordCredential) Scenario() Scenario {
	return AuthPassword
}

// newPasswordCredential 构造密码认证凭据
func newPasswordCredential(input AuthInput) (AuthCredential, error) {
	if input.Username == "" {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "username is required for password authentication")
	}
	if input.Password == "" {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "password is required for password authentication")
	}

	return &PasswordCredential{
		TenantID:  input.TenantID,
		RemoteIP:  input.RemoteIP,
		UserAgent: input.UserAgent,
		Username:  input.Username,
		Password:  input.Password,
	}, nil
}

// ================= 认证策略（执行认证的认证器） ========================

// PasswordAuthStrategy 用户名+密码认证策略
type PasswordAuthStrategy struct {
	scenario    Scenario
	credRepo    CredentialRepository
	accountRepo AccountRepository
	hasher      PasswordHasher
}

// 实现认证策略接口
var _ AuthStrategy = (*PasswordAuthStrategy)(nil)

// NewPasswordAuthStrategy 构造函数（注入依赖）
func NewPasswordAuthStrategy(
	credRepo CredentialRepository,
	accountRepo AccountRepository,
	hasher PasswordHasher,
) *PasswordAuthStrategy {
	return &PasswordAuthStrategy{
		scenario:    AuthPassword,
		credRepo:    credRepo,
		accountRepo: accountRepo,
		hasher:      hasher,
	}
}

// Kind 返回认证策略类型
func (p *PasswordAuthStrategy) Kind() Scenario {
	return p.scenario
}

// Authenticate 执行用户名+密码认证
// 认证流程：
// 1. 根据用户名查找账户
// 2. 检查账户状态（是否锁定/禁用）
// 3. 查找密码凭据
// 4. 验证密码（带pepper）
// 5. 检查是否需要密码rehash（算法升级）
// 6. 返回认证判决
func (p *PasswordAuthStrategy) Authenticate(ctx context.Context, credential AuthCredential) (AuthDecision, error) {
	passwordCredential, ok := credential.(*PasswordCredential)
	if !ok {
		return AuthDecision{}, fmt.Errorf("password strategy expects *PasswordCredential, got %T", credential)
	}

	// Step 1: 根据用户名查找账户
	accountID, userID, err := p.accountRepo.FindAccountByUsername(ctx, passwordCredential.TenantID, passwordCredential.Username)
	if err != nil {
		// 系统异常（如数据库错误）
		return AuthDecision{}, fmt.Errorf("failed to find account: %w", err)
	}
	if accountID == 0 {
		// 业务失败：账户不存在（用统一的错误码，防止用户名枚举攻击）
		return AuthDecision{
			OK:      false,
			ErrCode: ErrInvalidCredential,
		}, nil
	}

	// Step 2: 检查账户状态
	enabled, locked, err := p.accountRepo.GetAccountStatus(ctx, accountID)
	if err != nil {
		return AuthDecision{}, fmt.Errorf("failed to get account status: %w", err)
	}
	if !enabled {
		return AuthDecision{
			OK:      false,
			ErrCode: ErrDisabled,
		}, nil
	}
	if locked {
		return AuthDecision{
			OK:      false,
			ErrCode: ErrLocked,
		}, nil
	}

	// Step 3: 查找密码凭据
	credentialID, storedHash, err := p.credRepo.FindPasswordCredential(ctx, accountID)
	if err != nil {
		return AuthDecision{}, fmt.Errorf("failed to find password credential: %w", err)
	}
	if credentialID == 0 {
		// 账户没有设置密码
		return AuthDecision{
			OK:      false,
			ErrCode: ErrInvalidCredential,
		}, nil
	}

	// Step 4: 验证密码（加上全局pepper）
	plaintextWithPepper := passwordCredential.Password + p.hasher.Pepper()
	if !p.hasher.Verify(storedHash, plaintextWithPepper) {
		// 密码错误（返回凭据ID用于失败次数统计）
		return AuthDecision{
			OK:           false,
			ErrCode:      ErrInvalidCredential,
			CredentialID: credentialID,
		}, nil
	}

	// Step 5: 检查是否需要密码rehash（例如算法参数升级）
	var shouldRotate bool
	var newHashBytes []byte
	if p.hasher.NeedRehash(storedHash) {
		newHash, err := p.hasher.Hash(plaintextWithPepper)
		if err != nil {
			// rehash失败不应该阻止认证成功
			// 记录日志即可，由应用层决定是否处理
		} else {
			shouldRotate = true
			newHashBytes = []byte(newHash)
		}
	}

	// Step 6: 认证成功，构造Principal
	principal := &Principal{
		AccountID: accountID,
		UserID:    userID,
		TenantID:  passwordCredential.TenantID,
		AMR:       []string{string(AMRPassword)},
		Claims: map[string]any{
			"auth_time": ctx.Value("request_time"), // 认证时间
		},
	}

	return AuthDecision{
		OK:           true,
		Principal:    principal,
		CredentialID: credentialID,
		ShouldRotate: shouldRotate,
		NewMaterial:  newHashBytes,
	}, nil
}

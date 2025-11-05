package service

import (
	"context"
	"fmt"

	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/authentication"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/authentication/port"
)

// PasswordAuthStrategy 用户名+密码认证策略
type PasswordAuthStrategy struct {
	scenario    domain.Scenario
	credRepo    port.CredentialRepository
	accountRepo port.AccountRepository
	hasher      port.PasswordHasher
}

// 实现认证策略接口
var _ domain.AuthStrategy = (*PasswordAuthStrategy)(nil)

// NewPasswordAuthStrategy 构造函数（注入依赖）
func NewPasswordAuthStrategy(
	credRepo port.CredentialRepository,
	accountRepo port.AccountRepository,
	hasher port.PasswordHasher,
) *PasswordAuthStrategy {
	return &PasswordAuthStrategy{
		scenario:    domain.AuthPassword,
		credRepo:    credRepo,
		accountRepo: accountRepo,
		hasher:      hasher,
	}
}

// Kind 返回认证策略类型
func (p *PasswordAuthStrategy) Kind() domain.Scenario {
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
func (p *PasswordAuthStrategy) Authenticate(ctx context.Context, in domain.AuthInput) (domain.AuthDecision, error) {
	// Step 1: 根据用户名查找账户
	accountID, userID, err := p.accountRepo.FindAccountByUsername(ctx, in.TenantID, in.Username)
	if err != nil {
		// 系统异常（如数据库错误）
		return domain.AuthDecision{}, fmt.Errorf("failed to find account: %w", err)
	}
	if accountID == 0 {
		// 业务失败：账户不存在（用统一的错误码，防止用户名枚举攻击）
		return domain.AuthDecision{
			OK:      false,
			ErrCode: domain.ErrInvalidCredential,
		}, nil
	}

	// Step 2: 检查账户状态
	enabled, locked, err := p.accountRepo.GetAccountStatus(ctx, accountID)
	if err != nil {
		return domain.AuthDecision{}, fmt.Errorf("failed to get account status: %w", err)
	}
	if !enabled {
		return domain.AuthDecision{
			OK:      false,
			ErrCode: domain.ErrDisabled,
		}, nil
	}
	if locked {
		return domain.AuthDecision{
			OK:      false,
			ErrCode: domain.ErrLocked,
		}, nil
	}

	// Step 3: 查找密码凭据
	credentialID, storedHash, err := p.credRepo.FindPasswordCredential(ctx, accountID)
	if err != nil {
		return domain.AuthDecision{}, fmt.Errorf("failed to find password credential: %w", err)
	}
	if credentialID == 0 {
		// 账户没有设置密码
		return domain.AuthDecision{
			OK:      false,
			ErrCode: domain.ErrInvalidCredential,
		}, nil
	}

	// Step 4: 验证密码（加上全局pepper）
	plaintextWithPepper := in.Password + p.hasher.Pepper()
	if !p.hasher.Verify(storedHash, plaintextWithPepper) {
		// 密码错误（返回凭据ID用于失败次数统计）
		return domain.AuthDecision{
			OK:           false,
			ErrCode:      domain.ErrInvalidCredential,
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
	principal := &domain.Principal{
		AccountID: accountID,
		UserID:    userID,
		TenantID:  in.TenantID,
		AMR:       []string{string(domain.AMRPassword)},
		Claims: map[string]any{
			"auth_time": ctx.Value("request_time"), // 认证时间
		},
	}

	return domain.AuthDecision{
		OK:           true,
		Principal:    principal,
		CredentialID: credentialID,
		ShouldRotate: shouldRotate,
		NewMaterial:  newHashBytes,
	}, nil
}

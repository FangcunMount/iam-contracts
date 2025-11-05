package service

import (
"context"
"fmt"

domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/authentication"
"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/authentication/port"
)

// PhoneOTPAuthStrategy 手机短信验证码认证策略
type PhoneOTPAuthStrategy struct {
	scenario   domain.Scenario
	credRepo   port.CredentialRepository
	accountRepo port.AccountRepository
	otpVerifier port.OTPVerifier
}

// 实现认证策略接口
var _ domain.AuthStrategy = (*PhoneOTPAuthStrategy)(nil)

// NewPhoneOTPAuthStrategy 构造函数（注入依赖）
func NewPhoneOTPAuthStrategy(
credRepo port.CredentialRepository,
accountRepo port.AccountRepository,
otpVerifier port.OTPVerifier,
) *PhoneOTPAuthStrategy {
	return &PhoneOTPAuthStrategy{
		scenario:    domain.AuthPhoneOTP,
		credRepo:    credRepo,
		accountRepo: accountRepo,
		otpVerifier: otpVerifier,
	}
}

// Kind 返回认证策略类型
func (p *PhoneOTPAuthStrategy) Kind() domain.Scenario {
	return p.scenario
}

// Authenticate 执行手机验证码认证
// 认证流程：
// 1. 验证并消费OTP（防止重放攻击）
// 2. 根据手机号查找凭据绑定
// 3. 检查账户状态
// 4. 返回认证判决
func (p *PhoneOTPAuthStrategy) Authenticate(ctx context.Context, in domain.AuthInput) (domain.AuthDecision, error) {
	// Step 1: 验证OTP并标记为已使用
	const otpScene = "login" // OTP场景：登录
	if !p.otpVerifier.VerifyAndConsume(ctx, in.PhoneE164, otpScene, in.OTP) {
		// 业务失败：OTP无效或已过期
		return domain.AuthDecision{
			OK:      false,
			ErrCode: domain.ErrOTPMissingOrExpiry,
		}, nil
	}

	// Step 2: 根据手机号查找凭据绑定
	accountID, userID, credentialID, err := p.credRepo.FindPhoneOTPCredential(ctx, in.PhoneE164)
	if err != nil {
		return domain.AuthDecision{}, fmt.Errorf("failed to find phone OTP credential: %w", err)
	}
	if credentialID == 0 {
		// 业务失败：手机号未绑定账户
		return domain.AuthDecision{
			OK:      false,
			ErrCode: domain.ErrNoBinding,
		}, nil
	}

	// Step 3: 检查账户状态
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

	// Step 4: 认证成功，构造Principal
	principal := &domain.Principal{
		AccountID: accountID,
		UserID:    userID,
		TenantID:  in.TenantID,
		AMR:       []string{string(domain.AMROTP)},
		Claims: map[string]any{
			"phone_number": in.PhoneE164,
			"auth_time":    ctx.Value("request_time"),
		},
	}

	return domain.AuthDecision{
		OK:           true,
		Principal:    principal,
		CredentialID: credentialID,
	}, nil
}

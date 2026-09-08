package authentication

import (
	"context"
	"fmt"
	"time"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authn/loginidentity"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
)

// ====================== 身份核验证明（请求级输入） ========================

// PhoneOTPProofSpec 手机号验证码身份核验证明规格
type PhoneOTPProofSpec struct {
	PhoneE164 string // 手机号
	OTP       string // 验证码
}

// PhoneOTPProof 身份核验证明（手机号+验证码）
type PhoneOTPProof struct {
	PhoneE164 string // 手机号
	OTP       string // 验证码
}

// 确保 PhoneOTPProof 实现了 IdentityProof 接口
var _ IdentityProof = (*PhoneOTPProof)(nil)

// CredentialKind 返回身份核验证明类型
func (c *PhoneOTPProof) CredentialKind() CredentialKind {
	return CredentialKindPhoneOTP
}

// NewPhoneOTPProof 构造手机号验证码身份核验证明
func NewPhoneOTPProof(spec PhoneOTPProofSpec) (IdentityProof, error) {
	if spec.PhoneE164 == "" {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "phone number is required for phone otp authentication")
	}
	if spec.OTP == "" {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "otp code is required for phone otp authentication")
	}

	return &PhoneOTPProof{
		PhoneE164: spec.PhoneE164,
		OTP:       spec.OTP,
	}, nil
}

// ================= 身份核验策略 ========================

// PhoneOTPAuthStrategy 手机短信验证码身份核验策略
type PhoneOTPAuthStrategy struct {
	credentialKind CredentialKind
	identityRepo   LoginIdentityRepository
	otpVerifier    LoginPhoneOTPVerifier
}

// 实现身份核验策略接口
var _ AuthStrategy = (*PhoneOTPAuthStrategy)(nil)

func NewPhoneOTPAuthStrategyWithLoginIdentity(
	identityRepo LoginIdentityRepository,
	otpVerifier LoginPhoneOTPVerifier,
) *PhoneOTPAuthStrategy {
	return &PhoneOTPAuthStrategy{
		credentialKind: CredentialKindPhoneOTP,
		identityRepo:   identityRepo,
		otpVerifier:    otpVerifier,
	}
}

// Kind 返回身份核验策略类型
func (p *PhoneOTPAuthStrategy) Kind() CredentialKind {
	return p.credentialKind
}

// Authenticate 执行手机验证码认证
// 身份核验流程：
// 1. 验证并消费OTP（防止重放攻击）
// 2. 根据手机号查找登录身份
// 3. 检查 LoginIdentity 状态
// 4. 返回身份核验决策
func (p *PhoneOTPAuthStrategy) Authenticate(ctx context.Context, credential IdentityProof) (AuthDecision, error) {
	// 断言身份核验证明类型
	otpCredential, ok := credential.(*PhoneOTPProof)
	if !ok {
		return AuthDecision{}, fmt.Errorf("phone otp strategy expects *PhoneOTPProof, got %T", credential)
	}

	// 验证并消费OTP（防止重放攻击）
	verified, err := p.verifyLoginOTP(ctx, otpCredential)
	if err != nil {
		return AuthDecision{}, fmt.Errorf("verify login phone OTP: %w", err)
	}
	if !verified {
		return AuthDecision{
			OK:   false,
			Code: code.ErrOTPInvalid,
		}, nil
	}

	// 根据手机号查找登录身份
	lookup, err := p.identityRepo.FindLoginIdentityByProviderKey(
		ctx,
		loginidentity.ProviderPhone,
		loginidentity.RealmGlobal,
		otpCredential.PhoneE164,
	)
	if err != nil {
		return AuthDecision{}, fmt.Errorf("failed to find phone login identity: %w", err)
	}
	if lookup == nil || lookup.LoginIdentityID.IsZero() {
		return AuthDecision{
			OK:   false,
			Code: code.ErrNoBinding,
		}, nil
	}

	// 检查登录身份状态
	statusFailure, err := loginIdentityStatusFailureDecision(ctx, p.identityRepo, lookup.LoginIdentityID)
	if err != nil {
		return AuthDecision{}, err
	}
	if statusFailure != nil {
		return *statusFailure, nil
	}

	// 构造身份核验成功决策
	return p.buildPhoneOTPSuccessDecision(
		lookup.LoginIdentityID,
		lookup.UserID,
	), nil
}

// verifyLoginOTP 验证OTP并标记为已使用
func (p *PhoneOTPAuthStrategy) verifyLoginOTP(ctx context.Context, credential *PhoneOTPProof) (bool, error) {
	return p.otpVerifier.VerifyAndConsumeLoginPhoneOTP(ctx, credential.PhoneE164, credential.OTP)
}

// buildPhoneOTPSuccessDecision 身份核验成功，构造Principal
func (p *PhoneOTPAuthStrategy) buildPhoneOTPSuccessDecision(
	loginIdentityID meta.ID,
	userID meta.ID,
) AuthDecision {
	principal := &Principal{
		LoginIdentityID: loginIdentityID,
		UserID:          userID,
	}
	principal.ApplyAuthContext(NewAuthenticationContext(MethodPhoneOTP, loginidentity.RealmGlobal, []AMR{AMROTP}, time.Now().UTC()))

	return AuthDecision{
		OK:        true,
		Principal: principal,
	}
}

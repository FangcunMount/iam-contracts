package authentication

import (
	"context"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/authentication/port"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
)

// AuthStrategy 认证策略（领域服务接口）
type AuthStrategy interface {
	Kind() Scenario
	Authenticate(ctx context.Context, credential AuthCredential) (AuthDecision, error)
}

// Authenticater 认证器
type Authenticater struct {
	credRepo      port.CredentialRepository
	accountRepo   port.AccountRepository
	hasher        port.PasswordHasher
	otpVerifier   port.OTPVerifier
	idp           port.IdentityProvider
	tokenVerifier port.TokenVerifier
}

// NewAuthenticater 创建认证器
func NewAuthenticater(
	credRepo port.CredentialRepository,
	accountRepo port.AccountRepository,
	hasher port.PasswordHasher,
	otpVerifier port.OTPVerifier,
	idp port.IdentityProvider,
	tokenVerifier port.TokenVerifier,
) *Authenticater {
	return &Authenticater{
		credRepo:      credRepo,
		accountRepo:   accountRepo,
		hasher:        hasher,
		otpVerifier:   otpVerifier,
		idp:           idp,
		tokenVerifier: tokenVerifier,
	}
}

// Authenticate 认证
// 统一流程：
// 1. 根据场景构建领域凭据
// 2. 获取并创建认证策略
// 3. 执行认证
func (a *Authenticater) Authenticate(ctx context.Context, scenario Scenario, input AuthInput) (AuthDecision, error) {
	// 根据场景构建领域凭据
	credential, err := a.buildCredential(scenario, input)
	if err != nil {
		return AuthDecision{}, err
	}

	// 创建认证策略
	strategy := a.createStrategy(scenario)
	if strategy == nil {
		return AuthDecision{}, perrors.WithCode(code.ErrInvalidArgument, "unsupported authentication scenario: %s", scenario)
	}

	// 执行认证
	return strategy.Authenticate(ctx, credential)
}

// BuildCredential 根据认证场景构建领域凭据
func (a *Authenticater) buildCredential(kind Scenario, input AuthInput) (AuthCredential, error) {
	builder, err := getCredentialBuilder(kind)
	if err != nil {
		return nil, err
	}

	credential, err := builder(input)
	if err != nil {
		return nil, err
	}
	if credential == nil {
		return nil, perrors.WithCode(code.ErrMissingECParams, "credential builder returned nil for scenario: %s", kind)
	}
	return credential, nil
}

// CreateStrategy 根据场景创建认证策略
func (f *Authenticater) createStrategy(scenario Scenario) AuthStrategy {
	switch scenario {
	case AuthPassword:
		return NewPasswordAuthStrategy(f.credRepo, f.accountRepo, f.hasher)
	case AuthPhoneOTP:
		return NewPhoneOTPAuthStrategy(f.credRepo, f.accountRepo, f.otpVerifier)
	case AuthWxMinip:
		return NewOAuthWechatMinipAuthStrategy(f.credRepo, f.accountRepo, f.idp)
	case AuthWecom:
		return NewOAuthWeChatComAuthStrategy(f.credRepo, f.accountRepo, f.idp)
	case AuthJWTToken:
		return NewJWTTokenAuthStrategy(f.tokenVerifier, f.accountRepo)
	default:
		return nil
	}
}

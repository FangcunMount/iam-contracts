package service

import (
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/authentication"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/authentication/port"
)

// StrategyFactory 认证策略工厂（依赖注入容器）
// 职责：根据场景创建对应的认证策略实例
type StrategyFactory struct {
	credRepo      port.CredentialRepository
	accountRepo   port.AccountRepository
	hasher        port.PasswordHasher
	otpVerifier   port.OTPVerifier
	idp           port.IdentityProvider
	tokenVerifier port.TokenVerifier
}

// NewStrategyFactory 创建策略工厂（注入所有依赖端口）
func NewStrategyFactory(
	credRepo port.CredentialRepository,
	accountRepo port.AccountRepository,
	hasher port.PasswordHasher,
	otpVerifier port.OTPVerifier,
	idp port.IdentityProvider,
	tokenVerifier port.TokenVerifier,
) *StrategyFactory {
	return &StrategyFactory{
		credRepo:      credRepo,
		accountRepo:   accountRepo,
		hasher:        hasher,
		otpVerifier:   otpVerifier,
		idp:           idp,
		tokenVerifier: tokenVerifier,
	}
}

// CreateStrategy 根据场景创建认证策略
func (f *StrategyFactory) CreateStrategy(scenario domain.Scenario) domain.AuthStrategy {
	switch scenario {
	case domain.AuthPassword:
		return NewPasswordAuthStrategy(f.credRepo, f.accountRepo, f.hasher)
	case domain.AuthPhoneOTP:
		return NewPhoneOTPAuthStrategy(f.credRepo, f.accountRepo, f.otpVerifier)
	case domain.AuthWxMinip:
		return NewOAuthWechatMinipAuthStrategy(f.credRepo, f.accountRepo, f.idp)
	case domain.AuthWecom:
		return NewOAuthWeChatComAuthStrategy(f.credRepo, f.accountRepo, f.idp)
	case domain.AuthJWTToken:
		return NewJWTTokenAuthStrategy(f.tokenVerifier, f.accountRepo)
	default:
		return nil
	}
}

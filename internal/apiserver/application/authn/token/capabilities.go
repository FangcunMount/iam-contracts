package token

import (
	"context"
	"strings"
	"time"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	admissionapp "github.com/FangcunMount/iam/v5/internal/apiserver/application/authn/admission"
	sessiondomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authn/session"
	tokendomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authn/token"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
)

// application 组合令牌用例协作者，并分别实现对外窄能力。
type application struct {
	initialIssuer InitialTokenIssuer
	refresher     tokendomain.Refresher
	verifier      tokendomain.Verifier
	revoker       tokendomain.Revoker
}

// Dependencies 是令牌用例能力的装配依赖。
type Dependencies struct {
	Encoder               AccessTokenEncoder
	SignatureVerifier     AccessTokenSignatureVerifier               // 令牌编码器
	TokenStore            Store                                      // 令牌存储
	SessionLoader         SessionLoader                              // 会话加载器
	SessionRevoker        SessionRevoker                             // 会话撤销器
	SessionExtender       SessionExtender                            // 会话延期器
	SessionRefreshExpirer SessionRefreshExpirer                      // refresh token 过期时间计算器
	AdmissionPolicy       AdmissionPolicy                            // 登录准入策略
	LegacyContextDecoder  LegacyAuthenticationContextSnapshotDecoder // 历史 refresh 认证上下文快照解码器
	Issuance              IssuanceConfig
	Now                   func() time.Time // 令牌有效期
}

var (
	_ InitialTokenIssuer = (*application)(nil)
	_ Refresher          = (*application)(nil)
	_ Revoker            = (*application)(nil)
	_ Verifier           = (*application)(nil)
)

// NewCapabilities 装配并返回相互独立的令牌用例能力。
func NewCapabilities(deps Dependencies) Capabilities {
	domainCapabilities := tokendomain.NewCapabilities(tokendomain.Dependencies{
		Encoder: deps.Encoder, SignatureVerifier: deps.SignatureVerifier, TokenStore: deps.TokenStore,
		SessionLoader:  deps.SessionLoader,
		SessionRevoker: deps.SessionRevoker, SessionExtender: deps.SessionExtender,
		SessionRefreshExpirer: deps.SessionRefreshExpirer, AdmissionPolicy: deps.AdmissionPolicy,
		LegacyContextDecoder: deps.LegacyContextDecoder, Issuance: deps.Issuance, Now: deps.Now,
	})
	app := &application{
		initialIssuer: NewInitialTokenIssuer(domainCapabilities.TokenSetMinter, deps.TokenStore),
		refresher:     domainCapabilities.Refresher,
		verifier:      domainCapabilities.Verifier,
		revoker:       domainCapabilities.Revoker,
	}
	return Capabilities{
		InitialTokenIssuer: app,
		Refresher:          app,
		Revoker:            app,
		Verifier:           app,
	}
}

// IssueInitialTokens 只在既有 Session 上签发并保存初始令牌。
func (s *application) IssueInitialTokens(ctx context.Context, sess *sessiondomain.Session) (*TokenPair, error) {
	return s.initialIssuer.IssueInitialTokens(ctx, sess)
}

// RefreshToken 使用 refresh token 轮换出新的 access/refresh token pair。
// 具体流程由内部 refresher 完成，包括 refresh token 读取、旧 token 删除和 session 延期。
func (s *application) RefreshToken(ctx context.Context, refreshToken string) (*TokenRefreshResult, error) {
	tokenSet, err := s.refresher.RefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, admissionapp.MapError(err)
	}

	return &TokenRefreshResult{
		TokenPair: tokenPairFromDomain(tokenSet),
	}, nil
}

// RevokeAccessToken 保留已发布调用名，实际撤销 access bearer token；
// 只有 access token 会连带撤销用户 Session。
func (s *application) RevokeAccessToken(ctx context.Context, accessToken string) error {
	err := s.revoker.RevokeBearerToken(ctx, accessToken)
	if err != nil {
		return perrors.WrapC(err, code.ErrTokenRevokeFailed, "failed to revoke bearer token")
	}

	return nil
}

// RevokeRefreshToken 删除 refresh token；如果 refresh token 关联 session，则同步撤销 session。
func (s *application) RevokeRefreshToken(ctx context.Context, refreshToken string) error {
	err := s.refresher.RevokeRefreshToken(ctx, refreshToken)
	if err != nil {
		return perrors.WrapC(err, code.ErrTokenRevokeFailed, "failed to revoke refresh token")
	}
	return nil
}

// VerifyToken 在线验证令牌，并检查场景级 issuer/audience/token type 约束。
// 密码学与 canonical issuer 由 codec 负责；撤销、Session、Admission 由 domain verifier 负责。
func (s *application) VerifyToken(ctx context.Context, req VerifyTokenRequest) (*TokenVerifyResult, error) {
	audience, err := tokendomain.NormalizeExpectedAudience(req.ExpectedAudience)
	if err != nil {
		return nil, err
	}
	claims, err := s.verifier.VerifyToken(ctx, req.AccessToken, audience)
	if err != nil {
		err = admissionapp.MapError(err)
		failureCode := tokenVerificationFailureCode(err)
		if failureCode == 0 {
			return nil, err
		}
		return &TokenVerifyResult{
			Valid:       false,
			Claims:      nil,
			FailureCode: failureCode,
		}, nil
	}

	if expectedIssuer := strings.TrimSpace(req.ExpectedIssuer); expectedIssuer != "" && claims.Issuer != expectedIssuer {
		return &TokenVerifyResult{Valid: false, Claims: nil}, nil
	}

	acceptedTokenTypes := req.AcceptedTokenTypes
	if len(acceptedTokenTypes) == 0 {
		acceptedTokenTypes = []TokenType{TokenTypeAccess}
	}
	if !containsTokenType(acceptedTokenTypes, claims.TokenType) {
		return &TokenVerifyResult{Valid: false, Claims: nil}, nil
	}

	return &TokenVerifyResult{
		Valid:  true,
		Claims: claims,
	}, nil
}

func containsTokenType(accepted []TokenType, actual TokenType) bool {
	for _, tokenType := range accepted {
		if tokenType == actual {
			return true
		}
	}
	return false
}

func tokenVerificationFailureCode(err error) int {
	codeValue := perrors.ParseCoder(err).Code()
	switch codeValue {
	case code.ErrTokenInvalid,
		code.ErrExpired,
		code.ErrUserBlocked,
		code.ErrUserInactive,
		code.ErrLoginIdentityDisabled,
		code.ErrCredentialLocked,
		code.ErrSessionInactive:
		return codeValue
	default:
		return 0
	}
}

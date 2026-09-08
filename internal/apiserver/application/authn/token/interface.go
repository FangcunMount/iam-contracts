package token

import (
	"context"

	"github.com/FangcunMount/iam/v4/internal/apiserver/domain/authn/authentication"
)

// AuthenticationGrantIssuer 在认证成功后颁发完整在线认证结果。
// 调用方无需感知 Session、access token 与 refresh token 的内部装配过程。
type AuthenticationGrantIssuer interface {
	IssueAuthentication(ctx context.Context, principal *authentication.Principal) (*TokenPair, error)
}

// Refresher 通过 refresh token 轮换在线会话令牌。
type Refresher interface {
	RefreshToken(ctx context.Context, refreshToken string) (*TokenRefreshResult, error)
}

// Revoker 撤销 bearer token 或 refresh token，并按令牌类型收敛关联会话状态。
type Revoker interface {
	// RevokeAccessToken 是已发布兼容名，可接收 access bearer token。
	RevokeAccessToken(ctx context.Context, accessToken string) error
	RevokeRefreshToken(ctx context.Context, refreshToken string) error
}

// Verifier 在线验证访问令牌及其可选 issuer / audience 约束。
type Verifier interface {
	VerifyToken(ctx context.Context, req VerifyTokenRequest) (*TokenVerifyResult, error)
}

// Capabilities 是组合根输出的令牌用例能力集合。
// 它只承载窄接口，不是供业务代码依赖的统一门面。
type Capabilities struct {
	AuthenticationGrantIssuer AuthenticationGrantIssuer
	Refresher                 Refresher
	Revoker                   Revoker
	Verifier                  Verifier
}

// ================== DTOs ==================

// TokenIssueResult 令牌签发结果 DTO。
type TokenIssueResult struct {
	TokenPair *TokenPair // 令牌对
}

// TokenRefreshResult 令牌刷新结果 DTO。
type TokenRefreshResult struct {
	TokenPair *TokenPair // 令牌对
}

// VerifyTokenRequest 令牌验证请求 DTO。
type VerifyTokenRequest struct {
	AccessToken        string      // 访问令牌
	ExpectedIssuer     string      // 预期签发者
	ExpectedAudience   []string    // 预期受众
	AcceptedTokenTypes []TokenType // 场景允许的令牌类型；为空时安全默认只接受 access
}

// TokenVerifyResult 令牌验证结果 DTO。
type TokenVerifyResult struct {
	Valid       bool         // 令牌是否有效
	Claims      *TokenClaims // 令牌声明
	FailureCode int
}

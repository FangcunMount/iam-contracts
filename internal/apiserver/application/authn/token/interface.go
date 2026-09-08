package token

import (
	"context"

	sessiondomain "github.com/FangcunMount/iam/v4/internal/apiserver/domain/authn/session"
)

// InitialTokenIssuer 在既有 Session 上签发初始令牌，并保存 RefreshToken。
// 不执行准入、创建会话或撤销会话；失败补偿属于调用用例。
type InitialTokenIssuer interface {
	IssueInitialTokens(ctx context.Context, sess *sessiondomain.Session) (*TokenPair, error)
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

// Verifier 在线验证访问令牌、必填 audience 和可选额外 issuer 约束。
type Verifier interface {
	VerifyToken(ctx context.Context, req VerifyTokenRequest) (*TokenVerifyResult, error)
}

// Capabilities 是组合根输出的令牌用例能力集合。
// 它只承载窄接口，不是供业务代码依赖的统一门面。
type Capabilities struct {
	InitialTokenIssuer InitialTokenIssuer
	Refresher          Refresher
	Revoker            Revoker
	Verifier           Verifier
}

// ================== DTOs ==================

// TokenRefreshResult 令牌刷新结果 DTO。
type TokenRefreshResult struct {
	TokenPair *TokenPair // 令牌对
}

// VerifyTokenRequest 令牌验证请求 DTO。
type VerifyTokenRequest struct {
	AccessToken        string      // 访问令牌
	ExpectedIssuer     string      // 预期签发者
	ExpectedAudience   []string    // 必填预期受众，任一匹配
	AcceptedTokenTypes []TokenType // 场景允许的令牌类型；为空时安全默认只接受 access
}

// TokenVerifyResult 令牌验证结果 DTO。
type TokenVerifyResult struct {
	Valid       bool         // 令牌是否有效
	Claims      *TokenClaims // 令牌声明
	FailureCode int
}

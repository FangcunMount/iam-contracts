package port

import (
	"context"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/authentication"
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/token"
)

// ==================== Driving Ports (驱动端口) ====================
// 这些接口由领域层（领域服务）实现，供应用层调用
// 按照功能职责拆分，遵循接口隔离原则

// TokenIssuer 令牌签发端口
type TokenIssuer interface {
	// IssueToken 签发访问令牌和刷新令牌
	//
	// 参数:
	//   - auth: 认证结果（包含 Principal 信息）
	//
	// 返回:
	//   - TokenPair: 访问令牌和刷新令牌对
	//   - err: 错误信息
	IssueToken(ctx context.Context, principal *authentication.Principal) (*domain.TokenPair, error)

	// RevokeToken 撤销令牌
	RevokeToken(ctx context.Context, tokenValue string) error
}

type TokenRefresher interface {
	// RefreshToken 刷新访问令牌
	//
	// 参数:
	//   - refreshToken: 刷新令牌字符串
	//
	// 返回:
	//   - TokenPair: 新的访问令牌和刷新令牌对
	//   - err: 错误信息
	RefreshToken(ctx context.Context, refreshTokenValue string) (*domain.TokenPair, error)

	// RevokeRefreshToken 撤销刷新令牌
	RevokeRefreshToken(ctx context.Context, refreshTokenValue string) error
}

type TokenVerifier interface {
	// VerifyAccessToken 验证访问令牌
	//
	// 参数:
	//   - accessToken: 访问令牌字符串
	//
	// 返回:
	//   - TokenClaims: 令牌声明信息
	//   - err: 错误信息
	VerifyAccessToken(ctx context.Context, tokenValue string) (*domain.TokenClaims, error)
}

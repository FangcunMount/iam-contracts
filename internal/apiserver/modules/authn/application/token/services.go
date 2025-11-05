package token

import (
	"context"

	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/token"
)

// ============= 应用服务接口（Driving Ports）=============

// TokenApplicationService 令牌应用服务 - 令牌管理
type TokenApplicationService interface {
	// RefreshToken 刷新访问令牌
	RefreshToken(ctx context.Context, refreshToken string) (*TokenRefreshResult, error)

	// RevokeToken 撤销访问令牌
	RevokeToken(ctx context.Context, accessToken string) error

	// RevokeRefreshToken 撤销刷新令牌
	RevokeRefreshToken(ctx context.Context, refreshToken string) error

	// VerifyToken 验证访问令牌
	VerifyToken(ctx context.Context, accessToken string) (*TokenVerifyResult, error)
}

// ============= DTOs =============

// TokenRefreshResult 令牌刷新结果DTO
type TokenRefreshResult struct {
	TokenPair *domain.TokenPair // 新的令牌对
}

// TokenVerifyResult 令牌验证结果DTO
type TokenVerifyResult struct {
	Valid  bool                // 是否有效
	Claims *domain.TokenClaims // 令牌声明（如果有效）
}

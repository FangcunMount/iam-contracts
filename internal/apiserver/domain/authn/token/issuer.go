package token

import (
	"context"
	"time"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/authentication"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
	"github.com/google/uuid"
)

// TokenIssuer 令牌颁发者
type TokenIssuer struct {
	tokenGenerator TokenGenerator // JWT 生成器
	tokenStore     TokenStore     // 令牌存储（Redis）
	accessTTL      time.Duration  // 访问令牌有效期
	refreshTTL     time.Duration  // 刷新令牌有效期
}

// NewTokenIssuer 创建令牌颁发者
func NewTokenIssuer(
	tokenGenerator TokenGenerator,
	tokenStore TokenStore,
	accessTTL time.Duration,
	refreshTTL time.Duration,
) *TokenIssuer {
	return &TokenIssuer{
		tokenGenerator: tokenGenerator,
		tokenStore:     tokenStore,
		accessTTL:      accessTTL,
		refreshTTL:     refreshTTL,
	}
}

// IssueToken 颁发令牌对
func (s *TokenIssuer) IssueToken(ctx context.Context, principal *authentication.Principal) (*TokenPair, error) {
	if principal == nil {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "principal is required")
	}

	// 生成访问令牌（JWT）
	accessToken, err := s.tokenGenerator.GenerateAccessToken(principal, s.accessTTL)
	if err != nil {
		return nil, perrors.WrapC(err, code.ErrInternalServerError, "failed to generate access token")
	}

	// 生成刷新令牌（UUID）
	refreshTokenValue := uuid.New().String()
	refreshToken := NewRefreshToken(
		uuid.New().String(),                     // token ID
		refreshTokenValue,                       // token value
		meta.NewID(uint64(principal.UserID)),    // user ID
		meta.NewID(uint64(principal.AccountID)), // account ID
		s.refreshTTL,
	)

	// 保存刷新令牌到 Redis
	if err := s.tokenStore.SaveRefreshToken(ctx, refreshToken); err != nil {
		return nil, perrors.WrapC(err, code.ErrInternalServerError, "failed to save refresh token")
	}

	return NewTokenPair(accessToken, refreshToken), nil
}

// RevokeToken 撤销令牌
//
// 将令牌加入黑名单，使其立即失效
func (s *TokenIssuer) RevokeToken(ctx context.Context, tokenValue string) error {
	// 解析令牌以获取 tokenID 和过期时间
	claims, err := s.tokenGenerator.ParseAccessToken(tokenValue)
	if err != nil {
		return perrors.WrapC(err, code.ErrTokenInvalid, "failed to parse token for revocation")
	}

	// 如果令牌已过期，无需加入黑名单
	if claims.IsExpired() {
		return nil
	}

	// 将令牌加入黑名单，TTL 设置为剩余有效期
	expiry := time.Until(claims.ExpiresAt)
	if expiry <= 0 {
		return nil // 已过期
	}

	if err := s.tokenStore.AddToBlacklist(ctx, claims.TokenID, expiry); err != nil {
		return perrors.WrapC(err, code.ErrInternalServerError, "failed to add token to blacklist")
	}

	return nil
}

package token

import (
	"context"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
)

// TokenVerifyer 令牌验证者
type TokenVerifyer struct {
	tokenGenerator TokenGenerator // JWT 生成器
	tokenStore     TokenStore     // 令牌存储（Redis）
}

// NewTokenVerifyer 创建令牌验证者
func NewTokenVerifyer(
	tokenGenerator TokenGenerator,
	tokenStore TokenStore,
) *TokenVerifyer {
	return &TokenVerifyer{
		tokenGenerator: tokenGenerator,
		tokenStore:     tokenStore,
	}
}

// VerifyAccessToken 验证访问令牌
func (s *TokenVerifyer) VerifyAccessToken(ctx context.Context, tokenValue string) (*TokenClaims, error) {
	// 解析 JWT
	claims, err := s.tokenGenerator.ParseAccessToken(ctx, tokenValue)
	if err != nil {
		return nil, perrors.WrapC(err, code.ErrTokenInvalid, "failed to parse access token")
	}

	// 检查过期
	if claims.IsExpired() {
		return nil, perrors.WithCode(code.ErrExpired, "access token has expired")
	}

	// 检查黑名单
	isBlacklisted, err := s.tokenStore.IsBlacklisted(ctx, claims.TokenID)
	if err != nil {
		return nil, perrors.WrapC(err, code.ErrInternalServerError, "failed to check token blacklist")
	}
	if isBlacklisted {
		return nil, perrors.WithCode(code.ErrTokenInvalid, "access token has been revoked")
	}

	return claims, nil
}

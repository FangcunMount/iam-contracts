package token

import (
	"context"
	"time"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/authentication"
	drivenPort "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/authentication/port/driven"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
)

// TokenRefresher 令牌刷新者
type TokenRefresher struct {
	tokenGenerator drivenPort.TokenGenerator // JWT 生成器
	tokenStore     drivenPort.TokenStore     // 令牌存储（Redis）
	accessTTL      time.Duration             // 访问令牌有效期
	refreshTTL     time.Duration             // 刷新令牌有效期
}

// NewTokenRefresher 创建令牌刷新者
func NewTokenRefresher(
	tokenGenerator drivenPort.TokenGenerator,
	tokenStore drivenPort.TokenStore,
	accessTTL time.Duration,
	refreshTTL time.Duration,
) *TokenRefresher {
	return &TokenRefresher{
		tokenGenerator: tokenGenerator,
		tokenStore:     tokenStore,
		accessTTL:      accessTTL,
		refreshTTL:     refreshTTL,
	}
}

// RefreshToken 刷新令牌
func (s *TokenRefresher) RefreshToken(ctx context.Context, refreshTokenValue string) (*authentication.TokenPair, error) {
	// 从 Redis 获取刷新令牌
	refreshToken, err := s.tokenStore.GetRefreshToken(ctx, refreshTokenValue)
	if err != nil {
		return nil, perrors.WrapC(err, code.ErrTokenInvalid, "refresh token not found or invalid")
	}

	if refreshToken == nil {
		return nil, perrors.WithCode(code.ErrTokenInvalid, "refresh token not found")
	}

	// 检查刷新令牌是否过期
	if refreshToken.IsExpired() {
		// 删除过期的刷新令牌
		_ = s.tokenStore.DeleteRefreshToken(ctx, refreshTokenValue)
		return nil, perrors.WithCode(code.ErrExpired, "refresh token has expired")
	}

	// 创建新的认证结果（从刷新令牌中恢复）
	auth := authentication.NewAuthentication(
		refreshToken.UserID,
		refreshToken.AccountID,
		"",  // provider 可以从 account 查询，这里暂时留空
		nil, // metadata 刷新时不需要
	)

	// 颁发新的令牌对
	newTokenPair, err := NewTokenIssuer(s.tokenGenerator, s.tokenStore, s.accessTTL, s.refreshTTL).IssueToken(ctx, auth)
	if err != nil {
		return nil, err
	}

	// 轮换刷新令牌：删除旧的刷新令牌
	if err := s.tokenStore.DeleteRefreshToken(ctx, refreshTokenValue); err != nil {
		// 记录错误但不影响主流程
		// TODO: 添加日志
	}

	return newTokenPair, nil
}

// RevokeRefreshToken 撤销刷新令牌
func (s *TokenRefresher) RevokeRefreshToken(ctx context.Context, refreshTokenValue string) error {
	if err := s.tokenStore.DeleteRefreshToken(ctx, refreshTokenValue); err != nil {
		return perrors.WrapC(err, code.ErrInternalServerError, "failed to revoke refresh token")
	}
	return nil
}

package token

import (
	"context"
	"time"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/component-base/pkg/logger"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/authentication"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
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
	l := logger.L(ctx)

	if principal == nil {
		l.Errorw("颁发令牌时 principal 为空",
			"action", logger.ActionCreate,
			"resource", "token",
		)
		return nil, perrors.WithCode(code.ErrInvalidArgument, "principal is required")
	}

	l.Debugw("开始颁发令牌对",
		"action", logger.ActionCreate,
		"resource", "token",
		"user_id", principal.UserID.String(),
		"account_id", principal.AccountID.String(),
	)

	// 生成访问令牌（JWT）
	l.Debugw("生成访问令牌",
		"action", logger.ActionCreate,
		"resource", "access_token",
		"ttl_seconds", s.accessTTL.Seconds(),
	)

	accessToken, err := s.tokenGenerator.GenerateAccessToken(ctx, principal, s.accessTTL)
	if err != nil {
		l.Errorw("访问令牌生成失败",
			"action", logger.ActionCreate,
			"resource", "access_token",
			"error", err.Error(),
		)
		return nil, perrors.WrapC(err, code.ErrInternalServerError, "failed to generate access token")
	}

	l.Debugw("访问令牌生成成功",
		"action", logger.ActionCreate,
		"resource", "access_token",
		"token_id", accessToken.ID,
	)

	// 生成刷新令牌（UUID）
	l.Debugw("生成刷新令牌",
		"action", logger.ActionCreate,
		"resource", "refresh_token",
		"ttl_seconds", s.refreshTTL.Seconds(),
	)

	refreshTokenValue := uuid.New().String()
	refreshToken := NewRefreshToken(
		uuid.New().String(), // token ID
		refreshTokenValue,   // token value
		principal.UserID,    // user ID
		principal.AccountID, // account ID
		s.refreshTTL,
	)

	// 保存刷新令牌到 Redis
	l.Debugw("保存刷新令牌到存储",
		"action", logger.ActionCreate,
		"resource", "refresh_token",
		"refresh_token_id", refreshToken.ID,
	)

	if err := s.tokenStore.SaveRefreshToken(ctx, refreshToken); err != nil {
		l.Errorw("刷新令牌保存失败",
			"action", logger.ActionCreate,
			"resource", "refresh_token",
			"error", err.Error(),
		)
		return nil, perrors.WrapC(err, code.ErrInternalServerError, "failed to save refresh token")
	}

	l.Debugw("令牌对颁发成功",
		"action", logger.ActionCreate,
		"resource", "token",
		"user_id", principal.UserID.String(),
		"access_token_id", accessToken.ID,
		"refresh_token_id", refreshToken.ID,
		"result", logger.ResultSuccess,
	)

	return NewTokenPair(accessToken, refreshToken), nil
}

// RevokeToken 撤销令牌
//
// 将令牌加入黑名单，使其立即失效
func (s *TokenIssuer) RevokeToken(ctx context.Context, tokenValue string) error {
	l := logger.L(ctx)

	l.Debugw("开始撤销访问令牌",
		"action", logger.ActionDelete,
		"resource", "access_token",
	)

	// 解析令牌以获取 tokenID 和过期时间
	claims, err := s.tokenGenerator.ParseAccessToken(ctx, tokenValue)
	if err != nil {
		l.Warnw("令牌解析失败",
			"action", logger.ActionDelete,
			"resource", "access_token",
			"error", err.Error(),
		)
		return perrors.WrapC(err, code.ErrTokenInvalid, "failed to parse token for revocation")
	}

	// 如果令牌已过期，无需加入黑名单
	if claims.IsExpired() {
		l.Debugw("令牌已过期，无需加入黑名单",
			"action", logger.ActionDelete,
			"resource", "access_token",
			"token_id", claims.TokenID,
		)
		return nil
	}

	// 将令牌加入黑名单，TTL 设置为剩余有效期
	expiry := time.Until(claims.ExpiresAt)
	if expiry <= 0 {
		l.Debugw("令牌已过期，无需加入黑名单",
			"action", logger.ActionDelete,
			"resource", "access_token",
			"token_id", claims.TokenID,
		)
		return nil // 已过期
	}

	l.Debugw("将令牌加入黑名单",
		"action", logger.ActionDelete,
		"resource", "access_token",
		"token_id", claims.TokenID,
		"remaining_ttl_seconds", expiry.Seconds(),
	)

	if err := s.tokenStore.AddToBlacklist(ctx, claims.TokenID, expiry); err != nil {
		l.Errorw("令牌加入黑名单失败",
			"action", logger.ActionDelete,
			"resource", "access_token",
			"token_id", claims.TokenID,
			"error", err.Error(),
		)
		return perrors.WrapC(err, code.ErrInternalServerError, "failed to add token to blacklist")
	}

	l.Debugw("访问令牌撤销成功",
		"action", logger.ActionDelete,
		"resource", "access_token",
		"token_id", claims.TokenID,
		"result", logger.ResultSuccess,
	)

	return nil
}

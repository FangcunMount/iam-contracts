package authentication

import (
	"context"
	"time"

	"github.com/google/uuid"

	perrors "github.com/fangcun-mount/iam-contracts/pkg/errors"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/authentication"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/authentication/port"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
)

// TokenService 令牌服务
type TokenService struct {
	tokenGenerator port.TokenGenerator // JWT 生成器
	tokenStore     port.TokenStore     // 令牌存储（Redis）
	accessTTL      time.Duration       // 访问令牌有效期
	refreshTTL     time.Duration       // 刷新令牌有效期
}

// TokenServiceOption 令牌服务配置选项
type TokenServiceOption func(*TokenService)

// WithAccessTTL 设置访问令牌有效期
func WithAccessTTL(ttl time.Duration) TokenServiceOption {
	return func(s *TokenService) {
		s.accessTTL = ttl
	}
}

// WithRefreshTTL 设置刷新令牌有效期
func WithRefreshTTL(ttl time.Duration) TokenServiceOption {
	return func(s *TokenService) {
		s.refreshTTL = ttl
	}
}

// NewTokenService 创建令牌服务
func NewTokenService(
	tokenGenerator port.TokenGenerator,
	tokenStore port.TokenStore,
	opts ...TokenServiceOption,
) *TokenService {
	s := &TokenService{
		tokenGenerator: tokenGenerator,
		tokenStore:     tokenStore,
		accessTTL:      15 * time.Minute,   // 默认 15 分钟
		refreshTTL:     7 * 24 * time.Hour, // 默认 7 天
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// IssueToken 颁发令牌对
func (s *TokenService) IssueToken(ctx context.Context, auth *authentication.Authentication) (*authentication.TokenPair, error) {
	if auth == nil {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "authentication is required")
	}

	// 生成访问令牌（JWT）
	accessToken, err := s.tokenGenerator.GenerateAccessToken(auth, s.accessTTL)
	if err != nil {
		return nil, perrors.WrapC(err, code.ErrInternalServerError, "failed to generate access token")
	}

	// 生成刷新令牌（UUID）
	refreshTokenValue := uuid.New().String()
	refreshToken := authentication.NewRefreshToken(
		uuid.New().String(), // token ID
		refreshTokenValue,
		auth.UserID,
		auth.AccountID,
		s.refreshTTL,
	)

	// 保存刷新令牌到 Redis
	if err := s.tokenStore.SaveRefreshToken(ctx, refreshToken); err != nil {
		return nil, perrors.WrapC(err, code.ErrInternalServerError, "failed to save refresh token")
	}

	return authentication.NewTokenPair(accessToken, refreshToken), nil
}

// VerifyAccessToken 验证访问令牌
func (s *TokenService) VerifyAccessToken(ctx context.Context, tokenValue string) (*authentication.TokenClaims, error) {
	// 解析 JWT
	claims, err := s.tokenGenerator.ParseAccessToken(tokenValue)
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

// RefreshToken 刷新令牌
func (s *TokenService) RefreshToken(ctx context.Context, refreshTokenValue string) (*authentication.TokenPair, error) {
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
	newTokenPair, err := s.IssueToken(ctx, auth)
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

// RevokeToken 撤销令牌
//
// 将令牌加入黑名单，使其立即失效
func (s *TokenService) RevokeToken(ctx context.Context, tokenValue string) error {
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

// RevokeRefreshToken 撤销刷新令牌
func (s *TokenService) RevokeRefreshToken(ctx context.Context, refreshTokenValue string) error {
	if err := s.tokenStore.DeleteRefreshToken(ctx, refreshTokenValue); err != nil {
		return perrors.WrapC(err, code.ErrInternalServerError, "failed to revoke refresh token")
	}
	return nil
}

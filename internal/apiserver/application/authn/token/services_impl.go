package token

import (
	"context"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	tokenPort "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/token/port"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
)

// ============= TokenApplicationService 实现 =============

type tokenApplicationService struct {
	tokenIssuer    tokenPort.TokenIssuer
	tokenRefresher tokenPort.TokenRefresher
	tokenVerifier  tokenPort.TokenVerifier
}

var _ TokenApplicationService = (*tokenApplicationService)(nil)

func NewTokenApplicationService(
	tokenIssuer tokenPort.TokenIssuer,
	tokenRefresher tokenPort.TokenRefresher,
	tokenVerifier tokenPort.TokenVerifier,
) TokenApplicationService {
	return &tokenApplicationService{
		tokenIssuer:    tokenIssuer,
		tokenRefresher: tokenRefresher,
		tokenVerifier:  tokenVerifier,
	}
}

// RefreshToken 刷新访问令牌
func (s *tokenApplicationService) RefreshToken(ctx context.Context, refreshToken string) (*TokenRefreshResult, error) {
	// 使用刷新令牌获取新的令牌对
	tokenPair, err := s.tokenRefresher.RefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, perrors.WithCode(code.ErrTokenInvalid, "failed to refresh token: %v", err)
	}

	return &TokenRefreshResult{
		TokenPair: tokenPair,
	}, nil
}

// RevokeToken 撤销访问令牌
func (s *tokenApplicationService) RevokeToken(ctx context.Context, accessToken string) error {
	err := s.tokenIssuer.RevokeToken(ctx, accessToken)
	if err != nil {
		return perrors.WithCode(code.ErrInvalidArgument, "failed to revoke token: %v", err)
	}
	return nil
}

// RevokeRefreshToken 撤销刷新令牌
func (s *tokenApplicationService) RevokeRefreshToken(ctx context.Context, refreshToken string) error {
	err := s.tokenRefresher.RevokeRefreshToken(ctx, refreshToken)
	if err != nil {
		return perrors.WithCode(code.ErrInvalidArgument, "failed to revoke refresh token: %v", err)
	}
	return nil
}

// VerifyToken 验证访问令牌
func (s *tokenApplicationService) VerifyToken(ctx context.Context, accessToken string) (*TokenVerifyResult, error) {
	// 验证访问令牌
	claims, err := s.tokenVerifier.VerifyAccessToken(ctx, accessToken)
	if err != nil {
		// 令牌无效
		return &TokenVerifyResult{
			Valid:  false,
			Claims: nil,
		}, nil
	}

	return &TokenVerifyResult{
		Valid:  true,
		Claims: claims,
	}, nil
}

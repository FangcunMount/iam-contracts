package token

import (
	"context"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/component-base/pkg/logger"
	tokenDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/token"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	"github.com/FangcunMount/iam-contracts/internal/pkg/security/sanitize"
)

// ============= TokenApplicationService 实现 =============

type tokenApplicationService struct {
	tokenIssuer    tokenDomain.Issuer
	tokenRefresher tokenDomain.Refresher
	tokenVerifier  tokenDomain.Verifier
}

var _ TokenApplicationService = (*tokenApplicationService)(nil)

func NewTokenApplicationService(
	tokenIssuer tokenDomain.Issuer,
	tokenRefresher tokenDomain.Refresher,
	tokenVerifier tokenDomain.Verifier,
) TokenApplicationService {
	return &tokenApplicationService{
		tokenIssuer:    tokenIssuer,
		tokenRefresher: tokenRefresher,
		tokenVerifier:  tokenVerifier,
	}
}

// RefreshToken 刷新访问令牌
func (s *tokenApplicationService) RefreshToken(ctx context.Context, refreshToken string) (*TokenRefreshResult, error) {
	l := logger.L(ctx)

	l.Debugw("开始刷新访问令牌",
		"action", logger.ActionRefresh,
		"resource", logger.ResourceToken,
		"token_hint", sanitize.MaskToken(refreshToken),
	)

	// 使用刷新令牌获取新的令牌对
	tokenPair, err := s.tokenRefresher.RefreshToken(ctx, refreshToken)
	if err != nil {
		l.Warnw("刷新令牌失败",
			"action", logger.ActionRefresh,
			"resource", logger.ResourceToken,
			"error", err.Error(),
			"result", logger.ResultFailed,
		)
		return nil, perrors.WithCode(code.ErrTokenInvalid, "failed to refresh token: %v", err)
	}

	l.Infow("访问令牌刷新成功",
		"action", logger.ActionRefresh,
		"resource", logger.ResourceToken,
		"result", logger.ResultSuccess,
	)

	return &TokenRefreshResult{
		TokenPair: tokenPair,
	}, nil
}

// RevokeToken 撤销访问令牌
func (s *tokenApplicationService) RevokeToken(ctx context.Context, accessToken string) error {
	l := logger.L(ctx)

	l.Debugw("开始撤销访问令牌",
		"action", logger.ActionRevoke,
		"resource", logger.ResourceToken,
		"token_hint", sanitize.MaskToken(accessToken),
	)

	err := s.tokenIssuer.RevokeToken(ctx, accessToken)
	if err != nil {
		l.Errorw("撤销访问令牌失败",
			"action", logger.ActionRevoke,
			"resource", logger.ResourceToken,
			"error", err.Error(),
			"result", logger.ResultFailed,
		)
		return perrors.WithCode(code.ErrInvalidArgument, "failed to revoke token: %v", err)
	}

	l.Infow("访问令牌撤销成功",
		"action", logger.ActionRevoke,
		"resource", logger.ResourceToken,
		"result", logger.ResultSuccess,
	)
	return nil
}

// RevokeRefreshToken 撤销刷新令牌
func (s *tokenApplicationService) RevokeRefreshToken(ctx context.Context, refreshToken string) error {
	l := logger.L(ctx)

	l.Debugw("开始撤销刷新令牌",
		"action", logger.ActionRevoke,
		"resource", logger.ResourceToken,
		"token_type", "refresh",
		"token_hint", sanitize.MaskToken(refreshToken),
	)

	err := s.tokenRefresher.RevokeRefreshToken(ctx, refreshToken)
	if err != nil {
		l.Errorw("撤销刷新令牌失败",
			"action", logger.ActionRevoke,
			"resource", logger.ResourceToken,
			"token_type", "refresh",
			"error", err.Error(),
			"result", logger.ResultFailed,
		)
		return perrors.WithCode(code.ErrInvalidArgument, "failed to revoke refresh token: %v", err)
	}

	l.Infow("刷新令牌撤销成功",
		"action", logger.ActionRevoke,
		"resource", logger.ResourceToken,
		"token_type", "refresh",
		"result", logger.ResultSuccess,
	)
	return nil
}

// VerifyToken 验证访问令牌
func (s *tokenApplicationService) VerifyToken(ctx context.Context, accessToken string) (*TokenVerifyResult, error) {
	l := logger.L(ctx)

	l.Debugw("开始验证访问令牌",
		"action", logger.ActionVerify,
		"resource", logger.ResourceToken,
		"token_hint", sanitize.MaskToken(accessToken),
	)

	// 验证访问令牌
	claims, err := s.tokenVerifier.VerifyAccessToken(ctx, accessToken)
	if err != nil {
		l.Warnw("访问令牌验证失败",
			"action", logger.ActionVerify,
			"resource", logger.ResourceToken,
			"error", err.Error(),
			"token_hint", sanitize.MaskToken(accessToken),
			"result", logger.ResultFailed,
		)
		// 令牌无效
		return &TokenVerifyResult{
			Valid:  false,
			Claims: nil,
		}, nil
	}

	l.Infow("访问令牌验证成功",
		"action", logger.ActionVerify,
		"resource", logger.ResourceToken,
		"user_id", claims.UserID.String(),
		"result", logger.ResultSuccess,
	)

	return &TokenVerifyResult{
		Valid:  true,
		Claims: claims,
	}, nil
}

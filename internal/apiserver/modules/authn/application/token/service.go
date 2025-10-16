// Package token 令牌应用服务
package token

import (
	"context"

	perrors "github.com/fangcun-mount/iam-contracts/pkg/errors"
	"github.com/fangcun-mount/iam-contracts/pkg/util/idutil"

	tokenService "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/authentication/service/token"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
)

// TokenService 令牌应用服务
type TokenService struct {
	tokenIssuer    *tokenService.TokenIssuer    // 令牌颁发者
	tokenRefresher *tokenService.TokenRefresher // 令牌刷新器
	tokenVerifyer  *tokenService.TokenVerifyer  // 令牌验证器
}

// NewTokenService 创建令牌应用服务
func NewTokenService(
	tokenIssuer *tokenService.TokenIssuer,
	tokenRefresher *tokenService.TokenRefresher,
	tokenVerifyer *tokenService.TokenVerifyer,
) *TokenService {
	return &TokenService{
		tokenIssuer:    tokenIssuer,
		tokenRefresher: tokenRefresher,
		tokenVerifyer:  tokenVerifyer,
	}
}

// VerifyTokenRequest 验证令牌请求
type VerifyTokenRequest struct {
	AccessToken string
}

// VerifyTokenResponse 验证令牌响应
type VerifyTokenResponse struct {
	Valid     bool   `json:"valid"`
	UserID    uint64 `json:"user_id"`
	AccountID uint64 `json:"account_id"`
	TokenID   string `json:"token_id"`
}

// VerifyToken 验证访问令牌
func (s *TokenService) VerifyToken(ctx context.Context, req *VerifyTokenRequest) (*VerifyTokenResponse, error) {
	if req == nil || req.AccessToken == "" {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "access token is required")
	}

	// 验证令牌
	claims, err := s.tokenVerifyer.VerifyAccessToken(ctx, req.AccessToken)
	if err != nil {
		// 如果是令牌无效或过期，返回 valid=false 而不是错误
		if perrors.IsCode(err, code.ErrTokenInvalid) || perrors.IsCode(err, code.ErrExpired) {
			return &VerifyTokenResponse{
				Valid: false,
			}, nil
		}
		return nil, err
	}

	// 构造响应
	return &VerifyTokenResponse{
		Valid:     true,
		UserID:    claims.UserID.Value(),
		AccountID: idutil.ID(claims.AccountID).Value(),
		TokenID:   claims.TokenID,
	}, nil
}

// RefreshTokenRequest 刷新令牌请求
type RefreshTokenRequest struct {
	RefreshToken string
}

// RefreshTokenResponse 刷新令牌响应
type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"` // 秒
}

// RefreshToken 刷新访问令牌
func (s *TokenService) RefreshToken(ctx context.Context, req *RefreshTokenRequest) (*RefreshTokenResponse, error) {
	if req == nil || req.RefreshToken == "" {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "refresh token is required")
	}

	// 刷新令牌
	tokenPair, err := s.tokenRefresher.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, err
	}

	// 构造响应
	return &RefreshTokenResponse{
		AccessToken:  tokenPair.AccessToken.Value,
		RefreshToken: tokenPair.RefreshToken.Value,
		TokenType:    "Bearer",
		ExpiresIn:    int64(tokenPair.AccessToken.RemainingDuration().Seconds()),
	}, nil
}

// LogoutRequest 登出请求
type LogoutRequest struct {
	AccessToken  string // 访问令牌
	RefreshToken string // 刷新令牌（可选）
}

// Logout 登出（撤销令牌）
func (s *TokenService) Logout(ctx context.Context, req *LogoutRequest) error {
	if req == nil || req.AccessToken == "" {
		return perrors.WithCode(code.ErrInvalidArgument, "access token is required")
	}

	// 撤销访问令牌（加入黑名单）
	if err := s.tokenIssuer.RevokeToken(ctx, req.AccessToken); err != nil {
		return err
	}

	// 如果提供了刷新令牌，也撤销它
	if req.RefreshToken != "" {
		if err := s.tokenRefresher.RevokeRefreshToken(ctx, req.RefreshToken); err != nil {
			// 刷新令牌撤销失败不影响主流程，记录日志即可
			// TODO: 添加日志
		}
	}

	return nil
}

// GetUserInfoRequest 获取用户信息请求
type GetUserInfoRequest struct {
	AccessToken string
}

// GetUserInfoResponse 获取用户信息响应
type GetUserInfoResponse struct {
	UserID    uint64 `json:"user_id"`
	AccountID uint64 `json:"account_id"`
	// 可以扩展更多用户信息（需要查询用户中心）
}

// GetUserInfo 从令牌中获取用户信息
func (s *TokenService) GetUserInfo(ctx context.Context, req *GetUserInfoRequest) (*GetUserInfoResponse, error) {
	if req == nil || req.AccessToken == "" {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "access token is required")
	}

	// 验证并解析令牌
	claims, err := s.tokenVerifyer.VerifyAccessToken(ctx, req.AccessToken)
	if err != nil {
		return nil, err
	}

	// 返回用户信息
	return &GetUserInfoResponse{
		UserID:    claims.UserID.Value(),
		AccountID: idutil.ID(claims.AccountID).Value(),
	}, nil
}

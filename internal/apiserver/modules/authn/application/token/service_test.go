package token

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/authentication"
	tokenService "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/authentication/service/token"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	perrors "github.com/fangcun-mount/iam-contracts/pkg/errors"
	"github.com/stretchr/testify/assert"
)

// MockTokenGenerator 模拟令牌生成器
type MockTokenGenerator struct {
	GenerateAccessTokenFunc func(auth *authentication.Authentication, ttl time.Duration) (*authentication.Token, error)
	ParseAccessTokenFunc    func(tokenValue string) (*authentication.TokenClaims, error)
}

func (m *MockTokenGenerator) GenerateAccessToken(auth *authentication.Authentication, ttl time.Duration) (*authentication.Token, error) {
	if m.GenerateAccessTokenFunc != nil {
		return m.GenerateAccessTokenFunc(auth, ttl)
	}
	return authentication.NewAccessToken("token-id", "mock-access-token", auth.UserID, auth.AccountID, ttl), nil
}

func (m *MockTokenGenerator) ParseAccessToken(tokenValue string) (*authentication.TokenClaims, error) {
	if m.ParseAccessTokenFunc != nil {
		return m.ParseAccessTokenFunc(tokenValue)
	}
	return &authentication.TokenClaims{
		TokenID:   "token-id",
		UserID:    account.UserID(123),
		AccountID: account.NewAccountID(456),
		ExpiresAt: time.Now().Add(time.Hour),
	}, nil
}

// MockTokenStore 模拟令牌存储
type MockTokenStore struct {
	SaveRefreshTokenFunc   func(ctx context.Context, token *authentication.Token) error
	GetRefreshTokenFunc    func(ctx context.Context, tokenValue string) (*authentication.Token, error)
	DeleteRefreshTokenFunc func(ctx context.Context, tokenValue string) error
	AddToBlacklistFunc     func(ctx context.Context, tokenID string, expiry time.Duration) error
	IsBlacklistedFunc      func(ctx context.Context, tokenID string) (bool, error)
}

func (m *MockTokenStore) SaveRefreshToken(ctx context.Context, token *authentication.Token) error {
	if m.SaveRefreshTokenFunc != nil {
		return m.SaveRefreshTokenFunc(ctx, token)
	}
	return nil
}

func (m *MockTokenStore) GetRefreshToken(ctx context.Context, tokenValue string) (*authentication.Token, error) {
	if m.GetRefreshTokenFunc != nil {
		return m.GetRefreshTokenFunc(ctx, tokenValue)
	}
	return nil, errors.New("not found")
}

func (m *MockTokenStore) DeleteRefreshToken(ctx context.Context, tokenValue string) error {
	if m.DeleteRefreshTokenFunc != nil {
		return m.DeleteRefreshTokenFunc(ctx, tokenValue)
	}
	return nil
}

func (m *MockTokenStore) AddToBlacklist(ctx context.Context, tokenID string, expiry time.Duration) error {
	if m.AddToBlacklistFunc != nil {
		return m.AddToBlacklistFunc(ctx, tokenID, expiry)
	}
	return nil
}

func (m *MockTokenStore) IsBlacklisted(ctx context.Context, tokenID string) (bool, error) {
	if m.IsBlacklistedFunc != nil {
		return m.IsBlacklistedFunc(ctx, tokenID)
	}
	return false, nil
}

// TestTokenService_VerifyToken_Success 测试验证令牌成功
func TestTokenService_VerifyToken_Success(t *testing.T) {
	mockTokenGen := &MockTokenGenerator{
		ParseAccessTokenFunc: func(tokenValue string) (*authentication.TokenClaims, error) {
			return &authentication.TokenClaims{
				TokenID:   "token-123",
				UserID:    account.UserID(12345),
				AccountID: account.NewAccountID(456),
				ExpiresAt: time.Now().Add(time.Hour),
			}, nil
		},
	}
	mockTokenStore := &MockTokenStore{
		IsBlacklistedFunc: func(ctx context.Context, tokenID string) (bool, error) {
			return false, nil // 令牌不在黑名单
		},
	}

	issuer := tokenService.NewTokenIssuer(mockTokenGen, mockTokenStore, 15*time.Minute, 7*24*time.Hour)
	refresher := tokenService.NewTokenRefresher(mockTokenGen, mockTokenStore, 15*time.Minute, 7*24*time.Hour)
	verifyer := tokenService.NewTokenVerifyer(mockTokenGen, mockTokenStore)

	service := NewTokenService(issuer, refresher, verifyer)
	ctx := context.Background()

	req := &VerifyTokenRequest{
		AccessToken: "valid_token",
	}

	resp, err := service.VerifyToken(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Valid)
	assert.Equal(t, uint64(12345), resp.UserID)
	assert.Equal(t, uint64(456), resp.AccountID)
	assert.Equal(t, "token-123", resp.TokenID)
}

// TestTokenService_VerifyToken_InvalidToken 测试验证无效令牌
func TestTokenService_VerifyToken_InvalidToken(t *testing.T) {
	mockTokenGen := &MockTokenGenerator{
		ParseAccessTokenFunc: func(tokenValue string) (*authentication.TokenClaims, error) {
			return nil, perrors.WithCode(code.ErrTokenInvalid, "invalid token")
		},
	}
	mockTokenStore := &MockTokenStore{}

	issuer := tokenService.NewTokenIssuer(mockTokenGen, mockTokenStore, 15*time.Minute, 7*24*time.Hour)
	refresher := tokenService.NewTokenRefresher(mockTokenGen, mockTokenStore, 15*time.Minute, 7*24*time.Hour)
	verifyer := tokenService.NewTokenVerifyer(mockTokenGen, mockTokenStore)

	service := NewTokenService(issuer, refresher, verifyer)
	ctx := context.Background()

	req := &VerifyTokenRequest{
		AccessToken: "invalid_token",
	}

	resp, err := service.VerifyToken(ctx, req)

	assert.NoError(t, err) // 无效令牌不返回错误，返回 valid=false
	assert.NotNil(t, resp)
	assert.False(t, resp.Valid)
}

// TestTokenService_VerifyToken_EmptyToken 测试空令牌
func TestTokenService_VerifyToken_EmptyToken(t *testing.T) {
	mockTokenGen := &MockTokenGenerator{}
	mockTokenStore := &MockTokenStore{}

	issuer := tokenService.NewTokenIssuer(mockTokenGen, mockTokenStore, 15*time.Minute, 7*24*time.Hour)
	refresher := tokenService.NewTokenRefresher(mockTokenGen, mockTokenStore, 15*time.Minute, 7*24*time.Hour)
	verifyer := tokenService.NewTokenVerifyer(mockTokenGen, mockTokenStore)

	service := NewTokenService(issuer, refresher, verifyer)
	ctx := context.Background()

	req := &VerifyTokenRequest{
		AccessToken: "",
	}

	resp, err := service.VerifyToken(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, perrors.IsCode(err, code.ErrInvalidArgument))
}

// TestTokenService_RefreshToken_Success 测试刷新令牌成功
func TestTokenService_RefreshToken_Success(t *testing.T) {
	auth := authentication.NewAuthentication(
		account.UserID(12345),
		account.NewAccountID(456),
		"basic",
		nil,
	)

	mockTokenGen := &MockTokenGenerator{
		ParseAccessTokenFunc: func(tokenValue string) (*authentication.TokenClaims, error) {
			return &authentication.TokenClaims{
				TokenID:   "refresh-token-123",
				UserID:    account.UserID(12345),
				AccountID: account.NewAccountID(456),
				ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
			}, nil
		},
		GenerateAccessTokenFunc: func(auth *authentication.Authentication, ttl time.Duration) (*authentication.Token, error) {
			return authentication.NewAccessToken("new-token-id", "new-access-token", auth.UserID, auth.AccountID, ttl), nil
		},
	}

	mockTokenStore := &MockTokenStore{
		IsBlacklistedFunc: func(ctx context.Context, tokenID string) (bool, error) {
			return false, nil
		},
		GetRefreshTokenFunc: func(ctx context.Context, tokenValue string) (*authentication.Token, error) {
			return authentication.NewRefreshToken("refresh-token-123", "refresh-token-value", auth.UserID, auth.AccountID, 7*24*time.Hour), nil
		},
		DeleteRefreshTokenFunc: func(ctx context.Context, tokenValue string) error {
			return nil
		},
		SaveRefreshTokenFunc: func(ctx context.Context, token *authentication.Token) error {
			return nil
		},
	}

	issuer := tokenService.NewTokenIssuer(mockTokenGen, mockTokenStore, 15*time.Minute, 7*24*time.Hour)
	refresher := tokenService.NewTokenRefresher(mockTokenGen, mockTokenStore, 15*time.Minute, 7*24*time.Hour)
	verifyer := tokenService.NewTokenVerifyer(mockTokenGen, mockTokenStore)

	service := NewTokenService(issuer, refresher, verifyer)
	ctx := context.Background()

	req := &RefreshTokenRequest{
		RefreshToken: "valid_refresh_token",
	}

	resp, err := service.RefreshToken(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "new-access-token", resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken) // 新的refresh token
	assert.Equal(t, "Bearer", resp.TokenType)
	assert.Greater(t, resp.ExpiresIn, int64(0))
}

// TestTokenService_RefreshToken_InvalidToken 测试刷新无效令牌
func TestTokenService_RefreshToken_InvalidToken(t *testing.T) {
	mockTokenGen := &MockTokenGenerator{
		ParseAccessTokenFunc: func(tokenValue string) (*authentication.TokenClaims, error) {
			return nil, perrors.WithCode(code.ErrTokenInvalid, "invalid token")
		},
	}
	mockTokenStore := &MockTokenStore{}

	issuer := tokenService.NewTokenIssuer(mockTokenGen, mockTokenStore, 15*time.Minute, 7*24*time.Hour)
	refresher := tokenService.NewTokenRefresher(mockTokenGen, mockTokenStore, 15*time.Minute, 7*24*time.Hour)
	verifyer := tokenService.NewTokenVerifyer(mockTokenGen, mockTokenStore)

	service := NewTokenService(issuer, refresher, verifyer)
	ctx := context.Background()

	req := &RefreshTokenRequest{
		RefreshToken: "invalid_refresh_token",
	}

	resp, err := service.RefreshToken(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
}

// TestTokenService_RefreshToken_EmptyToken 测试空刷新令牌
func TestTokenService_RefreshToken_EmptyToken(t *testing.T) {
	mockTokenGen := &MockTokenGenerator{}
	mockTokenStore := &MockTokenStore{}

	issuer := tokenService.NewTokenIssuer(mockTokenGen, mockTokenStore, 15*time.Minute, 7*24*time.Hour)
	refresher := tokenService.NewTokenRefresher(mockTokenGen, mockTokenStore, 15*time.Minute, 7*24*time.Hour)
	verifyer := tokenService.NewTokenVerifyer(mockTokenGen, mockTokenStore)

	service := NewTokenService(issuer, refresher, verifyer)
	ctx := context.Background()

	req := &RefreshTokenRequest{
		RefreshToken: "",
	}

	resp, err := service.RefreshToken(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, perrors.IsCode(err, code.ErrInvalidArgument))
}

// TestTokenService_Logout_Success 测试登出成功
func TestTokenService_Logout_Success(t *testing.T) {
	revoked := false
	mockTokenGen := &MockTokenGenerator{
		ParseAccessTokenFunc: func(tokenValue string) (*authentication.TokenClaims, error) {
			return &authentication.TokenClaims{
				TokenID:   "token-123",
				UserID:    account.UserID(12345),
				AccountID: account.NewAccountID(456),
				ExpiresAt: time.Now().Add(time.Hour),
			}, nil
		},
	}
	mockTokenStore := &MockTokenStore{
		AddToBlacklistFunc: func(ctx context.Context, tokenID string, expiry time.Duration) error {
			revoked = true
			return nil
		},
	}

	issuer := tokenService.NewTokenIssuer(mockTokenGen, mockTokenStore, 15*time.Minute, 7*24*time.Hour)
	refresher := tokenService.NewTokenRefresher(mockTokenGen, mockTokenStore, 15*time.Minute, 7*24*time.Hour)
	verifyer := tokenService.NewTokenVerifyer(mockTokenGen, mockTokenStore)

	service := NewTokenService(issuer, refresher, verifyer)
	ctx := context.Background()

	req := &LogoutRequest{
		AccessToken: "valid_token",
	}

	err := service.Logout(ctx, req)

	assert.NoError(t, err)
	assert.True(t, revoked)
}

// TestTokenService_Logout_EmptyToken 测试空令牌登出
func TestTokenService_Logout_EmptyToken(t *testing.T) {
	mockTokenGen := &MockTokenGenerator{}
	mockTokenStore := &MockTokenStore{}

	issuer := tokenService.NewTokenIssuer(mockTokenGen, mockTokenStore, 15*time.Minute, 7*24*time.Hour)
	refresher := tokenService.NewTokenRefresher(mockTokenGen, mockTokenStore, 15*time.Minute, 7*24*time.Hour)
	verifyer := tokenService.NewTokenVerifyer(mockTokenGen, mockTokenStore)

	service := NewTokenService(issuer, refresher, verifyer)
	ctx := context.Background()

	req := &LogoutRequest{
		AccessToken: "",
	}

	err := service.Logout(ctx, req)

	assert.Error(t, err)
	assert.True(t, perrors.IsCode(err, code.ErrInvalidArgument))
}

// TestTokenService_Logout_RevokeFailed 测试撤销失败
func TestTokenService_Logout_RevokeFailed(t *testing.T) {
	mockTokenGen := &MockTokenGenerator{
		ParseAccessTokenFunc: func(tokenValue string) (*authentication.TokenClaims, error) {
			return &authentication.TokenClaims{
				TokenID:   "token-123",
				UserID:    account.UserID(12345),
				AccountID: account.NewAccountID(456),
				ExpiresAt: time.Now().Add(time.Hour),
			}, nil
		},
	}
	mockTokenStore := &MockTokenStore{
		AddToBlacklistFunc: func(ctx context.Context, tokenID string, expiry time.Duration) error {
			return errors.New("redis error")
		},
	}

	issuer := tokenService.NewTokenIssuer(mockTokenGen, mockTokenStore, 15*time.Minute, 7*24*time.Hour)
	refresher := tokenService.NewTokenRefresher(mockTokenGen, mockTokenStore, 15*time.Minute, 7*24*time.Hour)
	verifyer := tokenService.NewTokenVerifyer(mockTokenGen, mockTokenStore)

	service := NewTokenService(issuer, refresher, verifyer)
	ctx := context.Background()

	req := &LogoutRequest{
		AccessToken: "valid_token",
	}

	err := service.Logout(ctx, req)

	assert.Error(t, err)
}

// TestTokenService_GetUserInfo_Success 测试获取用户信息成功
func TestTokenService_GetUserInfo_Success(t *testing.T) {
	mockTokenGen := &MockTokenGenerator{
		ParseAccessTokenFunc: func(tokenValue string) (*authentication.TokenClaims, error) {
			return &authentication.TokenClaims{
				TokenID:   "token-123",
				UserID:    account.UserID(12345),
				AccountID: account.NewAccountID(456),
				ExpiresAt: time.Now().Add(time.Hour),
			}, nil
		},
	}
	mockTokenStore := &MockTokenStore{
		IsBlacklistedFunc: func(ctx context.Context, tokenID string) (bool, error) {
			return false, nil
		},
	}

	issuer := tokenService.NewTokenIssuer(mockTokenGen, mockTokenStore, 15*time.Minute, 7*24*time.Hour)
	refresher := tokenService.NewTokenRefresher(mockTokenGen, mockTokenStore, 15*time.Minute, 7*24*time.Hour)
	verifyer := tokenService.NewTokenVerifyer(mockTokenGen, mockTokenStore)

	service := NewTokenService(issuer, refresher, verifyer)
	ctx := context.Background()

	req := &GetUserInfoRequest{
		AccessToken: "valid_token",
	}

	resp, err := service.GetUserInfo(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, uint64(12345), resp.UserID)
	assert.Equal(t, uint64(456), resp.AccountID)
}

// TestTokenService_GetUserInfo_InvalidToken 测试获取用户信息-无效令牌
func TestTokenService_GetUserInfo_InvalidToken(t *testing.T) {
	mockTokenGen := &MockTokenGenerator{
		ParseAccessTokenFunc: func(tokenValue string) (*authentication.TokenClaims, error) {
			return nil, perrors.WithCode(code.ErrTokenInvalid, "invalid token")
		},
	}
	mockTokenStore := &MockTokenStore{}

	issuer := tokenService.NewTokenIssuer(mockTokenGen, mockTokenStore, 15*time.Minute, 7*24*time.Hour)
	refresher := tokenService.NewTokenRefresher(mockTokenGen, mockTokenStore, 15*time.Minute, 7*24*time.Hour)
	verifyer := tokenService.NewTokenVerifyer(mockTokenGen, mockTokenStore)

	service := NewTokenService(issuer, refresher, verifyer)
	ctx := context.Background()

	req := &GetUserInfoRequest{
		AccessToken: "invalid_token",
	}

	resp, err := service.GetUserInfo(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, perrors.IsCode(err, code.ErrTokenInvalid))
}

// TestTokenService_GetUserInfo_EmptyToken 测试获取用户信息-空令牌
func TestTokenService_GetUserInfo_EmptyToken(t *testing.T) {
	mockTokenGen := &MockTokenGenerator{}
	mockTokenStore := &MockTokenStore{}

	issuer := tokenService.NewTokenIssuer(mockTokenGen, mockTokenStore, 15*time.Minute, 7*24*time.Hour)
	refresher := tokenService.NewTokenRefresher(mockTokenGen, mockTokenStore, 15*time.Minute, 7*24*time.Hour)
	verifyer := tokenService.NewTokenVerifyer(mockTokenGen, mockTokenStore)

	service := NewTokenService(issuer, refresher, verifyer)
	ctx := context.Background()

	req := &GetUserInfoRequest{
		AccessToken: "",
	}

	resp, err := service.GetUserInfo(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, perrors.IsCode(err, code.ErrInvalidArgument))
}

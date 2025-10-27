package token

import (
	"context"
	"errors"
	"testing"
	"time"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/authentication"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
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
	return authentication.NewAccessToken("token-id-123", "mock-access-token", auth.UserID, auth.AccountID, ttl), nil
}

func (m *MockTokenGenerator) ParseAccessToken(tokenValue string) (*authentication.TokenClaims, error) {
	if m.ParseAccessTokenFunc != nil {
		return m.ParseAccessTokenFunc(tokenValue)
	}
	return &authentication.TokenClaims{
		TokenID:   "token-id-123",
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

// TestTokenIssuer_IssueToken_Success 测试成功颁发令牌
func TestTokenIssuer_IssueToken_Success(t *testing.T) {
	// Arrange
	mockGenerator := &MockTokenGenerator{
		GenerateAccessTokenFunc: func(auth *authentication.Authentication, ttl time.Duration) (*authentication.Token, error) {
			assert.Equal(t, account.UserID(123), auth.UserID)
			assert.Equal(t, account.NewAccountID(456), auth.AccountID)
			assert.Equal(t, 15*time.Minute, ttl)
			return authentication.NewAccessToken("token-id-xyz", "mock-access-token-xyz", auth.UserID, auth.AccountID, ttl), nil
		},
	}

	mockStore := &MockTokenStore{
		SaveRefreshTokenFunc: func(ctx context.Context, token *authentication.Token) error {
			assert.Equal(t, account.UserID(123), token.UserID)
			assert.Equal(t, account.NewAccountID(456), token.AccountID)
			assert.NotEmpty(t, token.Value)
			return nil
		},
	}

	issuer := NewTokenIssuer(mockGenerator, mockStore, 15*time.Minute, 7*24*time.Hour)

	auth := authentication.NewAuthentication(
		account.UserID(123),
		account.NewAccountID(456),
		"basic",
		nil,
	)

	ctx := context.Background()

	// Act
	tokenPair, err := issuer.IssueToken(ctx, auth)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, tokenPair)
	assert.Equal(t, "mock-access-token-xyz", tokenPair.AccessToken.Value)
	assert.NotEmpty(t, tokenPair.RefreshToken.Value)
	assert.Equal(t, account.UserID(123), tokenPair.RefreshToken.UserID)
	assert.Equal(t, account.NewAccountID(456), tokenPair.RefreshToken.AccountID)
}

// TestTokenIssuer_IssueToken_NilAuthentication 测试传入空认证
func TestTokenIssuer_IssueToken_NilAuthentication(t *testing.T) {
	// Arrange
	mockGenerator := &MockTokenGenerator{}
	mockStore := &MockTokenStore{}
	issuer := NewTokenIssuer(mockGenerator, mockStore, 15*time.Minute, 7*24*time.Hour)
	ctx := context.Background()

	// Act
	tokenPair, err := issuer.IssueToken(ctx, nil)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, tokenPair)
	assert.True(t, perrors.IsCode(err, code.ErrInvalidArgument))
}

// TestTokenIssuer_IssueToken_GenerateAccessTokenFailed 测试生成访问令牌失败
func TestTokenIssuer_IssueToken_GenerateAccessTokenFailed(t *testing.T) {
	// Arrange
	mockGenerator := &MockTokenGenerator{
		GenerateAccessTokenFunc: func(auth *authentication.Authentication, ttl time.Duration) (*authentication.Token, error) {
			return nil, errors.New("jwt generation failed")
		},
	}

	mockStore := &MockTokenStore{}
	issuer := NewTokenIssuer(mockGenerator, mockStore, 15*time.Minute, 7*24*time.Hour)

	auth := authentication.NewAuthentication(account.UserID(123), account.NewAccountID(456), "basic", nil)
	ctx := context.Background()

	// Act
	tokenPair, err := issuer.IssueToken(ctx, auth)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, tokenPair)
	assert.True(t, perrors.IsCode(err, code.ErrInternalServerError))
}

// TestTokenIssuer_IssueToken_SaveRefreshTokenFailed 测试保存刷新令牌失败
func TestTokenIssuer_IssueToken_SaveRefreshTokenFailed(t *testing.T) {
	// Arrange
	mockGenerator := &MockTokenGenerator{
		GenerateAccessTokenFunc: func(auth *authentication.Authentication, ttl time.Duration) (*authentication.Token, error) {
			return authentication.NewAccessToken("token-id", "mock-access-token", auth.UserID, auth.AccountID, ttl), nil
		},
	}

	mockStore := &MockTokenStore{
		SaveRefreshTokenFunc: func(ctx context.Context, token *authentication.Token) error {
			return errors.New("redis connection failed")
		},
	}

	issuer := NewTokenIssuer(mockGenerator, mockStore, 15*time.Minute, 7*24*time.Hour)
	auth := authentication.NewAuthentication(account.UserID(123), account.NewAccountID(456), "basic", nil)
	ctx := context.Background()

	// Act
	tokenPair, err := issuer.IssueToken(ctx, auth)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, tokenPair)
	assert.True(t, perrors.IsCode(err, code.ErrInternalServerError))
}

// TestTokenIssuer_RevokeToken_Success 测试成功撤销令牌
func TestTokenIssuer_RevokeToken_Success(t *testing.T) {
	// Arrange
	mockGenerator := &MockTokenGenerator{
		ParseAccessTokenFunc: func(tokenValue string) (*authentication.TokenClaims, error) {
			return &authentication.TokenClaims{
				TokenID:   "token-id-789",
				UserID:    account.UserID(123),
				AccountID: account.NewAccountID(456),
				ExpiresAt: time.Now().Add(time.Hour),
			}, nil
		},
	}

	mockStore := &MockTokenStore{
		AddToBlacklistFunc: func(ctx context.Context, tokenID string, expiry time.Duration) error {
			assert.Equal(t, "token-id-789", tokenID)
			assert.True(t, expiry > 0)
			return nil
		},
	}

	issuer := NewTokenIssuer(mockGenerator, mockStore, 15*time.Minute, 7*24*time.Hour)
	ctx := context.Background()

	// Act
	err := issuer.RevokeToken(ctx, "valid-access-token")

	// Assert
	assert.NoError(t, err)
}

// TestTokenIssuer_RevokeToken_ParseTokenFailed 测试解析令牌失败
func TestTokenIssuer_RevokeToken_ParseTokenFailed(t *testing.T) {
	// Arrange
	mockGenerator := &MockTokenGenerator{
		ParseAccessTokenFunc: func(tokenValue string) (*authentication.TokenClaims, error) {
			return nil, errors.New("invalid token format")
		},
	}

	mockStore := &MockTokenStore{}
	issuer := NewTokenIssuer(mockGenerator, mockStore, 15*time.Minute, 7*24*time.Hour)
	ctx := context.Background()

	// Act
	err := issuer.RevokeToken(ctx, "invalid-token")

	// Assert
	assert.Error(t, err)
	assert.True(t, perrors.IsCode(err, code.ErrTokenInvalid))
}

// TestTokenIssuer_RevokeToken_AddToBlacklistFailed 测试添加到黑名单失败
func TestTokenIssuer_RevokeToken_AddToBlacklistFailed(t *testing.T) {
	// Arrange
	mockGenerator := &MockTokenGenerator{
		ParseAccessTokenFunc: func(tokenValue string) (*authentication.TokenClaims, error) {
			return &authentication.TokenClaims{
				TokenID:   "token-id-999",
				ExpiresAt: time.Now().Add(time.Hour),
			}, nil
		},
	}

	mockStore := &MockTokenStore{
		AddToBlacklistFunc: func(ctx context.Context, tokenID string, expiry time.Duration) error {
			return errors.New("redis write failed")
		},
	}

	issuer := NewTokenIssuer(mockGenerator, mockStore, 15*time.Minute, 7*24*time.Hour)
	ctx := context.Background()

	// Act
	err := issuer.RevokeToken(ctx, "valid-token")

	// Assert
	assert.Error(t, err)
	assert.True(t, perrors.IsCode(err, code.ErrInternalServerError))
}

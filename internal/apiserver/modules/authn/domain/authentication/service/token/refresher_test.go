package token

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/authentication"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	perrors "github.com/FangcunMount/iam-contracts/pkg/errors"
	"github.com/stretchr/testify/assert"
)

// TestTokenRefresher_RefreshToken_Success 测试成功刷新令牌
func TestTokenRefresher_RefreshToken_Success(t *testing.T) {
	// Arrange
	oldRefreshToken := authentication.NewRefreshToken(
		"old-refresh-token-id",
		"old-refresh-token-value",
		account.UserID(123),
		account.NewAccountID(456),
		7*24*time.Hour,
	)

	mockGenerator := &MockTokenGenerator{
		GenerateAccessTokenFunc: func(auth *authentication.Authentication, ttl time.Duration) (*authentication.Token, error) {
			return authentication.NewAccessToken("new-access-token-id", "new-access-token", auth.UserID, auth.AccountID, ttl), nil
		},
	}

	var deleteCalled bool
	mockStore := &MockTokenStore{
		GetRefreshTokenFunc: func(ctx context.Context, tokenValue string) (*authentication.Token, error) {
			assert.Equal(t, "old-refresh-token-value", tokenValue)
			return oldRefreshToken, nil
		},
		SaveRefreshTokenFunc: func(ctx context.Context, token *authentication.Token) error {
			assert.Equal(t, account.UserID(123), token.UserID)
			assert.Equal(t, account.NewAccountID(456), token.AccountID)
			return nil
		},
		DeleteRefreshTokenFunc: func(ctx context.Context, tokenValue string) error {
			assert.Equal(t, "old-refresh-token-value", tokenValue)
			deleteCalled = true
			return nil
		},
	}

	refresher := NewTokenRefresher(mockGenerator, mockStore, 15*time.Minute, 7*24*time.Hour)
	ctx := context.Background()

	// Act
	tokenPair, err := refresher.RefreshToken(ctx, "old-refresh-token-value")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, tokenPair)
	assert.Equal(t, "new-access-token", tokenPair.AccessToken.Value)
	assert.NotEmpty(t, tokenPair.RefreshToken.Value)
	assert.True(t, deleteCalled, "旧刷新令牌应被删除")
}

// TestTokenRefresher_RefreshToken_NotFound 测试刷新令牌不存在
func TestTokenRefresher_RefreshToken_NotFound(t *testing.T) {
	// Arrange
	mockGenerator := &MockTokenGenerator{}
	mockStore := &MockTokenStore{
		GetRefreshTokenFunc: func(ctx context.Context, tokenValue string) (*authentication.Token, error) {
			return nil, errors.New("token not found")
		},
	}

	refresher := NewTokenRefresher(mockGenerator, mockStore, 15*time.Minute, 7*24*time.Hour)
	ctx := context.Background()

	// Act
	tokenPair, err := refresher.RefreshToken(ctx, "non-existent-token")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, tokenPair)
	assert.True(t, perrors.IsCode(err, code.ErrTokenInvalid))
}

// TestTokenRefresher_RefreshToken_ReturnsNil 测试返回空令牌
func TestTokenRefresher_RefreshToken_ReturnsNil(t *testing.T) {
	// Arrange
	mockGenerator := &MockTokenGenerator{}
	mockStore := &MockTokenStore{
		GetRefreshTokenFunc: func(ctx context.Context, tokenValue string) (*authentication.Token, error) {
			return nil, nil // 明确返回 nil
		},
	}

	refresher := NewTokenRefresher(mockGenerator, mockStore, 15*time.Minute, 7*24*time.Hour)
	ctx := context.Background()

	// Act
	tokenPair, err := refresher.RefreshToken(ctx, "some-token")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, tokenPair)
	assert.True(t, perrors.IsCode(err, code.ErrTokenInvalid))
}

// TestTokenRefresher_RefreshToken_Expired 测试刷新令牌已过期
func TestTokenRefresher_RefreshToken_Expired(t *testing.T) {
	// Arrange
	expiredToken := authentication.NewRefreshToken(
		"expired-token-id",
		"expired-token-value",
		account.UserID(123),
		account.NewAccountID(456),
		-time.Hour, // 已过期
	)

	mockGenerator := &MockTokenGenerator{}
	var deleteCalled bool
	mockStore := &MockTokenStore{
		GetRefreshTokenFunc: func(ctx context.Context, tokenValue string) (*authentication.Token, error) {
			return expiredToken, nil
		},
		DeleteRefreshTokenFunc: func(ctx context.Context, tokenValue string) error {
			assert.Equal(t, "expired-token-value", tokenValue)
			deleteCalled = true
			return nil
		},
	}

	refresher := NewTokenRefresher(mockGenerator, mockStore, 15*time.Minute, 7*24*time.Hour)
	ctx := context.Background()

	// Act
	tokenPair, err := refresher.RefreshToken(ctx, "expired-token-value")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, tokenPair)
	assert.True(t, perrors.IsCode(err, code.ErrExpired))
	assert.True(t, deleteCalled, "过期的刷新令牌应被删除")
}

// TestTokenRefresher_RevokeRefreshToken_Success 测试成功撤销刷新令牌
func TestTokenRefresher_RevokeRefreshToken_Success(t *testing.T) {
	// Arrange
	mockGenerator := &MockTokenGenerator{}
	mockStore := &MockTokenStore{
		DeleteRefreshTokenFunc: func(ctx context.Context, tokenValue string) error {
			assert.Equal(t, "token-to-revoke", tokenValue)
			return nil
		},
	}

	refresher := NewTokenRefresher(mockGenerator, mockStore, 15*time.Minute, 7*24*time.Hour)
	ctx := context.Background()

	// Act
	err := refresher.RevokeRefreshToken(ctx, "token-to-revoke")

	// Assert
	assert.NoError(t, err)
}

// TestTokenRefresher_RevokeRefreshToken_Failed 测试撤销刷新令牌失败
func TestTokenRefresher_RevokeRefreshToken_Failed(t *testing.T) {
	// Arrange
	mockGenerator := &MockTokenGenerator{}
	mockStore := &MockTokenStore{
		DeleteRefreshTokenFunc: func(ctx context.Context, tokenValue string) error {
			return errors.New("redis connection failed")
		},
	}

	refresher := NewTokenRefresher(mockGenerator, mockStore, 15*time.Minute, 7*24*time.Hour)
	ctx := context.Background()

	// Act
	err := refresher.RevokeRefreshToken(ctx, "some-token")

	// Assert
	assert.Error(t, err)
	assert.True(t, perrors.IsCode(err, code.ErrInternalServerError))
}

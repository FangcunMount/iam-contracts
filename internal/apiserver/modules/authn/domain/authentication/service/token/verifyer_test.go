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

// TestTokenVerifyer_VerifyAccessToken_Success 测试成功验证访问令牌
func TestTokenVerifyer_VerifyAccessToken_Success(t *testing.T) {
	// Arrange
	mockGenerator := &MockTokenGenerator{
		ParseAccessTokenFunc: func(tokenValue string) (*authentication.TokenClaims, error) {
			return &authentication.TokenClaims{
				TokenID:   "token-id-123",
				UserID:    account.UserID(123),
				AccountID: account.NewAccountID(456),
				IssuedAt:  time.Now().Add(-10 * time.Minute),
				ExpiresAt: time.Now().Add(5 * time.Minute), // 还未过期
			}, nil
		},
	}

	mockStore := &MockTokenStore{
		IsBlacklistedFunc: func(ctx context.Context, tokenID string) (bool, error) {
			assert.Equal(t, "token-id-123", tokenID)
			return false, nil // 不在黑名单中
		},
	}

	verifier := NewTokenVerifyer(mockGenerator, mockStore)
	ctx := context.Background()

	// Act
	claims, err := verifier.VerifyAccessToken(ctx, "valid-access-token")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, "token-id-123", claims.TokenID)
	assert.Equal(t, account.UserID(123), claims.UserID)
	assert.Equal(t, account.NewAccountID(456), claims.AccountID)
}

// TestTokenVerifyer_VerifyAccessToken_ParseFailed 测试解析令牌失败
func TestTokenVerifyer_VerifyAccessToken_ParseFailed(t *testing.T) {
	// Arrange
	mockGenerator := &MockTokenGenerator{
		ParseAccessTokenFunc: func(tokenValue string) (*authentication.TokenClaims, error) {
			return nil, errors.New("invalid jwt format")
		},
	}

	mockStore := &MockTokenStore{}
	verifier := NewTokenVerifyer(mockGenerator, mockStore)
	ctx := context.Background()

	// Act
	claims, err := verifier.VerifyAccessToken(ctx, "invalid-token")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.True(t, perrors.IsCode(err, code.ErrTokenInvalid))
}

// TestTokenVerifyer_VerifyAccessToken_Expired 测试令牌已过期
func TestTokenVerifyer_VerifyAccessToken_Expired(t *testing.T) {
	// Arrange
	mockGenerator := &MockTokenGenerator{
		ParseAccessTokenFunc: func(tokenValue string) (*authentication.TokenClaims, error) {
			return &authentication.TokenClaims{
				TokenID:   "token-id-expired",
				UserID:    account.UserID(123),
				AccountID: account.NewAccountID(456),
				IssuedAt:  time.Now().Add(-2 * time.Hour),
				ExpiresAt: time.Now().Add(-time.Hour), // 已过期
			}, nil
		},
	}

	mockStore := &MockTokenStore{}
	verifier := NewTokenVerifyer(mockGenerator, mockStore)
	ctx := context.Background()

	// Act
	claims, err := verifier.VerifyAccessToken(ctx, "expired-token")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.True(t, perrors.IsCode(err, code.ErrExpired))
}

// TestTokenVerifyer_VerifyAccessToken_Blacklisted 测试令牌在黑名单中
func TestTokenVerifyer_VerifyAccessToken_Blacklisted(t *testing.T) {
	// Arrange
	mockGenerator := &MockTokenGenerator{
		ParseAccessTokenFunc: func(tokenValue string) (*authentication.TokenClaims, error) {
			return &authentication.TokenClaims{
				TokenID:   "token-id-blacklisted",
				UserID:    account.UserID(123),
				AccountID: account.NewAccountID(456),
				IssuedAt:  time.Now().Add(-10 * time.Minute),
				ExpiresAt: time.Now().Add(5 * time.Minute),
			}, nil
		},
	}

	mockStore := &MockTokenStore{
		IsBlacklistedFunc: func(ctx context.Context, tokenID string) (bool, error) {
			return true, nil // 在黑名单中
		},
	}

	verifier := NewTokenVerifyer(mockGenerator, mockStore)
	ctx := context.Background()

	// Act
	claims, err := verifier.VerifyAccessToken(ctx, "blacklisted-token")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.True(t, perrors.IsCode(err, code.ErrTokenInvalid))
}

// TestTokenVerifyer_VerifyAccessToken_BlacklistCheckFailed 测试检查黑名单失败
func TestTokenVerifyer_VerifyAccessToken_BlacklistCheckFailed(t *testing.T) {
	// Arrange
	mockGenerator := &MockTokenGenerator{
		ParseAccessTokenFunc: func(tokenValue string) (*authentication.TokenClaims, error) {
			return &authentication.TokenClaims{
				TokenID:   "token-id-check-failed",
				UserID:    account.UserID(123),
				AccountID: account.NewAccountID(456),
				IssuedAt:  time.Now().Add(-10 * time.Minute),
				ExpiresAt: time.Now().Add(5 * time.Minute),
			}, nil
		},
	}

	mockStore := &MockTokenStore{
		IsBlacklistedFunc: func(ctx context.Context, tokenID string) (bool, error) {
			return false, errors.New("redis connection failed")
		},
	}

	verifier := NewTokenVerifyer(mockGenerator, mockStore)
	ctx := context.Background()

	// Act
	claims, err := verifier.VerifyAccessToken(ctx, "some-token")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.True(t, perrors.IsCode(err, code.ErrInternalServerError))
}

package authentication

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/account"
)

// ==================== AccessToken 测试 ====================

func TestNewAccessToken_Success(t *testing.T) {
	// Arrange
	id := "token_123"
	value := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."
	userID := account.NewUserID(12345)
	accountID := account.NewAccountID(67890)
	expiresIn := 1 * time.Hour

	// Act
	token := NewAccessToken(id, value, userID, accountID, expiresIn)

	// Assert
	require.NotNil(t, token)
	assert.Equal(t, id, token.ID)
	assert.Equal(t, TokenTypeAccess, token.Type)
	assert.Equal(t, value, token.Value)
	assert.Equal(t, userID, token.UserID)
	assert.Equal(t, accountID, token.AccountID)
	assert.WithinDuration(t, time.Now(), token.IssuedAt, time.Second)
	assert.WithinDuration(t, time.Now().Add(expiresIn), token.ExpiresAt, time.Second)
}

func TestAccessToken_IsExpired_NotExpired(t *testing.T) {
	// Arrange
	token := NewAccessToken(
		"token_123",
		"jwt_value",
		account.NewUserID(123),
		account.NewAccountID(456),
		1*time.Hour,
	)

	// Act
	expired := token.IsExpired()

	// Assert
	assert.False(t, expired, "Token should not be expired")
}

func TestAccessToken_IsExpired_Expired(t *testing.T) {
	// Arrange
	token := NewAccessToken(
		"token_123",
		"jwt_value",
		account.NewUserID(123),
		account.NewAccountID(456),
		-1*time.Hour, // 过去的时间
	)

	// Act
	expired := token.IsExpired()

	// Assert
	assert.True(t, expired, "Token should be expired")
}

func TestAccessToken_RemainingDuration(t *testing.T) {
	// Arrange
	expiresIn := 30 * time.Minute
	token := NewAccessToken(
		"token_123",
		"jwt_value",
		account.NewUserID(123),
		account.NewAccountID(456),
		expiresIn,
	)

	// Act
	remaining := token.RemainingDuration()

	// Assert
	assert.True(t, remaining > 29*time.Minute && remaining <= 30*time.Minute,
		"Remaining duration should be close to 30 minutes")
}

func TestAccessToken_RemainingDuration_Expired(t *testing.T) {
	// Arrange
	token := NewAccessToken(
		"token_123",
		"jwt_value",
		account.NewUserID(123),
		account.NewAccountID(456),
		-1*time.Hour, // 已过期
	)

	// Act
	remaining := token.RemainingDuration()

	// Assert
	assert.Equal(t, time.Duration(0), remaining, "Expired token should have 0 remaining duration")
}

// ==================== RefreshToken 测试 ====================

func TestNewRefreshToken_Success(t *testing.T) {
	// Arrange
	id := "refresh_token_456"
	value := "refresh_eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."
	userID := account.NewUserID(12345)
	accountID := account.NewAccountID(67890)
	expiresIn := 7 * 24 * time.Hour // 7天

	// Act
	token := NewRefreshToken(id, value, userID, accountID, expiresIn)

	// Assert
	require.NotNil(t, token)
	assert.Equal(t, id, token.ID)
	assert.Equal(t, TokenTypeRefresh, token.Type)
	assert.Equal(t, value, token.Value)
	assert.Equal(t, userID, token.UserID)
	assert.Equal(t, accountID, token.AccountID)
	assert.WithinDuration(t, time.Now(), token.IssuedAt, time.Second)
	assert.WithinDuration(t, time.Now().Add(expiresIn), token.ExpiresAt, time.Second)
}

func TestRefreshToken_IsExpired_NotExpired(t *testing.T) {
	// Arrange
	token := NewRefreshToken(
		"refresh_123",
		"jwt_value",
		account.NewUserID(123),
		account.NewAccountID(456),
		7*24*time.Hour,
	)

	// Act
	expired := token.IsExpired()

	// Assert
	assert.False(t, expired, "Refresh token should not be expired")
}

func TestRefreshToken_IsExpired_Expired(t *testing.T) {
	// Arrange
	token := NewRefreshToken(
		"refresh_123",
		"jwt_value",
		account.NewUserID(123),
		account.NewAccountID(456),
		-1*time.Hour, // 过去的时间
	)

	// Act
	expired := token.IsExpired()

	// Assert
	assert.True(t, expired, "Refresh token should be expired")
}

// ==================== TokenPair 测试 ====================

func TestNewTokenPair_Success(t *testing.T) {
	// Arrange
	accessToken := NewAccessToken(
		"access_123",
		"access_jwt",
		account.NewUserID(123),
		account.NewAccountID(456),
		1*time.Hour,
	)
	refreshToken := NewRefreshToken(
		"refresh_456",
		"refresh_jwt",
		account.NewUserID(123),
		account.NewAccountID(456),
		7*24*time.Hour,
	)

	// Act
	tokenPair := NewTokenPair(accessToken, refreshToken)

	// Assert
	require.NotNil(t, tokenPair)
	assert.Equal(t, accessToken, tokenPair.AccessToken)
	assert.Equal(t, refreshToken, tokenPair.RefreshToken)
}

func TestToken_TypeDifference(t *testing.T) {
	// Arrange
	userID := account.NewUserID(123)
	accountID := account.NewAccountID(456)

	accessToken := NewAccessToken("id1", "val1", userID, accountID, 1*time.Hour)
	refreshToken := NewRefreshToken("id2", "val2", userID, accountID, 1*time.Hour)

	// Assert
	assert.Equal(t, TokenTypeAccess, accessToken.Type)
	assert.Equal(t, TokenTypeRefresh, refreshToken.Type)
	assert.NotEqual(t, accessToken.Type, refreshToken.Type)
}

// ==================== 边界测试 ====================

func TestToken_ZeroExpiration(t *testing.T) {
	// Arrange - 0秒后过期
	token := NewAccessToken(
		"token_123",
		"jwt_value",
		account.NewUserID(123),
		account.NewAccountID(456),
		0,
	)

	// Act & Assert
	// 可能已经过期,也可能还未过期(取决于执行速度)
	_ = token.IsExpired()
	assert.Equal(t, token.Type, TokenTypeAccess)
}

func TestToken_VeryShortExpiration(t *testing.T) {
	// Arrange - 1毫秒后过期
	token := NewAccessToken(
		"token_123",
		"jwt_value",
		account.NewUserID(123),
		account.NewAccountID(456),
		1*time.Millisecond,
	)

	// Wait for expiration
	time.Sleep(2 * time.Millisecond)

	// Act
	expired := token.IsExpired()

	// Assert
	assert.True(t, expired, "Token with 1ms expiration should be expired after 2ms")
}

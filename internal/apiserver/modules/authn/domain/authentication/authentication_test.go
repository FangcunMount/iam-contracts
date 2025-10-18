package authentication

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/account"
)

// ==================== Authentication 实体测试 ====================

func TestNewAuthentication_Success(t *testing.T) {
	// Arrange
	userID := account.NewUserID(12345)
	accountID := account.NewAccountID(67890)
	provider := account.ProviderPassword
	metadata := map[string]string{
		"ip":     "192.168.1.1",
		"device": "iPhone 14",
	}

	// Act
	auth := NewAuthentication(userID, accountID, provider, metadata)

	// Assert
	require.NotNil(t, auth)
	assert.Equal(t, userID, auth.UserID)
	assert.Equal(t, accountID, auth.AccountID)
	assert.Equal(t, provider, auth.Provider)
	assert.Equal(t, "192.168.1.1", auth.Metadata["ip"])
	assert.Equal(t, "iPhone 14", auth.Metadata["device"])
	assert.WithinDuration(t, time.Now(), auth.AuthenticatedAt, time.Second)
}

func TestNewAuthentication_WithNilMetadata(t *testing.T) {
	// Arrange
	userID := account.NewUserID(12345)
	accountID := account.NewAccountID(67890)
	provider := account.ProviderWeChat

	// Act
	auth := NewAuthentication(userID, accountID, provider, nil)

	// Assert
	require.NotNil(t, auth)
	assert.NotNil(t, auth.Metadata)
	assert.Empty(t, auth.Metadata)
}

func TestAuthentication_WithMetadata(t *testing.T) {
	// Arrange
	auth := NewAuthentication(
		account.NewUserID(123),
		account.NewAccountID(456),
		account.ProviderPassword,
		nil,
	)

	// Act
	auth.WithMetadata("ip", "10.0.0.1")
	auth.WithMetadata("user_agent", "Mozilla/5.0")

	// Assert
	assert.Equal(t, "10.0.0.1", auth.Metadata["ip"])
	assert.Equal(t, "Mozilla/5.0", auth.Metadata["user_agent"])
}

func TestAuthentication_GetMetadata(t *testing.T) {
	// Arrange
	metadata := map[string]string{
		"ip":     "172.16.0.1",
		"device": "Android",
	}
	auth := NewAuthentication(
		account.NewUserID(123),
		account.NewAccountID(456),
		account.ProviderPassword,
		metadata,
	)

	// Act & Assert - 存在的 key
	value, ok := auth.GetMetadata("ip")
	assert.True(t, ok)
	assert.Equal(t, "172.16.0.1", value)

	// Act & Assert - 不存在的 key
	value, ok = auth.GetMetadata("nonexistent")
	assert.False(t, ok)
	assert.Empty(t, value)
}

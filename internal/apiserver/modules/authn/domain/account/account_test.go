package account

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewAccount 测试创建账号
func TestNewAccount(t *testing.T) {
	// Act
	account := NewAccount(
		UserID(123),
		ProviderPassword,
		WithID(NewAccountID(456)),
		WithExternalID("testuser"),
		WithStatus(StatusActive),
	)

	// Assert
	assert.Equal(t, NewAccountID(456), account.ID)
	assert.Equal(t, UserID(123), account.UserID)
	assert.Equal(t, ProviderPassword, account.Provider)
	assert.Equal(t, "testuser", account.ExternalID)
	assert.Equal(t, StatusActive, account.Status)
}

// TestNewAccount_WithAppID 测试创建带 AppID 的账号
func TestNewAccount_WithAppID(t *testing.T) {
	// Arrange
	appID := "wx1234567890abcdef"

	// Act
	account := NewAccount(
		UserID(123),
		ProviderWeChat,
		WithAppID(appID),
		WithExternalID("openid-xyz"),
	)

	// Assert
	assert.NotNil(t, account.AppID)
	assert.Equal(t, appID, *account.AppID)
	assert.Equal(t, ProviderWeChat, account.Provider)
	assert.Equal(t, "openid-xyz", account.ExternalID)
}

// TestAccount_StatusMethods 测试账号状态变更方法
func TestAccount_StatusMethods(t *testing.T) {
	// Arrange
	account := NewAccount(UserID(123), ProviderPassword)

	// 初始状态应该是默认值（0 = StatusDisabled）
	assert.Equal(t, StatusDisabled, account.Status)
	assert.True(t, account.IsDisabled())

	// Test Activate
	account.Activate()
	assert.Equal(t, StatusActive, account.Status)
	assert.True(t, account.IsActive())
	assert.False(t, account.IsDisabled())
	assert.False(t, account.IsArchived())
	assert.False(t, account.IsDeleted())

	// Test Disable
	account.Disable()
	assert.Equal(t, StatusDisabled, account.Status)
	assert.True(t, account.IsDisabled())
	assert.False(t, account.IsActive())

	// Test Archive
	account.Archive()
	assert.Equal(t, StatusArchived, account.Status)
	assert.True(t, account.IsArchived())
	assert.False(t, account.IsActive())
	assert.False(t, account.IsDisabled())

	// Test Delete
	account.Delete()
	assert.Equal(t, StatusDeleted, account.Status)
	assert.True(t, account.IsDeleted())
	assert.False(t, account.IsActive())
	assert.False(t, account.IsDisabled())
	assert.False(t, account.IsArchived())
}

// TestAccountStatus_String 测试账号状态字符串转换
func TestAccountStatus_String(t *testing.T) {
	tests := []struct {
		status   AccountStatus
		expected string
	}{
		{StatusDisabled, "disabled"},
		{StatusActive, "active"},
		{StatusArchived, "archived"},
		{StatusDeleted, "deleted"},
		{AccountStatus(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.String())
		})
	}
}

// TestNewAccountID 测试创建账号 ID
func TestNewAccountID(t *testing.T) {
	// Act
	id1 := NewAccountID(123)
	id2 := NewAccountID(123)
	id3 := NewAccountID(456)

	// Assert
	assert.Equal(t, id1, id2, "相同值的 AccountID 应该相等")
	assert.NotEqual(t, id1, id3, "不同值的 AccountID 应该不相等")
}

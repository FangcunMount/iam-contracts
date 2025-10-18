package authentication

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== UsernamePasswordCredential 测试 ====================

func TestNewUsernamePasswordCredential_Success(t *testing.T) {
	// Arrange
	username := "testuser"
	password := "testpass123"

	// Act
	credential := NewUsernamePasswordCredential(username, password)

	// Assert
	require.NotNil(t, credential)
	assert.Equal(t, username, credential.Username)
	assert.Equal(t, password, credential.Password)
	assert.Equal(t, CredentialTypeUsernamePassword, credential.Type())
}

func TestUsernamePasswordCredential_Validate_Success(t *testing.T) {
	// Arrange
	credential := NewUsernamePasswordCredential("user123", "pass456")

	// Act
	err := credential.Validate()

	// Assert
	assert.NoError(t, err)
}

func TestUsernamePasswordCredential_Validate_EmptyUsername(t *testing.T) {
	// Arrange
	credential := NewUsernamePasswordCredential("", "password")

	// Act
	err := credential.Validate()

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "username is required")
}

func TestUsernamePasswordCredential_Validate_EmptyPassword(t *testing.T) {
	// Arrange
	credential := NewUsernamePasswordCredential("username", "")

	// Act
	err := credential.Validate()

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "password is required")
}

// ==================== WeChatCodeCredential 测试 ====================

func TestNewWeChatCodeCredential_Success(t *testing.T) {
	// Arrange
	code := "001abcdef123456"
	appID := "wx1234567890abcdef"

	// Act
	credential := NewWeChatCodeCredential(code, appID)

	// Assert
	require.NotNil(t, credential)
	assert.Equal(t, code, credential.Code)
	assert.Equal(t, appID, credential.AppID)
	assert.Equal(t, CredentialTypeWeChatCode, credential.Type())
}

func TestWeChatCodeCredential_Validate_Success(t *testing.T) {
	// Arrange
	credential := NewWeChatCodeCredential("valid_code", "wx_app_id")

	// Act
	err := credential.Validate()

	// Assert
	assert.NoError(t, err)
}

func TestWeChatCodeCredential_Validate_EmptyCode(t *testing.T) {
	// Arrange
	credential := NewWeChatCodeCredential("", "wx_app_id")

	// Act
	err := credential.Validate()

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "wechat code is required")
}

func TestWeChatCodeCredential_Validate_EmptyAppID(t *testing.T) {
	// Arrange
	credential := NewWeChatCodeCredential("valid_code", "")

	// Act
	err := credential.Validate()

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "wechat app_id is required")
}

// ==================== TokenCredential 测试 ====================

func TestNewTokenCredential_Success(t *testing.T) {
	// Arrange
	token := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0..."

	// Act
	credential := NewTokenCredential(token)

	// Assert
	require.NotNil(t, credential)
	assert.Equal(t, token, credential.Token)
	assert.Equal(t, CredentialTypeToken, credential.Type())
}

func TestTokenCredential_Validate_Success(t *testing.T) {
	// Arrange
	credential := NewTokenCredential("valid.jwt.token")

	// Act
	err := credential.Validate()

	// Assert
	assert.NoError(t, err)
}

func TestTokenCredential_Validate_EmptyToken(t *testing.T) {
	// Arrange
	credential := NewTokenCredential("")

	// Act
	err := credential.Validate()

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token is required")
}

// ==================== Credential 接口测试 ====================

func TestCredential_InterfaceImplementation(t *testing.T) {
	// 验证所有凭证类型都实现了 Credential 接口
	var _ Credential = &UsernamePasswordCredential{}
	var _ Credential = &WeChatCodeCredential{}
	var _ Credential = &TokenCredential{}

	// 测试接口方法
	credentials := []Credential{
		NewUsernamePasswordCredential("user", "pass"),
		NewWeChatCodeCredential("code", "appid"),
		NewTokenCredential("token"),
	}

	for _, cred := range credentials {
		assert.NotEmpty(t, cred.Type())
		// 所有凭证都应该能验证
		_ = cred.Validate()
	}
}

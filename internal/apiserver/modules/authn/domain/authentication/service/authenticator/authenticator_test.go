package authenticator

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/authentication"
)

// ==================== Mock Authenticator ====================

// MockAuthenticator Mock 认证器 (用于测试 Authenticator 编排器)
type MockAuthenticator struct {
	supportedType authentication.CredentialType
	authResult    *authentication.Authentication
	authError     error
}

func NewMockAuthenticator(supportedType authentication.CredentialType) *MockAuthenticator {
	return &MockAuthenticator{
		supportedType: supportedType,
	}
}

func (m *MockAuthenticator) Supports(credential authentication.Credential) bool {
	return credential.Type() == m.supportedType
}

func (m *MockAuthenticator) Authenticate(ctx context.Context, credential authentication.Credential) (*authentication.Authentication, error) {
	if m.authError != nil {
		return nil, m.authError
	}
	if m.authResult != nil {
		return m.authResult, nil
	}
	// 默认返回成功的认证结果
	return authentication.NewAuthentication(
		account.NewUserID(999),
		account.NewAccountID(888),
		account.ProviderPassword,
		map[string]string{"mock": "true"},
	), nil
}

func (m *MockAuthenticator) SetAuthResult(auth *authentication.Authentication) {
	m.authResult = auth
}

func (m *MockAuthenticator) SetAuthError(err error) {
	m.authError = err
}

// ==================== Authenticator 编排器测试 ====================

func TestAuthenticator_Authenticate_SelectBasicAuthenticator(t *testing.T) {
	// Arrange
	ctx := context.Background()

	// 创建 Mock 认证器
	basicAuth := NewMockAuthenticator(authentication.CredentialTypeUsernamePassword)
	wechatAuth := NewMockAuthenticator(authentication.CredentialTypeWeChatCode)

	// 设置 BasicAuthenticator 的返回结果
	expectedAuth := authentication.NewAuthentication(
		account.NewUserID(12345),
		account.NewAccountID(67890),
		account.ProviderPassword,
		map[string]string{"username": "testuser"},
	)
	basicAuth.SetAuthResult(expectedAuth)

	// 创建编排器
	authenticator := NewAuthenticator(basicAuth, wechatAuth)

	// 创建用户名密码凭证
	credential := authentication.NewUsernamePasswordCredential("testuser", "password123")

	// Act
	auth, err := authenticator.Authenticate(ctx, credential)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, auth)
	assert.Equal(t, expectedAuth.UserID, auth.UserID)
	assert.Equal(t, expectedAuth.AccountID, auth.AccountID)
	assert.Equal(t, expectedAuth.Provider, auth.Provider)
}

func TestAuthenticator_Authenticate_SelectWeChatAuthenticator(t *testing.T) {
	// Arrange
	ctx := context.Background()

	// 创建 Mock 认证器
	basicAuth := NewMockAuthenticator(authentication.CredentialTypeUsernamePassword)
	wechatAuth := NewMockAuthenticator(authentication.CredentialTypeWeChatCode)

	// 设置 WeChatAuthenticator 的返回结果
	expectedAuth := authentication.NewAuthentication(
		account.NewUserID(54321),
		account.NewAccountID(98765),
		account.ProviderWeChat,
		map[string]string{"openid": "wx_openid_123"},
	)
	wechatAuth.SetAuthResult(expectedAuth)

	// 创建编排器
	authenticator := NewAuthenticator(basicAuth, wechatAuth)

	// 创建微信凭证
	credential := authentication.NewWeChatCodeCredential("code123", "wx_app_id")

	// Act
	auth, err := authenticator.Authenticate(ctx, credential)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, auth)
	assert.Equal(t, expectedAuth.UserID, auth.UserID)
	assert.Equal(t, expectedAuth.AccountID, auth.AccountID)
	assert.Equal(t, expectedAuth.Provider, auth.Provider)
}

func TestAuthenticator_Authenticate_UnsupportedCredentialType(t *testing.T) {
	// Arrange
	ctx := context.Background()

	// 创建 Mock 认证器 (只支持用户名密码)
	basicAuth := NewMockAuthenticator(authentication.CredentialTypeUsernamePassword)

	// 创建编排器
	authenticator := NewAuthenticator(basicAuth)

	// 创建微信凭证 (不支持)
	credential := authentication.NewWeChatCodeCredential("code123", "wx_app_id")

	// Act
	auth, err := authenticator.Authenticate(ctx, credential)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, auth)
	// 错误信息被包装,只验证有错误即可
}

func TestAuthenticator_Authenticate_InvalidCredential(t *testing.T) {
	// Arrange
	ctx := context.Background()

	basicAuth := NewMockAuthenticator(authentication.CredentialTypeUsernamePassword)
	authenticator := NewAuthenticator(basicAuth)

	// 创建无效的凭证 (空用户名)
	credential := authentication.NewUsernamePasswordCredential("", "password")

	// Act
	auth, err := authenticator.Authenticate(ctx, credential)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, auth)
	// 错误信息被包装,只验证有错误即可
}

func TestAuthenticator_Authenticate_AuthenticatorReturnsError(t *testing.T) {
	// Arrange
	ctx := context.Background()

	basicAuth := NewMockAuthenticator(authentication.CredentialTypeUsernamePassword)
	expectedError := errors.New("authentication failed: invalid password")
	basicAuth.SetAuthError(expectedError)

	authenticator := NewAuthenticator(basicAuth)

	credential := authentication.NewUsernamePasswordCredential("testuser", "wrong_password")

	// Act
	auth, err := authenticator.Authenticate(ctx, credential)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, auth)
	assert.Contains(t, err.Error(), "authentication failed")
}

func TestAuthenticator_Supports_SingleAuthenticator(t *testing.T) {
	// Arrange
	basicAuth := NewMockAuthenticator(authentication.CredentialTypeUsernamePassword)
	authenticator := NewAuthenticator(basicAuth)

	// Test supported credential
	upCredential := authentication.NewUsernamePasswordCredential("user", "pass")
	assert.True(t, authenticator.Supports(upCredential))

	// Test unsupported credential
	wxCredential := authentication.NewWeChatCodeCredential("code", "appid")
	assert.False(t, authenticator.Supports(wxCredential))
}

func TestAuthenticator_Supports_MultipleAuthenticators(t *testing.T) {
	// Arrange
	basicAuth := NewMockAuthenticator(authentication.CredentialTypeUsernamePassword)
	wechatAuth := NewMockAuthenticator(authentication.CredentialTypeWeChatCode)
	authenticator := NewAuthenticator(basicAuth, wechatAuth)

	// Test both supported credentials
	upCredential := authentication.NewUsernamePasswordCredential("user", "pass")
	assert.True(t, authenticator.Supports(upCredential))

	wxCredential := authentication.NewWeChatCodeCredential("code", "appid")
	assert.True(t, authenticator.Supports(wxCredential))

	// Test unsupported credential
	tokenCredential := authentication.NewTokenCredential("token")
	assert.False(t, authenticator.Supports(tokenCredential))
}

func TestAuthenticator_RegisterAuthenticator(t *testing.T) {
	// Arrange
	basicAuth := NewMockAuthenticator(authentication.CredentialTypeUsernamePassword)
	authenticator := NewAuthenticator(basicAuth)

	// 初始状态：不支持微信凭证
	wxCredential := authentication.NewWeChatCodeCredential("code", "appid")
	assert.False(t, authenticator.Supports(wxCredential))

	// Act - 动态注册微信认证器
	wechatAuth := NewMockAuthenticator(authentication.CredentialTypeWeChatCode)
	authenticator.RegisterAuthenticator(wechatAuth)

	// Assert - 现在支持微信凭证
	assert.True(t, authenticator.Supports(wxCredential))
}

func TestAuthenticator_EmptyAuthenticators(t *testing.T) {
	// Arrange
	ctx := context.Background()
	authenticator := NewAuthenticator() // 没有注册任何认证器

	credential := authentication.NewUsernamePasswordCredential("user", "pass")

	// Act
	auth, err := authenticator.Authenticate(ctx, credential)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, auth)
	// 错误信息被包装,只验证有错误即可
}

func TestAuthenticator_FirstMatchingAuthenticatorWins(t *testing.T) {
	// Arrange
	ctx := context.Background()

	// 创建两个都支持用户名密码的认证器
	firstAuth := NewMockAuthenticator(authentication.CredentialTypeUsernamePassword)
	secondAuth := NewMockAuthenticator(authentication.CredentialTypeUsernamePassword)

	expectedAuth1 := authentication.NewAuthentication(
		account.NewUserID(111),
		account.NewAccountID(222),
		account.ProviderPassword,
		map[string]string{"authenticator": "first"},
	)
	firstAuth.SetAuthResult(expectedAuth1)

	expectedAuth2 := authentication.NewAuthentication(
		account.NewUserID(333),
		account.NewAccountID(444),
		account.ProviderPassword,
		map[string]string{"authenticator": "second"},
	)
	secondAuth.SetAuthResult(expectedAuth2)

	// 创建编排器 (firstAuth 在前)
	authenticator := NewAuthenticator(firstAuth, secondAuth)

	credential := authentication.NewUsernamePasswordCredential("user", "pass")

	// Act
	auth, err := authenticator.Authenticate(ctx, credential)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, auth)
	// 应该使用第一个匹配的认证器
	assert.Equal(t, expectedAuth1.UserID, auth.UserID)
	assert.Equal(t, "first", auth.Metadata["authenticator"])
}

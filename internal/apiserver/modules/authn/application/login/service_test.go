package login

import (
	"context"
	"errors"
	"testing"
	"time"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/authentication"
	authService "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/authentication/service/authenticator"
	tokenService "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/authentication/service/token"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	"github.com/stretchr/testify/assert"
)

// MockAuthStrategy 模拟认证策略
type MockAuthStrategy struct {
	SupportsFunc     func(credential authentication.Credential) bool
	AuthenticateFunc func(ctx context.Context, credential authentication.Credential) (*authentication.Authentication, error)
}

func (m *MockAuthStrategy) Supports(credential authentication.Credential) bool {
	if m.SupportsFunc != nil {
		return m.SupportsFunc(credential)
	}
	return true
}

func (m *MockAuthStrategy) Authenticate(ctx context.Context, credential authentication.Credential) (*authentication.Authentication, error) {
	if m.AuthenticateFunc != nil {
		return m.AuthenticateFunc(ctx, credential)
	}
	return authentication.NewAuthentication(
		account.UserID(123),
		account.NewAccountID(456),
		"basic",
		nil,
	), nil
}

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

// TestLoginService_LoginWithPassword_Success 测试成功用密码登录
func TestLoginService_LoginWithPassword_Success(t *testing.T) {
	// Arrange
	mockStrategy := &MockAuthStrategy{
		SupportsFunc: func(credential authentication.Credential) bool {
			assert.Equal(t, authentication.CredentialTypeUsernamePassword, credential.Type())
			return true
		},
		AuthenticateFunc: func(ctx context.Context, credential authentication.Credential) (*authentication.Authentication, error) {
			upCred, ok := credential.(*authentication.UsernamePasswordCredential)
			assert.True(t, ok)
			assert.Equal(t, "testuser", upCred.Username)
			assert.Equal(t, "password123", upCred.Password)

			return authentication.NewAuthentication(
				account.UserID(100),
				account.NewAccountID(200),
				"basic",
				nil,
			), nil
		},
	}

	authenticator := authService.NewAuthenticator()
	authenticator.RegisterAuthenticator(mockStrategy)

	mockTokenGen := &MockTokenGenerator{}
	mockTokenStore := &MockTokenStore{}
	tokenIssuer := tokenService.NewTokenIssuer(mockTokenGen, mockTokenStore, 15*time.Minute, 7*24*time.Hour)

	service := NewLoginService(authenticator, tokenIssuer)
	ctx := context.Background()

	req := &LoginWithPasswordRequest{
		Username: "testuser",
		Password: "password123",
		IP:       "192.168.1.100",
		Device:   "iPhone 13",
	}

	// Act
	resp, err := service.LoginWithPassword(ctx, req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "mock-access-token", resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Equal(t, "Bearer", resp.TokenType)
	assert.Greater(t, resp.ExpiresIn, int64(0))
}

// TestLoginService_LoginWithPassword_NilRequest 测试空请求
func TestLoginService_LoginWithPassword_NilRequest(t *testing.T) {
	// Arrange
	authenticator := authService.NewAuthenticator()
	mockTokenGen := &MockTokenGenerator{}
	mockTokenStore := &MockTokenStore{}
	tokenIssuer := tokenService.NewTokenIssuer(mockTokenGen, mockTokenStore, 15*time.Minute, 7*24*time.Hour)
	service := NewLoginService(authenticator, tokenIssuer)
	ctx := context.Background()

	// Act
	resp, err := service.LoginWithPassword(ctx, nil)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, perrors.IsCode(err, code.ErrInvalidArgument))
}

// TestLoginService_LoginWithPassword_UnsupportedCredential 测试不支持的凭证类型
func TestLoginService_LoginWithPassword_UnsupportedCredential(t *testing.T) {
	// Arrange - authenticator 没有注册任何策略
	authenticator := authService.NewAuthenticator()
	mockTokenGen := &MockTokenGenerator{}
	mockTokenStore := &MockTokenStore{}
	tokenIssuer := tokenService.NewTokenIssuer(mockTokenGen, mockTokenStore, 15*time.Minute, 7*24*time.Hour)
	service := NewLoginService(authenticator, tokenIssuer)
	ctx := context.Background()

	req := &LoginWithPasswordRequest{
		Username: "testuser",
		Password: "password123",
	}

	// Act
	resp, err := service.LoginWithPassword(ctx, req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, perrors.IsCode(err, code.ErrInvalidArgument))
}

// TestLoginService_LoginWithPassword_AuthenticationFailed 测试认证失败
func TestLoginService_LoginWithPassword_AuthenticationFailed(t *testing.T) {
	// Arrange
	mockStrategy := &MockAuthStrategy{
		SupportsFunc: func(credential authentication.Credential) bool {
			return true
		},
		AuthenticateFunc: func(ctx context.Context, credential authentication.Credential) (*authentication.Authentication, error) {
			return nil, perrors.WithCode(code.ErrPasswordIncorrect, "wrong password")
		},
	}

	authenticator := authService.NewAuthenticator()
	authenticator.RegisterAuthenticator(mockStrategy)
	mockTokenGen := &MockTokenGenerator{}
	mockTokenStore := &MockTokenStore{}
	tokenIssuer := tokenService.NewTokenIssuer(mockTokenGen, mockTokenStore, 15*time.Minute, 7*24*time.Hour)
	service := NewLoginService(authenticator, tokenIssuer)
	ctx := context.Background()

	req := &LoginWithPasswordRequest{
		Username: "testuser",
		Password: "wrongpassword",
	}

	// Act
	resp, err := service.LoginWithPassword(ctx, req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, perrors.IsCode(err, code.ErrPasswordIncorrect))
}

// TestLoginService_LoginWithPassword_IssueTokenFailed 测试颁发令牌失败
func TestLoginService_LoginWithPassword_IssueTokenFailed(t *testing.T) {
	// Arrange
	mockStrategy := &MockAuthStrategy{}
	authenticator := authService.NewAuthenticator()
	authenticator.RegisterAuthenticator(mockStrategy)

	mockTokenGen := &MockTokenGenerator{
		GenerateAccessTokenFunc: func(auth *authentication.Authentication, ttl time.Duration) (*authentication.Token, error) {
			return nil, errors.New("token generation failed")
		},
	}
	mockTokenStore := &MockTokenStore{}
	tokenIssuer := tokenService.NewTokenIssuer(mockTokenGen, mockTokenStore, 15*time.Minute, 7*24*time.Hour)
	service := NewLoginService(authenticator, tokenIssuer)
	ctx := context.Background()

	req := &LoginWithPasswordRequest{
		Username: "testuser",
		Password: "password123",
	}

	// Act
	resp, err := service.LoginWithPassword(ctx, req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, resp)
}

// TestLoginService_LoginWithWeChat_Success 测试成功用微信登录
func TestLoginService_LoginWithWeChat_Success(t *testing.T) {
	// Arrange
	mockStrategy := &MockAuthStrategy{
		SupportsFunc: func(credential authentication.Credential) bool {
			assert.Equal(t, authentication.CredentialTypeWeChatCode, credential.Type())
			return true
		},
		AuthenticateFunc: func(ctx context.Context, credential authentication.Credential) (*authentication.Authentication, error) {
			wxCred, ok := credential.(*authentication.WeChatCodeCredential)
			assert.True(t, ok)
			assert.Equal(t, "wx-code-123", wxCred.Code)
			assert.Equal(t, "wx1234567890abcdef", wxCred.AppID)

			return authentication.NewAuthentication(
				account.UserID(300),
				account.NewAccountID(400),
				"wechat",
				nil,
			), nil
		},
	}

	authenticator := authService.NewAuthenticator()
	authenticator.RegisterAuthenticator(mockStrategy)
	mockTokenGen := &MockTokenGenerator{}
	mockTokenStore := &MockTokenStore{}
	tokenIssuer := tokenService.NewTokenIssuer(mockTokenGen, mockTokenStore, 15*time.Minute, 7*24*time.Hour)
	service := NewLoginService(authenticator, tokenIssuer)
	ctx := context.Background()

	req := &LoginWithWeChatRequest{
		Code:   "wx-code-123",
		AppID:  "wx1234567890abcdef",
		IP:     "192.168.1.101",
		Device: "Android Phone",
	}

	// Act
	resp, err := service.LoginWithWeChat(ctx, req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "mock-access-token", resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Equal(t, "Bearer", resp.TokenType)
	assert.Greater(t, resp.ExpiresIn, int64(0))
}

// TestLoginService_LoginWithWeChat_NilRequest 测试微信登录空请求
func TestLoginService_LoginWithWeChat_NilRequest(t *testing.T) {
	// Arrange
	authenticator := authService.NewAuthenticator()
	mockTokenGen := &MockTokenGenerator{}
	mockTokenStore := &MockTokenStore{}
	tokenIssuer := tokenService.NewTokenIssuer(mockTokenGen, mockTokenStore, 15*time.Minute, 7*24*time.Hour)
	service := NewLoginService(authenticator, tokenIssuer)
	ctx := context.Background()

	// Act
	resp, err := service.LoginWithWeChat(ctx, nil)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, perrors.IsCode(err, code.ErrInvalidArgument))
}

// TestLoginService_LoginWithWeChat_UnsupportedCredential 测试微信登录不支持的凭证
func TestLoginService_LoginWithWeChat_UnsupportedCredential(t *testing.T) {
	// Arrange
	authenticator := authService.NewAuthenticator()
	mockTokenGen := &MockTokenGenerator{}
	mockTokenStore := &MockTokenStore{}
	tokenIssuer := tokenService.NewTokenIssuer(mockTokenGen, mockTokenStore, 15*time.Minute, 7*24*time.Hour)
	service := NewLoginService(authenticator, tokenIssuer)
	ctx := context.Background()

	req := &LoginWithWeChatRequest{
		Code:  "wx-code-123",
		AppID: "wx1234567890abcdef",
	}

	// Act
	resp, err := service.LoginWithWeChat(ctx, req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, perrors.IsCode(err, code.ErrInvalidArgument))
}

// TestLoginService_LoginWithWeChat_AuthenticationFailed 测试微信认证失败
func TestLoginService_LoginWithWeChat_AuthenticationFailed(t *testing.T) {
	// Arrange
	mockStrategy := &MockAuthStrategy{
		SupportsFunc: func(credential authentication.Credential) bool {
			return true
		},
		AuthenticateFunc: func(ctx context.Context, credential authentication.Credential) (*authentication.Authentication, error) {
			return nil, perrors.WithCode(code.ErrPasswordIncorrect, "invalid code")
		},
	}

	authenticator := authService.NewAuthenticator()
	authenticator.RegisterAuthenticator(mockStrategy)
	mockTokenGen := &MockTokenGenerator{}
	mockTokenStore := &MockTokenStore{}
	tokenIssuer := tokenService.NewTokenIssuer(mockTokenGen, mockTokenStore, 15*time.Minute, 7*24*time.Hour)
	service := NewLoginService(authenticator, tokenIssuer)
	ctx := context.Background()

	req := &LoginWithWeChatRequest{
		Code:  "invalid-code",
		AppID: "wx1234567890abcdef",
	}

	// Act
	resp, err := service.LoginWithWeChat(ctx, req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, perrors.IsCode(err, code.ErrPasswordIncorrect))
}

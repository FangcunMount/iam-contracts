package service

import (
	"testing"

	domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	perrors "github.com/fangcun-mount/iam-contracts/pkg/errors"
	"github.com/stretchr/testify/assert"
)

// TestCreateAccount_Success 测试成功创建账号
func TestCreateAccount_Success(t *testing.T) {
	// Arrange
	userID := domain.UserID(123)
	provider := domain.ProviderPassword
	externalID := "testuser"

	// Act
	account, err := CreateAccount(userID, provider, externalID, nil)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, account)
	assert.Equal(t, userID, account.UserID)
	assert.Equal(t, provider, account.Provider)
	assert.Equal(t, "testuser", account.ExternalID)
	assert.Equal(t, domain.StatusActive, account.Status)
	assert.Nil(t, account.AppID)
}

// TestCreateAccount_WithAppID 测试创建带 AppID 的账号
func TestCreateAccount_WithAppID(t *testing.T) {
	// Arrange
	userID := domain.UserID(456)
	provider := domain.ProviderWeChat
	externalID := "openid-xyz"
	appID := "wx1234567890abcdef"

	// Act
	account, err := CreateAccount(userID, provider, externalID, &appID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, account)
	assert.Equal(t, userID, account.UserID)
	assert.Equal(t, provider, account.Provider)
	assert.Equal(t, "openid-xyz", account.ExternalID)
	assert.NotNil(t, account.AppID)
	assert.Equal(t, appID, *account.AppID)
	assert.Equal(t, domain.StatusActive, account.Status)
}

// TestCreateAccount_InvalidUserID 测试用户 ID 为空
func TestCreateAccount_InvalidUserID(t *testing.T) {
	// Arrange
	userID := domain.UserID(0) // Zero value
	provider := domain.ProviderPassword
	externalID := "testuser"

	// Act
	account, err := CreateAccount(userID, provider, externalID, nil)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, account)
	assert.True(t, perrors.IsCode(err, code.ErrInvalidArgument))
}

// TestCreateAccount_EmptyProvider 测试 Provider 为空
func TestCreateAccount_EmptyProvider(t *testing.T) {
	// Arrange
	userID := domain.UserID(123)
	provider := domain.Provider("")
	externalID := "testuser"

	// Act
	account, err := CreateAccount(userID, provider, externalID, nil)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, account)
	assert.True(t, perrors.IsCode(err, code.ErrInvalidArgument))
}

// TestCreateAccount_EmptyExternalID 测试 ExternalID 为空
func TestCreateAccount_EmptyExternalID(t *testing.T) {
	// Arrange
	userID := domain.UserID(123)
	provider := domain.ProviderPassword
	externalID := "  " // Whitespace only

	// Act
	account, err := CreateAccount(userID, provider, externalID, nil)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, account)
	assert.True(t, perrors.IsCode(err, code.ErrInvalidArgument))
}

// TestCreateAccount_EmptyAppID 测试 AppID 为空字符串（指针非空）
func TestCreateAccount_EmptyAppID(t *testing.T) {
	// Arrange
	userID := domain.UserID(123)
	provider := domain.ProviderWeChat
	externalID := "openid-xyz"
	appID := "  " // Whitespace only

	// Act
	account, err := CreateAccount(userID, provider, externalID, &appID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, account)
	assert.True(t, perrors.IsCode(err, code.ErrInvalidArgument))
}

// TestCreateOperationAccount_Success 测试成功创建运营账号
func TestCreateOperationAccount_Success(t *testing.T) {
	// Arrange
	accountID := domain.NewAccountID(123)
	username := "admin"
	algo := "bcrypt"

	// Act
	opAccount, err := CreateOperationAccount(accountID, username, algo)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, opAccount)
	assert.Equal(t, accountID, opAccount.AccountID)
	assert.Equal(t, username, opAccount.Username)
	assert.Equal(t, algo, opAccount.Algo)
	assert.NotNil(t, opAccount.PasswordHash)
	assert.NotEmpty(t, opAccount.LastChangedAt)
}

// TestCreateOperationAccount_DefaultAlgo 测试使用默认算法
func TestCreateOperationAccount_DefaultAlgo(t *testing.T) {
	// Arrange
	accountID := domain.NewAccountID(456)
	username := "operator"
	algo := "  " // Empty/whitespace

	// Act
	opAccount, err := CreateOperationAccount(accountID, username, algo)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, opAccount)
	assert.Equal(t, "plain", opAccount.Algo, "应使用默认算法")
}

// TestCreateOperationAccount_EmptyUsername 测试用户名为空
func TestCreateOperationAccount_EmptyUsername(t *testing.T) {
	// Arrange
	accountID := domain.NewAccountID(123)
	username := "  "
	algo := "bcrypt"

	// Act
	opAccount, err := CreateOperationAccount(accountID, username, algo)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, opAccount)
	assert.True(t, perrors.IsCode(err, code.ErrInvalidArgument))
}

// TestCreateWeChatAccount_Success 测试成功创建微信账号
func TestCreateWeChatAccount_Success(t *testing.T) {
	// Arrange
	accountID := domain.NewAccountID(789)
	appID := "wx1234567890abcdef"
	openID := "openid-xyz-123"

	// Act
	wxAccount, err := CreateWeChatAccount(accountID, appID, openID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, wxAccount)
	assert.Equal(t, accountID, wxAccount.AccountID)
	assert.Equal(t, appID, wxAccount.AppID)
	assert.Equal(t, openID, wxAccount.OpenID)
}

// TestCreateWeChatAccount_EmptyAppID 测试 AppID 为空
func TestCreateWeChatAccount_EmptyAppID(t *testing.T) {
	// Arrange
	accountID := domain.NewAccountID(789)
	appID := "  "
	openID := "openid-xyz-123"

	// Act
	wxAccount, err := CreateWeChatAccount(accountID, appID, openID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, wxAccount)
	assert.True(t, perrors.IsCode(err, code.ErrInvalidArgument))
}

// TestCreateWeChatAccount_EmptyOpenID 测试 OpenID 为空
func TestCreateWeChatAccount_EmptyOpenID(t *testing.T) {
	// Arrange
	accountID := domain.NewAccountID(789)
	appID := "wx1234567890abcdef"
	openID := ""

	// Act
	wxAccount, err := CreateWeChatAccount(accountID, appID, openID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, wxAccount)
	assert.True(t, perrors.IsCode(err, code.ErrInvalidArgument))
}

// TestCreateWeChatAccount_BothEmpty 测试 AppID 和 OpenID 都为空
func TestCreateWeChatAccount_BothEmpty(t *testing.T) {
	// Arrange
	accountID := domain.NewAccountID(789)
	appID := ""
	openID := "  "

	// Act
	wxAccount, err := CreateWeChatAccount(accountID, appID, openID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, wxAccount)
	assert.True(t, perrors.IsCode(err, code.ErrInvalidArgument))
}

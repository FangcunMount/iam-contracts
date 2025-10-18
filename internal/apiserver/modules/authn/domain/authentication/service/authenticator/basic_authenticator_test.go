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

// ==================== Mock 实现 ====================

// MockAccountRepo Mock 账号仓储
type MockAccountRepo struct {
	accounts map[account.AccountID]*account.Account
}

func NewMockAccountRepo() *MockAccountRepo {
	return &MockAccountRepo{
		accounts: make(map[account.AccountID]*account.Account),
	}
}

func (m *MockAccountRepo) Create(ctx context.Context, a *account.Account) error {
	m.accounts[a.ID] = a
	return nil
}

func (m *MockAccountRepo) FindByID(ctx context.Context, id account.AccountID) (*account.Account, error) {
	acc, ok := m.accounts[id]
	if !ok {
		return nil, errors.New("account not found")
	}
	return acc, nil
}

func (m *MockAccountRepo) FindByRef(ctx context.Context, provider account.Provider, externalID string, appID *string) (*account.Account, error) {
	return nil, errors.New("not implemented")
}

func (m *MockAccountRepo) UpdateStatus(ctx context.Context, id account.AccountID, status account.AccountStatus) error {
	return nil
}

func (m *MockAccountRepo) UpdateUserID(ctx context.Context, id account.AccountID, userID account.UserID) error {
	return nil
}

func (m *MockAccountRepo) UpdateExternalRef(ctx context.Context, id account.AccountID, externalID string, appID *string) error {
	return nil
}

// MockOperationRepo Mock 运营账号仓储
type MockOperationRepo struct {
	operations map[string]*account.OperationAccount // key: username
}

func NewMockOperationRepo() *MockOperationRepo {
	return &MockOperationRepo{
		operations: make(map[string]*account.OperationAccount),
	}
}

func (m *MockOperationRepo) Create(ctx context.Context, cred *account.OperationAccount) error {
	m.operations[cred.Username] = cred
	return nil
}

func (m *MockOperationRepo) FindByAccountID(ctx context.Context, accountID account.AccountID) (*account.OperationAccount, error) {
	return nil, errors.New("not implemented")
}

func (m *MockOperationRepo) FindByUsername(ctx context.Context, username string) (*account.OperationAccount, error) {
	op, ok := m.operations[username]
	if !ok {
		return nil, errors.New("username not found")
	}
	return op, nil
}

func (m *MockOperationRepo) UpdateHash(ctx context.Context, username string, hash []byte, algo string, params []byte) error {
	return nil
}

func (m *MockOperationRepo) UpdateUsername(ctx context.Context, accountID account.AccountID, newUsername string) error {
	return nil
}

func (m *MockOperationRepo) ResetFailures(ctx context.Context, username string) error {
	return nil
}

func (m *MockOperationRepo) Unlock(ctx context.Context, username string) error {
	return nil
}

// MockPasswordPort Mock 密码端口
type MockPasswordPort struct {
	passwordHashes map[account.AccountID]*authentication.PasswordHash
}

func NewMockPasswordPort() *MockPasswordPort {
	return &MockPasswordPort{
		passwordHashes: make(map[account.AccountID]*authentication.PasswordHash),
	}
}

func (m *MockPasswordPort) GetPasswordHash(ctx context.Context, accountID account.AccountID) (*authentication.PasswordHash, error) {
	hash, ok := m.passwordHashes[accountID]
	if !ok {
		return nil, errors.New("password hash not found")
	}
	return hash, nil
}

func (m *MockPasswordPort) SetPasswordHash(accountID account.AccountID, hash *authentication.PasswordHash) {
	m.passwordHashes[accountID] = hash
}

// ==================== BasicAuthenticator 测试 ====================

func TestBasicAuthenticator_Authenticate_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()

	// 创建 Mock
	accountRepo := NewMockAccountRepo()
	operationRepo := NewMockOperationRepo()
	passwordPort := NewMockPasswordPort()

	// 准备测试数据
	accountID := account.NewAccountID(1001)
	userID := account.NewUserID(2001)

	acc := account.NewAccount(
		userID,
		account.ProviderPassword,
		account.WithID(accountID),
		account.WithExternalID("testuser"),
		account.WithStatus(account.StatusActive),
	)
	accountRepo.Create(ctx, &acc)

	opAccount := account.NewOperationAccount(accountID, "testuser", "bcrypt")
	operationRepo.Create(ctx, &opAccount)

	// 创建密码哈希
	passwordHash, err := authentication.HashPassword("password123", authentication.AlgorithmBcrypt)
	require.NoError(t, err)
	passwordPort.SetPasswordHash(accountID, passwordHash)

	// 创建认证器
	authenticator := NewBasicAuthenticator(accountRepo, operationRepo, passwordPort)

	// 创建凭证
	credential := authentication.NewUsernamePasswordCredential("testuser", "password123")

	// Act
	auth, err := authenticator.Authenticate(ctx, credential)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, auth)
	assert.Equal(t, userID, auth.UserID)
	assert.Equal(t, accountID, auth.AccountID)
	assert.Equal(t, account.ProviderPassword, auth.Provider)
	assert.Equal(t, "testuser", auth.Metadata["username"])
}

func TestBasicAuthenticator_Authenticate_UsernameNotFound(t *testing.T) {
	// Arrange
	ctx := context.Background()

	accountRepo := NewMockAccountRepo()
	operationRepo := NewMockOperationRepo()
	passwordPort := NewMockPasswordPort()

	authenticator := NewBasicAuthenticator(accountRepo, operationRepo, passwordPort)

	// 创建凭证 (用户名不存在)
	credential := authentication.NewUsernamePasswordCredential("nonexistent", "password123")

	// Act
	auth, err := authenticator.Authenticate(ctx, credential)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, auth)
	// 错误信息被包装,只验证有错误即可
}

func TestBasicAuthenticator_Authenticate_WrongPassword(t *testing.T) {
	// Arrange
	ctx := context.Background()

	accountRepo := NewMockAccountRepo()
	operationRepo := NewMockOperationRepo()
	passwordPort := NewMockPasswordPort()

	// 准备测试数据
	accountID := account.NewAccountID(1001)
	userID := account.NewUserID(2001)

	acc := account.NewAccount(
		userID,
		account.ProviderPassword,
		account.WithID(accountID),
		account.WithExternalID("testuser"),
		account.WithStatus(account.StatusActive),
	)
	accountRepo.Create(ctx, &acc)

	opAccount := account.NewOperationAccount(accountID, "testuser", "bcrypt")
	operationRepo.Create(ctx, &opAccount)

	// 创建密码哈希 (正确密码)
	passwordHash, err := authentication.HashPassword("correct_password", authentication.AlgorithmBcrypt)
	require.NoError(t, err)
	passwordPort.SetPasswordHash(accountID, passwordHash)

	authenticator := NewBasicAuthenticator(accountRepo, operationRepo, passwordPort)

	// 创建凭证 (错误密码)
	credential := authentication.NewUsernamePasswordCredential("testuser", "wrong_password")

	// Act
	auth, err := authenticator.Authenticate(ctx, credential)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, auth)
	// 错误信息被包装,只验证有错误即可
}

func TestBasicAuthenticator_Authenticate_AccountDisabled(t *testing.T) {
	// Arrange
	ctx := context.Background()

	accountRepo := NewMockAccountRepo()
	operationRepo := NewMockOperationRepo()
	passwordPort := NewMockPasswordPort()

	// 准备测试数据 - 账号被禁用
	accountID := account.NewAccountID(1001)
	userID := account.NewUserID(2001)

	acc := account.NewAccount(
		userID,
		account.ProviderPassword,
		account.WithID(accountID),
		account.WithExternalID("testuser"),
		account.WithStatus(account.StatusDisabled), // 禁用状态
	)
	accountRepo.Create(ctx, &acc)

	opAccount := account.NewOperationAccount(accountID, "testuser", "bcrypt")
	operationRepo.Create(ctx, &opAccount)

	// 创建密码哈希
	passwordHash, err := authentication.HashPassword("password123", authentication.AlgorithmBcrypt)
	require.NoError(t, err)
	passwordPort.SetPasswordHash(accountID, passwordHash)

	authenticator := NewBasicAuthenticator(accountRepo, operationRepo, passwordPort)

	// 创建凭证
	credential := authentication.NewUsernamePasswordCredential("testuser", "password123")

	// Act
	auth, err := authenticator.Authenticate(ctx, credential)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, auth)
	// 错误信息被包装,只验证有错误即可
}

func TestBasicAuthenticator_Authenticate_InvalidCredentialType(t *testing.T) {
	// Arrange
	ctx := context.Background()

	accountRepo := NewMockAccountRepo()
	operationRepo := NewMockOperationRepo()
	passwordPort := NewMockPasswordPort()

	authenticator := NewBasicAuthenticator(accountRepo, operationRepo, passwordPort)

	// 创建错误类型的凭证 (微信凭证)
	credential := authentication.NewWeChatCodeCredential("code123", "wx_app_id")

	// Act
	auth, err := authenticator.Authenticate(ctx, credential)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, auth)
	// 错误信息被包装,只验证有错误即可
}

func TestBasicAuthenticator_Authenticate_EmptyUsername(t *testing.T) {
	// Arrange
	ctx := context.Background()

	accountRepo := NewMockAccountRepo()
	operationRepo := NewMockOperationRepo()
	passwordPort := NewMockPasswordPort()

	authenticator := NewBasicAuthenticator(accountRepo, operationRepo, passwordPort)

	// 创建空用户名凭证
	credential := authentication.NewUsernamePasswordCredential("", "password123")

	// Act
	auth, err := authenticator.Authenticate(ctx, credential)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, auth)
	// 错误信息被包装,只验证有错误即可
}

func TestBasicAuthenticator_Supports(t *testing.T) {
	// Arrange
	accountRepo := NewMockAccountRepo()
	operationRepo := NewMockOperationRepo()
	passwordPort := NewMockPasswordPort()

	authenticator := NewBasicAuthenticator(accountRepo, operationRepo, passwordPort)

	// Test case 1: 支持用户名密码凭证
	upCredential := authentication.NewUsernamePasswordCredential("user", "pass")
	assert.True(t, authenticator.Supports(upCredential))

	// Test case 2: 不支持微信凭证
	wxCredential := authentication.NewWeChatCodeCredential("code", "appid")
	assert.False(t, authenticator.Supports(wxCredential))

	// Test case 3: 不支持Token凭证
	tokenCredential := authentication.NewTokenCredential("token")
	assert.False(t, authenticator.Supports(tokenCredential))
}

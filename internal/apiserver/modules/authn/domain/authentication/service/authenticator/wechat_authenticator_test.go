package authenticator

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/authentication"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	perrors "github.com/fangcun-mount/iam-contracts/pkg/errors"
)

// ==================== Mock 实现 ====================

// MockWeChatAuthPort 模拟微信认证端口
type MockWeChatAuthPort struct {
	ExchangeOpenIDFunc func(ctx context.Context, code, appID string) (string, error)
}

func (m *MockWeChatAuthPort) ExchangeOpenID(ctx context.Context, code, appID string) (string, error) {
	if m.ExchangeOpenIDFunc != nil {
		return m.ExchangeOpenIDFunc(ctx, code, appID)
	}
	return "", errors.New("not implemented")
}

// MockWeChatRepo 模拟微信账号仓储
type MockWeChatRepo struct {
	accounts map[string]*account.WeChatAccount // key: appID:openID
}

func NewMockWeChatRepo() *MockWeChatRepo {
	return &MockWeChatRepo{
		accounts: make(map[string]*account.WeChatAccount),
	}
}

func (m *MockWeChatRepo) Create(ctx context.Context, wechat *account.WeChatAccount) error {
	key := wechat.AppID + ":" + wechat.OpenID
	m.accounts[key] = wechat
	return nil
}

func (m *MockWeChatRepo) FindByAccountID(ctx context.Context, accountID account.AccountID) (*account.WeChatAccount, error) {
	return nil, errors.New("not implemented")
}

func (m *MockWeChatRepo) FindByAppOpenID(ctx context.Context, appID, openID string) (*account.WeChatAccount, error) {
	key := appID + ":" + openID
	wechat, ok := m.accounts[key]
	if !ok {
		return nil, errors.New("wechat account not found")
	}
	return wechat, nil
}

func (m *MockWeChatRepo) UpdateProfile(ctx context.Context, accountID account.AccountID, nickname, avatarURL *string, meta []byte) error {
	return nil
}

func (m *MockWeChatRepo) UpdateUnionID(ctx context.Context, accountID account.AccountID, unionID string) error {
	return nil
}

// ==================== 测试用例 ====================

// TestWeChatAuthenticator_Supports 测试 Supports 方法
func TestWeChatAuthenticator_Supports(t *testing.T) {
	authenticator := NewWeChatAuthenticator(nil, nil, nil)

	tests := []struct {
		name       string
		credential authentication.Credential
		want       bool
	}{
		{
			name:       "支持微信Code凭证",
			credential: authentication.NewWeChatCodeCredential("test_code", "test_appid"),
			want:       true,
		},
		{
			name:       "不支持密码凭证",
			credential: authentication.NewUsernamePasswordCredential("user", "pass"),
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := authenticator.Supports(tt.credential)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestWeChatAuthenticator_Authenticate_Success 测试认证成功场景
func TestWeChatAuthenticator_Authenticate_Success(t *testing.T) {
	ctx := context.Background()
	userID := account.NewUserID(100)
	accountID := account.NewAccountID(200)
	openID := "test_openid_123"
	appID := "test_appid"
	wxCode := "test_code"

	// Mock repositories
	mockAccountRepo := NewMockAccountRepo()
	acc := account.NewAccount(
		userID,
		account.ProviderWeChat,
		account.WithID(accountID),
		account.WithStatus(account.StatusActive),
	)
	mockAccountRepo.accounts[accountID] = &acc

	mockWeChatRepo := NewMockWeChatRepo()
	wxAcc := account.NewWeChatAccount(accountID, appID, openID)
	key := appID + ":" + openID
	mockWeChatRepo.accounts[key] = &wxAcc

	mockWeChatPort := &MockWeChatAuthPort{
		ExchangeOpenIDFunc: func(ctx context.Context, c, a string) (string, error) {
			assert.Equal(t, wxCode, c)
			assert.Equal(t, appID, a)
			return openID, nil
		},
	}

	authenticator := NewWeChatAuthenticator(mockAccountRepo, mockWeChatRepo, mockWeChatPort)
	credential := authentication.NewWeChatCodeCredential(wxCode, appID)

	// 执行认证
	auth, err := authenticator.Authenticate(ctx, credential)

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, auth)
	assert.Equal(t, userID, auth.UserID)
	assert.Equal(t, accountID, auth.AccountID)
	assert.Equal(t, account.ProviderWeChat, auth.Provider)

	assert.Equal(t, openID, auth.Metadata["openid"])
	assert.Equal(t, appID, auth.Metadata["app_id"])
}

// TestWeChatAuthenticator_Authenticate_InvalidCredentialType 测试错误的凭证类型
func TestWeChatAuthenticator_Authenticate_InvalidCredentialType(t *testing.T) {
	ctx := context.Background()
	authenticator := NewWeChatAuthenticator(nil, nil, nil)

	// 使用密码凭证，应该失败
	credential := authentication.NewUsernamePasswordCredential("user", "pass")

	auth, err := authenticator.Authenticate(ctx, credential)

	assert.Error(t, err)
	assert.Nil(t, auth)
	assert.True(t, perrors.IsCode(err, code.ErrInvalidArgument))
}

// TestWeChatAuthenticator_Authenticate_EmptyCode 测试空 Code
func TestWeChatAuthenticator_Authenticate_EmptyCode(t *testing.T) {
	ctx := context.Background()
	authenticator := NewWeChatAuthenticator(nil, nil, nil)

	// Code 为空
	credential := authentication.NewWeChatCodeCredential("", "test_appid")

	auth, err := authenticator.Authenticate(ctx, credential)

	assert.Error(t, err)
	assert.Nil(t, auth)
	// 验证失败返回 ErrInvalidArgument
	assert.True(t, perrors.IsCode(err, code.ErrInvalidArgument))
}

// TestWeChatAuthenticator_Authenticate_EmptyAppID 测试空 AppID
func TestWeChatAuthenticator_Authenticate_EmptyAppID(t *testing.T) {
	ctx := context.Background()
	authenticator := NewWeChatAuthenticator(nil, nil, nil)

	// AppID 为空
	credential := authentication.NewWeChatCodeCredential("test_code", "")

	auth, err := authenticator.Authenticate(ctx, credential)

	assert.Error(t, err)
	assert.Nil(t, auth)
	assert.True(t, perrors.IsCode(err, code.ErrInvalidArgument))
}

// TestWeChatAuthenticator_Authenticate_ExchangeOpenIDFailed 测试微信 API 调用失败
func TestWeChatAuthenticator_Authenticate_ExchangeOpenIDFailed(t *testing.T) {
	ctx := context.Background()

	mockWeChatPort := &MockWeChatAuthPort{
		ExchangeOpenIDFunc: func(ctx context.Context, code, appID string) (string, error) {
			return "", errors.New("wechat api error")
		},
	}

	authenticator := NewWeChatAuthenticator(nil, nil, mockWeChatPort)
	credential := authentication.NewWeChatCodeCredential("test_code", "test_appid")

	auth, err := authenticator.Authenticate(ctx, credential)

	assert.Error(t, err)
	assert.Nil(t, auth)
	assert.True(t, perrors.IsCode(err, code.ErrUnauthenticated))
}

// TestWeChatAuthenticator_Authenticate_WeChatAccountNotFound 测试微信账号不存在
func TestWeChatAuthenticator_Authenticate_WeChatAccountNotFound(t *testing.T) {
	ctx := context.Background()
	openID := "test_openid"
	appID := "test_appid"

	mockWeChatRepo := NewMockWeChatRepo()

	mockWeChatPort := &MockWeChatAuthPort{
		ExchangeOpenIDFunc: func(ctx context.Context, code, appID string) (string, error) {
			return openID, nil
		},
	}

	authenticator := NewWeChatAuthenticator(nil, mockWeChatRepo, mockWeChatPort)
	credential := authentication.NewWeChatCodeCredential("test_code", appID)

	auth, err := authenticator.Authenticate(ctx, credential)

	assert.Error(t, err)
	assert.Nil(t, auth)
	assert.True(t, perrors.IsCode(err, code.ErrUnauthenticated))
}

// TestWeChatAuthenticator_Authenticate_AccountNotFound 测试账号不存在
func TestWeChatAuthenticator_Authenticate_AccountNotFound(t *testing.T) {
	ctx := context.Background()
	openID := "test_openid"
	appID := "test_appid"
	accountID := account.NewAccountID(200)

	mockAccountRepo := NewMockAccountRepo()
	// 故意不添加账号到 repo

	mockWeChatRepo := NewMockWeChatRepo()
	wxAcc := account.NewWeChatAccount(accountID, appID, openID)
	key := appID + ":" + openID
	mockWeChatRepo.accounts[key] = &wxAcc

	mockWeChatPort := &MockWeChatAuthPort{
		ExchangeOpenIDFunc: func(ctx context.Context, code, appID string) (string, error) {
			return openID, nil
		},
	}

	authenticator := NewWeChatAuthenticator(mockAccountRepo, mockWeChatRepo, mockWeChatPort)
	credential := authentication.NewWeChatCodeCredential("test_code", appID)

	auth, err := authenticator.Authenticate(ctx, credential)

	assert.Error(t, err)
	assert.Nil(t, auth)
	assert.True(t, perrors.IsCode(err, code.ErrUnauthenticated))
}

// TestWeChatAuthenticator_Authenticate_AccountNotActive 测试账号未激活
func TestWeChatAuthenticator_Authenticate_AccountNotActive(t *testing.T) {
	ctx := context.Background()
	userID := account.NewUserID(100)
	accountID := account.NewAccountID(200)
	openID := "test_openid"
	appID := "test_appid"

	tests := []struct {
		name   string
		status account.AccountStatus
	}{
		{"账号已停用", account.StatusDisabled},
		{"账号已归档", account.StatusArchived},
		{"账号已删除", account.StatusDeleted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAccountRepo := NewMockAccountRepo()
			acc := account.NewAccount(
				userID,
				account.ProviderWeChat,
				account.WithID(accountID),
				account.WithStatus(tt.status), // 非激活状态
			)
			mockAccountRepo.accounts[accountID] = &acc

			mockWeChatRepo := NewMockWeChatRepo()
			wxAcc := account.NewWeChatAccount(accountID, appID, openID)
			key := appID + ":" + openID
			mockWeChatRepo.accounts[key] = &wxAcc

			mockWeChatPort := &MockWeChatAuthPort{
				ExchangeOpenIDFunc: func(ctx context.Context, code, appID string) (string, error) {
					return openID, nil
				},
			}

			authenticator := NewWeChatAuthenticator(mockAccountRepo, mockWeChatRepo, mockWeChatPort)
			credential := authentication.NewWeChatCodeCredential("test_code", appID)

			auth, err := authenticator.Authenticate(ctx, credential)

			assert.Error(t, err)
			assert.Nil(t, auth)
			assert.True(t, perrors.IsCode(err, code.ErrUnauthenticated))
		})
	}
}

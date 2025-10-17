package jwt_test

import (
"context"
"crypto/rand"
"crypto/rsa"
"testing"
"time"

"github.com/stretchr/testify/assert"
"github.com/stretchr/testify/mock"
"github.com/stretchr/testify/require"

"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/account"
"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/authentication"
"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/jwks"
jwtGen "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/infra/jwt"
"github.com/fangcun-mount/iam-contracts/pkg/util/idutil"
"github.com/golang-jwt/jwt/v4"
)

// MockKeyManagementService mock 实现
type MockKeyManagementService struct {
	mock.Mock
}

func (m *MockKeyManagementService) CreateKey(ctx context.Context, alg string, notBefore, notAfter *time.Time) (*jwks.Key, error) {
	args := m.Called(ctx, alg, notBefore, notAfter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*jwks.Key), args.Error(1)
}

func (m *MockKeyManagementService) GetActiveKey(ctx context.Context) (*jwks.Key, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*jwks.Key), args.Error(1)
}

func (m *MockKeyManagementService) GetKeyByKid(ctx context.Context, kid string) (*jwks.Key, error) {
	args := m.Called(ctx, kid)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*jwks.Key), args.Error(1)
}

func (m *MockKeyManagementService) RetireKey(ctx context.Context, kid string) error {
	args := m.Called(ctx, kid)
	return args.Error(0)
}

func (m *MockKeyManagementService) ForceRetireKey(ctx context.Context, kid string) error {
	args := m.Called(ctx, kid)
	return args.Error(0)
}

func (m *MockKeyManagementService) EnterGracePeriod(ctx context.Context, kid string) error {
	args := m.Called(ctx, kid)
	return args.Error(0)
}

func (m *MockKeyManagementService) CleanupExpiredKeys(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *MockKeyManagementService) ListKeys(ctx context.Context, status jwks.KeyStatus, limit, offset int) ([]*jwks.Key, int64, error) {
	args := m.Called(ctx, status, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*jwks.Key), args.Get(1).(int64), args.Error(2)
}

// MockPrivateKeyResolver mock 实现
type MockPrivateKeyResolver struct {
	mock.Mock
}

func (m *MockPrivateKeyResolver) ResolveSigningKey(ctx context.Context, kid, alg string) (any, error) {
	args := m.Called(ctx, kid, alg)
	return args.Get(0), args.Error(1)
}

// TestJWTGeneratorWithJWKS 集成测试：验证 JWT 生成器使用 JWKS 密钥签名
func TestJWTGeneratorWithJWKS(t *testing.T) {
	ctx := context.Background()

	// 1. 创建 mock 依赖
	mockKeyMgmt := new(MockKeyManagementService)
	mockPrivKeyResolver := new(MockPrivateKeyResolver)

	// 2. 生成测试用 RSA 密钥对
	rsaPrivKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// 3. 创建测试密钥实体
	testKey := &jwks.Key{
		Kid:    "test-key-123",
		Status: jwks.KeyActive,
		JWK: jwks.PublicJWK{
			Kty: "RSA",
			Use: "sig",
			Alg: "RS256",
			Kid: "test-key-123",
		},
	}

	// 4. 设置 mock 预期
	mockKeyMgmt.On("GetActiveKey", ctx).Return(testKey, nil)
	mockPrivKeyResolver.On("ResolveSigningKey", ctx, "test-key-123", "RS256").Return(rsaPrivKey, nil)

	// 5. 创建 JWT 生成器
	generator := jwtGen.NewGenerator("iam-auth-service", mockKeyMgmt, mockPrivKeyResolver)

	// 6. 创建测试用户认证信息
	auth := &authentication.Authentication{
		UserID:    account.NewUserID(12345),
		AccountID: account.AccountID(idutil.NewID(67890)),
	}

	// 7. 生成访问令牌（应该使用 mock 的 RSA 密钥签名）
	token, err := generator.GenerateAccessToken(auth, 1*time.Hour)
	require.NoError(t, err)
	assert.NotEmpty(t, token.Value)

	// 8. 解析 JWT 并验证 header 中的 kid
	parsedToken, err := jwt.Parse(token.Value, nil)
	require.Error(t, err) // 预期会因为没有提供验证密钥而失败
	require.NotNil(t, parsedToken)

	// 验证 kid 存在且等于创建的密钥 ID
	kidInterface, ok := parsedToken.Header["kid"]
	require.True(t, ok, "JWT header should contain kid")
	kid, ok := kidInterface.(string)
	require.True(t, ok, "kid should be string")
	assert.Equal(t, testKey.Kid, kid, "kid in JWT should match the active key")

	// 验证签名算法是 RS256
	assert.Equal(t, "RS256", parsedToken.Header["alg"])

	// 9. 验证 claims（虽然签名验证失败，但 claims 还是能解析出来）
	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	require.True(t, ok)
	assert.Equal(t, "iam-auth-service", claims["iss"])
	assert.Equal(t, float64(12345), claims["user_id"])
	assert.Equal(t, float64(67890), claims["account_id"])
	assert.NotEmpty(t, claims["jti"])
	assert.NotEmpty(t, claims["iat"])
	assert.NotEmpty(t, claims["exp"])

	// 验证 mock 被调用
	mockKeyMgmt.AssertExpectations(t)
	mockPrivKeyResolver.AssertExpectations(t)

	t.Logf("✅ JWT generated successfully with kid=%s", kid)
	t.Logf("Token: %s", token.Value)
}

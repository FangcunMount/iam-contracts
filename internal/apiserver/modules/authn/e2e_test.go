// Package authn 端到端集成测试
// 验证完整的 JWT 签名 → JWKS 发布 → JWT 验证流程
package authn_test

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/FangcunMount/component-base/pkg/util/idutil"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/application/jwks"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/authentication"
	jwksDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/jwks"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/jwks/port/driven"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/jwks/service"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/infra/crypto"
	jwtGen "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/infra/jwt"
	"github.com/FangcunMount/iam-contracts/pkg/log"
)

// InMemoryKeyRepository 内存密钥仓储（用于测试）
type InMemoryKeyRepository struct {
	mu   sync.RWMutex
	keys map[string]*jwksDomain.Key
}

// NewInMemoryKeyRepository 创建内存密钥仓储
func NewInMemoryKeyRepository() *InMemoryKeyRepository {
	return &InMemoryKeyRepository{
		keys: make(map[string]*jwksDomain.Key),
	}
}

// Save 保存新密钥
func (r *InMemoryKeyRepository) Save(ctx context.Context, key *jwksDomain.Key) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.keys[key.Kid]; exists {
		return errors.New("key already exists")
	}

	r.keys[key.Kid] = key
	return nil
}

// Update 更新密钥
func (r *InMemoryKeyRepository) Update(ctx context.Context, key *jwksDomain.Key) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.keys[key.Kid]; !exists {
		return errors.New("key not found")
	}

	r.keys[key.Kid] = key
	return nil
}

// Delete 删除密钥
func (r *InMemoryKeyRepository) Delete(ctx context.Context, kid string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.keys[kid]; !exists {
		return errors.New("key not found")
	}

	delete(r.keys, kid)
	return nil
}

// FindByKid 根据 kid 查询密钥
func (r *InMemoryKeyRepository) FindByKid(ctx context.Context, kid string) (*jwksDomain.Key, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key, exists := r.keys[kid]
	if !exists {
		return nil, errors.New("key not found")
	}

	return key, nil
}

// FindByStatus 根据状态查询密钥列表
func (r *InMemoryKeyRepository) FindByStatus(ctx context.Context, status jwksDomain.KeyStatus) ([]*jwksDomain.Key, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*jwksDomain.Key
	for _, key := range r.keys {
		if key.Status == status {
			result = append(result, key)
		}
	}

	return result, nil
}

// FindPublishable 查询可发布的密钥（Active + Grace 状态且未过期）
func (r *InMemoryKeyRepository) FindPublishable(ctx context.Context) ([]*jwksDomain.Key, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	now := time.Now()
	var result []*jwksDomain.Key

	for _, key := range r.keys {
		if (key.Status == jwksDomain.KeyActive || key.Status == jwksDomain.KeyGrace) &&
			(key.NotAfter == nil || key.NotAfter.After(now)) {
			result = append(result, key)
		}
	}

	return result, nil
}

// FindExpired 查询已过期的密钥
func (r *InMemoryKeyRepository) FindExpired(ctx context.Context) ([]*jwksDomain.Key, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	now := time.Now()
	var result []*jwksDomain.Key

	for _, key := range r.keys {
		if key.NotAfter != nil && key.NotAfter.Before(now) {
			result = append(result, key)
		}
	}

	return result, nil
}

// FindAll 查询所有密钥（分页）
func (r *InMemoryKeyRepository) FindAll(ctx context.Context, limit, offset int) ([]*jwksDomain.Key, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	total := int64(len(r.keys))
	result := make([]*jwksDomain.Key, 0, len(r.keys))

	for _, key := range r.keys {
		result = append(result, key)
	}

	// 简单分页
	if offset < len(result) {
		end := offset + limit
		if end > len(result) {
			end = len(result)
		}
		result = result[offset:end]
	} else {
		result = []*jwksDomain.Key{}
	}

	return result, total, nil
}

// CountByStatus 统计指定状态的密钥数量
func (r *InMemoryKeyRepository) CountByStatus(ctx context.Context, status jwksDomain.KeyStatus) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := int64(0)
	for _, key := range r.keys {
		if key.Status == status {
			count++
		}
	}

	return count, nil
}

// InMemoryPrivateKeyResolver 内存私钥解析器（用于测试）
type InMemoryPrivateKeyResolver struct {
	mu          sync.RWMutex
	privateKeys map[string]any // kid -> private key
}

// NewInMemoryPrivateKeyResolver 创建内存私钥解析器
func NewInMemoryPrivateKeyResolver() *InMemoryPrivateKeyResolver {
	return &InMemoryPrivateKeyResolver{
		privateKeys: make(map[string]any),
	}
}

// StoreKey 存储私钥（测试辅助方法）
func (r *InMemoryPrivateKeyResolver) StoreKey(kid string, privateKey any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.privateKeys[kid] = privateKey
}

// ResolveSigningKey 解析私钥用于签名
func (r *InMemoryPrivateKeyResolver) ResolveSigningKey(ctx context.Context, kid, alg string) (any, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	privateKey, exists := r.privateKeys[kid]
	if !exists {
		return nil, errors.New("Key not found")
	}

	return privateKey, nil
}

// KeyGeneratorWithInterceptor 密钥生成器包装器（用于测试）
// 拦截生成的私钥并存储到 PrivateKeyResolver
type KeyGeneratorWithInterceptor struct {
	generator crypto.RSAKeyGenerator
	resolver  *InMemoryPrivateKeyResolver
}

// NewKeyGeneratorWithInterceptor 创建带拦截的密钥生成器
func NewKeyGeneratorWithInterceptor(resolver *InMemoryPrivateKeyResolver) *KeyGeneratorWithInterceptor {
	return &KeyGeneratorWithInterceptor{
		generator: *crypto.NewRSAKeyGenerator(),
		resolver:  resolver,
	}
}

// GenerateKeyPair 生成密钥对并拦截私钥
func (g *KeyGeneratorWithInterceptor) GenerateKeyPair(ctx context.Context, algorithm, kid string) (*driven.KeyPair, error) {
	keyPair, err := g.generator.GenerateKeyPair(ctx, algorithm, kid)
	if err != nil {
		return nil, err
	}

	// 拦截并存储私钥
	g.resolver.StoreKey(kid, keyPair.PrivateKey)

	return keyPair, nil
}

// SupportedAlgorithms 返回支持的算法
func (g *KeyGeneratorWithInterceptor) SupportedAlgorithms() []string {
	return g.generator.SupportedAlgorithms()
}

// TestE2E_JWT_JWKS_Verification 端到端测试：JWT 签名 → JWKS 发布 → JWT 验证
func TestE2E_JWT_JWKS_Verification(t *testing.T) {
	ctx := context.Background()

	// ========== 第 1 步：设置基础设施层 ==========
	t.Log("📦 Step 1: 设置基础设施层...")

	// 1.1 创建内存密钥仓库
	keyRepo := NewInMemoryKeyRepository()

	// 1.2 创建内存私钥解析器
	privKeyResolver := NewInMemoryPrivateKeyResolver()

	// 1.3 创建带拦截的密钥生成器（自动存储私钥）
	keyGenerator := NewKeyGeneratorWithInterceptor(privKeyResolver)

	t.Log("✅ 基础设施层就绪")

	// ========== 第 2 步：设置领域服务层 ==========
	t.Log("🔧 Step 2: 设置领域服务层...")

	// 2.1 密钥管理服务
	keyManager := service.NewKeyManager(keyRepo, keyGenerator)

	// 2.2 密钥集构建服务
	keySetBuilder := service.NewKeySetBuilder(keyRepo)

	t.Log("✅ 领域服务层就绪")

	// ========== 第 3 步：设置应用服务层 ==========
	t.Log("⚙️  Step 3: 设置应用服务层...")

	// 3.1 创建 Logger
	logger := log.New(log.NewOptions())

	// 3.2 密钥管理应用服务
	keyMgmtApp := jwks.NewKeyManagementAppService(keyManager, logger)

	// 3.3 密钥发布应用服务
	keyPublishApp := jwks.NewKeyPublishAppService(keySetBuilder, logger)

	t.Log("✅ 应用服务层就绪")

	// ========== 第 4 步：创建 RSA 密钥 ==========
	t.Log("🔑 Step 4: 创建 RSA 密钥...")

	keyReq := jwks.CreateKeyRequest{
		Algorithm: "RS256",
		NotBefore: nil,
		NotAfter:  nil,
	}

	keyResp, err := keyMgmtApp.CreateKey(ctx, keyReq)
	require.NoError(t, err)
	require.NotNil(t, keyResp)

	t.Logf("✅ 密钥创建成功 (kid=%s, alg=%s, status=%s)",
		keyResp.Kid, keyResp.Algorithm, keyResp.Status)

	// ========== 第 5 步：签发 JWT ==========
	t.Log("✍️  Step 5: 使用活跃密钥签发 JWT...")

	// 5.1 创建 JWT Generator
	generator := jwtGen.NewGenerator("iam-auth-service", keyManager, privKeyResolver)

	// 5.2 创建测试用户认证信息
	auth := &authentication.Authentication{
		UserID:    account.NewUserID(12345),
		AccountID: account.AccountID(idutil.NewID(67890)),
	}

	// 5.3 生成访问令牌
	token, err := generator.GenerateAccessToken(auth, 1*time.Hour)
	require.NoError(t, err)
	require.NotEmpty(t, token.Value)

	t.Logf("✅ JWT 签发成功")
	t.Logf("Token: %s", token.Value)

	// ========== 第 6 步：发布 JWKS ==========
	t.Log("📢 Step 6: 发布 JWKS（模拟 GET /.well-known/jwks.json）...")

	// 6.1 构建 JWKS
	jwksResp, err := keyPublishApp.BuildJWKS(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, jwksResp.JWKS)

	// 6.2 解析 JWKS JSON
	var jwksObj jwksDomain.JWKS
	err = json.Unmarshal(jwksResp.JWKS, &jwksObj)
	require.NoError(t, err)

	t.Logf("✅ JWKS 发布成功")
	t.Logf("JWKS 包含 %d 个密钥", len(jwksObj.Keys))
	t.Logf("JWKS JSON: %s", string(jwksResp.JWKS))

	// 验证 JWKS 包含刚创建的密钥
	foundKey := jwksObj.FindByKid(keyResp.Kid)
	require.NotNil(t, foundKey, "JWKS 应该包含刚创建的密钥")
	assert.Equal(t, "RSA", foundKey.Kty)
	assert.Equal(t, "sig", foundKey.Use)
	assert.Equal(t, "RS256", foundKey.Alg)
	assert.Equal(t, keyResp.Kid, foundKey.Kid)

	// ========== 第 7 步：验证 JWT 签名 ==========
	t.Log("🔍 Step 7: 验证 JWT 签名...")

	// 7.1 解析 JWT header 提取 kid
	parsedToken, _, err := new(jwt.Parser).ParseUnverified(token.Value, jwt.MapClaims{})
	require.NoError(t, err)

	kidInterface, ok := parsedToken.Header["kid"]
	require.True(t, ok, "JWT header 应该包含 kid")
	kid := kidInterface.(string)

	t.Logf("JWT kid=%s", kid)

	// 7.2 从 JWKS 获取对应的公钥
	publicJWK := jwksObj.FindByKid(kid)
	require.NotNil(t, publicJWK, "应该能从 JWKS 找到对应的公钥")

	// 7.3 从 JWK 构造 RSA 公钥
	rsaPublicKey, err := parseRSAPublicKeyFromJWK(publicJWK)
	require.NoError(t, err)

	t.Logf("✅ 从 JWKS 提取公钥成功")

	// 7.4 使用公钥验证 JWT 签名
	verified, err := jwt.Parse(token.Value, func(token *jwt.Token) (interface{}, error) {
		// 验证签名方法
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, jwt.NewValidationError("unexpected signing method", jwt.ValidationErrorSignatureInvalid)
		}
		return rsaPublicKey, nil
	})

	require.NoError(t, err)
	require.NotNil(t, verified)
	require.True(t, verified.Valid, "JWT 签名验证应该通过")

	t.Log("✅ JWT 签名验证成功！")

	// 7.5 验证 claims
	claims, ok := verified.Claims.(jwt.MapClaims)
	require.True(t, ok)

	assert.Equal(t, "iam-auth-service", claims["iss"])
	assert.Equal(t, float64(12345), claims["user_id"])
	assert.Equal(t, float64(67890), claims["account_id"])

	t.Log("✅ JWT claims 验证成功！")

	// ========== 第 8 步：测试密钥轮换场景 ==========
	t.Log("🔄 Step 8: 测试密钥轮换场景...")

	// 8.1 旧密钥进入宽限期
	err = keyManager.EnterGracePeriod(ctx, keyResp.Kid)
	require.NoError(t, err)

	t.Logf("✅ 旧密钥进入宽限期 (kid=%s)", keyResp.Kid)

	// 8.2 创建新密钥
	newKeyResp, err := keyMgmtApp.CreateKey(ctx, keyReq)
	require.NoError(t, err)

	t.Logf("✅ 新密钥创建成功 (kid=%s)", newKeyResp.Kid)

	// 8.3 使用新密钥签发新 JWT
	newToken, err := generator.GenerateAccessToken(auth, 1*time.Hour)
	require.NoError(t, err)

	// 验证新 JWT 使用新密钥
	newParsedToken, _, _ := new(jwt.Parser).ParseUnverified(newToken.Value, jwt.MapClaims{})
	newKid := newParsedToken.Header["kid"].(string)
	assert.Equal(t, newKeyResp.Kid, newKid, "新 JWT 应该使用新密钥")
	assert.NotEqual(t, kid, newKid, "新旧密钥 ID 应该不同")

	t.Logf("✅ 新 JWT 使用新密钥签发 (kid=%s)", newKid)

	// 8.4 重新发布 JWKS，应该同时包含新旧密钥
	newJWKSResp, err := keyPublishApp.BuildJWKS(ctx)
	require.NoError(t, err)

	var newJWKSObj jwksDomain.JWKS
	err = json.Unmarshal(newJWKSResp.JWKS, &newJWKSObj)
	require.NoError(t, err)

	assert.Equal(t, 2, len(newJWKSObj.Keys), "JWKS 应该包含 2 个密钥（1个Active + 1个Grace）")

	// 验证旧 JWT 仍然可以验证（使用 Grace 密钥）
	oldPublicJWK := newJWKSObj.FindByKid(kid)
	require.NotNil(t, oldPublicJWK, "JWKS 应该仍然包含旧密钥（Grace 状态）")

	oldRSAPublicKey, err := parseRSAPublicKeyFromJWK(oldPublicJWK)
	require.NoError(t, err)

	oldVerified, err := jwt.Parse(token.Value, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, jwt.NewValidationError("unexpected signing method", jwt.ValidationErrorSignatureInvalid)
		}
		return oldRSAPublicKey, nil
	})

	require.NoError(t, err)
	require.True(t, oldVerified.Valid, "旧 JWT 仍然应该能验证（使用 Grace 密钥）")

	t.Log("✅ 旧 JWT 仍然可以验证（Grace 密钥）")

	// 验证新 JWT 可以验证
	newPublicJWK := newJWKSObj.FindByKid(newKid)
	require.NotNil(t, newPublicJWK)

	newRSAPublicKey, err := parseRSAPublicKeyFromJWK(newPublicJWK)
	require.NoError(t, err)

	newVerified, err := jwt.Parse(newToken.Value, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, jwt.NewValidationError("unexpected signing method", jwt.ValidationErrorSignatureInvalid)
		}
		return newRSAPublicKey, nil
	})

	require.NoError(t, err)
	require.True(t, newVerified.Valid, "新 JWT 应该能验证")

	t.Log("✅ 新 JWT 验证成功")

	// ========== 测试总结 ==========
	separator := strings.Repeat("=", 60)
	t.Log("\n" + separator)
	t.Log("🎉 端到端测试全部通过！")
	t.Log(separator)
	t.Log("验证流程:")
	t.Log("  1️⃣  创建 RSA 密钥 ✅")
	t.Log("  2️⃣  使用私钥签发 JWT ✅")
	t.Log("  3️⃣  发布 JWKS 公钥集 ✅")
	t.Log("  4️⃣  从 JWKS 提取公钥 ✅")
	t.Log("  5️⃣  验证 JWT 签名 ✅")
	t.Log("  6️⃣  密钥轮换（Grace 期）✅")
	t.Log("  7️⃣  新旧 JWT 共存验证 ✅")
	t.Log(separator)
}

// parseRSAPublicKeyFromJWK 从 JWK 解析 RSA 公钥
// 这是一个辅助函数，将 JWK (N, E) 转换为 *rsa.PublicKey
func parseRSAPublicKeyFromJWK(jwk *jwksDomain.PublicJWK) (*rsa.PublicKey, error) {
	if jwk.Kty != "RSA" {
		return nil, jwt.NewValidationError("expected RSA key type", jwt.ValidationErrorSignatureInvalid)
	}

	if jwk.N == nil || jwk.E == nil {
		return nil, jwt.NewValidationError("missing N or E in RSA JWK", jwt.ValidationErrorSignatureInvalid)
	}

	// 解码 base64url 编码的 N (modulus)
	nBytes, err := base64.RawURLEncoding.DecodeString(*jwk.N)
	if err != nil {
		return nil, jwt.NewValidationError("failed to decode N", jwt.ValidationErrorSignatureInvalid)
	}

	// 解码 base64url 编码的 E (exponent)
	eBytes, err := base64.RawURLEncoding.DecodeString(*jwk.E)
	if err != nil {
		return nil, jwt.NewValidationError("failed to decode E", jwt.ValidationErrorSignatureInvalid)
	}

	// 构造 RSA 公钥
	n := new(big.Int).SetBytes(nBytes)
	e := 0
	for _, b := range eBytes {
		e = e<<8 + int(b)
	}

	rsaPublicKey := &rsa.PublicKey{
		N: n,
		E: e,
	}

	return rsaPublicKey, nil
}

// TestE2E_JWKS_Caching 测试 JWKS 缓存机制
func TestE2E_JWKS_Caching(t *testing.T) {
	ctx := context.Background()

	// 设置
	keyRepo := NewInMemoryKeyRepository()
	keyGenerator := crypto.NewRSAKeyGenerator()
	keyManager := service.NewKeyManager(keyRepo, keyGenerator)
	keySetBuilder := service.NewKeySetBuilder(keyRepo)
	logger := log.New(log.NewOptions())
	keyPublishApp := jwks.NewKeyPublishAppService(keySetBuilder, logger)
	keyMgmtApp := jwks.NewKeyManagementAppService(keyManager, logger)

	// 创建密钥
	keyReq := jwks.CreateKeyRequest{Algorithm: "RS256"}
	_, err := keyMgmtApp.CreateKey(ctx, keyReq)
	require.NoError(t, err)

	// 第一次构建 JWKS
	resp1, err := keyPublishApp.BuildJWKS(ctx)
	require.NoError(t, err)

	t.Logf("第一次构建 JWKS: ETag=%s", resp1.ETag)

	// 第二次构建 JWKS（无变化）
	resp2, err := keyPublishApp.BuildJWKS(ctx)
	require.NoError(t, err)

	// ETag 应该相同（因为密钥集未变化）
	assert.Equal(t, resp1.ETag, resp2.ETag, "相同密钥集的 ETag 应该一致")

	t.Logf("第二次构建 JWKS: ETag=%s (相同)", resp2.ETag)

	// 创建新密钥
	_, err = keyMgmtApp.CreateKey(ctx, keyReq)
	require.NoError(t, err)

	// 第三次构建 JWKS（有变化）
	resp3, err := keyPublishApp.BuildJWKS(ctx)
	require.NoError(t, err)

	// ETag 应该不同
	assert.NotEqual(t, resp1.ETag, resp3.ETag, "密钥集变化后 ETag 应该改变")

	t.Logf("第三次构建 JWKS: ETag=%s (不同)", resp3.ETag)

	t.Log("✅ JWKS 缓存机制测试通过")
}

package authn_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/FangcunMount/component-base/pkg/log"
	"github.com/FangcunMount/component-base/pkg/util/idutil"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/application/jwks"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/authentication"
	jwksDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/jwks"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/jwks/service"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/infra/crypto"
	jwtGen "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/infra/jwt"
)

// TestE2E_Production_JWKS_With_File_Storage 生产环境端到端测试
// 使用真实的文件存储系统，验证完整的密钥生命周期
func TestE2E_Production_JWKS_With_File_Storage(t *testing.T) {
	ctx := context.Background()

	// ========== 第 1 步：设置生产环境基础设施 ==========
	t.Log("📦 Step 1: 设置生产环境基础设施...")

	// 1.1 创建临时目录模拟生产环境
	keysDir := t.TempDir()
	t.Logf("Keys directory: %s", keysDir)

	// 1.2 创建密钥仓库（内存版本用于测试）
	keyRepo := NewInMemoryKeyRepository()

	// 1.3 创建私钥存储（PEM 文件存储）
	privateKeyStorage := crypto.NewPEMPrivateKeyStorage(keysDir)

	// 1.4 创建带持久化的 RSA 密钥生成器
	keyGenerator := crypto.NewRSAKeyGeneratorWithStorage(privateKeyStorage)

	// 1.5 创建私钥解析器（从 PEM 文件读取）
	privKeyResolver := crypto.NewPEMPrivateKeyResolver(keysDir)

	t.Log("✅ 生产环境基础设施就绪")

	// ========== 第 2 步：设置领域服务 ==========
	t.Log("🔧 Step 2: 设置领域服务...")

	keyManager := service.NewKeyManager(keyRepo, keyGenerator)
	keySetBuilder := service.NewKeySetBuilder(keyRepo)

	t.Log("✅ 领域服务就绪")

	// ========== 第 3 步：设置应用服务 ==========
	t.Log("⚙️  Step 3: 设置应用服务...")

	logger := log.New(log.NewOptions())
	keyMgmtApp := jwks.NewKeyManagementAppService(keyManager, logger)
	keyPublishApp := jwks.NewKeyPublishAppService(keySetBuilder, logger)

	t.Log("✅ 应用服务就绪")

	// ========== 第 4 步：创建密钥（自动持久化） ==========
	t.Log("🔑 Step 4: 创建 RSA 密钥（私钥自动保存到文件）...")

	keyReq := jwks.CreateKeyRequest{
		Algorithm: "RS256",
		NotBefore: nil,
		NotAfter:  nil,
	}

	keyResp, err := keyMgmtApp.CreateKey(ctx, keyReq)
	require.NoError(t, err)
	require.NotNil(t, keyResp)

	t.Logf("✅ 密钥创建成功 (kid=%s)", keyResp.Kid)

	// 验证私钥文件已创建
	exists, err := privateKeyStorage.KeyExists(ctx, keyResp.Kid)
	require.NoError(t, err)
	assert.True(t, exists, "私钥文件应该已创建")

	t.Logf("✅ 私钥文件已保存: %s.pem", keyResp.Kid)

	// ========== 第 5 步：使用私钥签发 JWT ==========
	t.Log("✍️  Step 5: 使用持久化的私钥签发 JWT...")

	generator := jwtGen.NewGenerator("iam-auth-service", keyManager, privKeyResolver)

	auth := &authentication.Authentication{
		UserID:    account.NewUserID(12345),
		AccountID: account.AccountID(idutil.NewID(67890)),
	}

	token, err := generator.GenerateAccessToken(auth, 1*time.Hour)
	require.NoError(t, err)
	require.NotEmpty(t, token.Value)

	t.Log("✅ JWT 签发成功（使用文件系统中的私钥）")
	t.Logf("Token: %s...", token.Value[:50])

	// ========== 第 6 步：发布 JWKS ==========
	t.Log("📢 Step 6: 发布 JWKS...")

	jwksResp, err := keyPublishApp.BuildJWKS(ctx)
	require.NoError(t, err)

	var jwksObj jwksDomain.JWKS
	err = json.Unmarshal(jwksResp.JWKS, &jwksObj)
	require.NoError(t, err)

	t.Logf("✅ JWKS 发布成功 (包含 %d 个密钥)", len(jwksObj.Keys))

	// ========== 第 7 步：验证 JWT ==========
	t.Log("🔍 Step 7: 使用 JWKS 验证 JWT 签名...")

	parsedToken, _, err := new(jwt.Parser).ParseUnverified(token.Value, jwt.MapClaims{})
	require.NoError(t, err)

	kid := parsedToken.Header["kid"].(string)
	publicJWK := jwksObj.FindByKid(kid)
	require.NotNil(t, publicJWK)

	rsaPublicKey, err := parseRSAPublicKeyFromJWK(publicJWK)
	require.NoError(t, err)

	verified, err := jwt.Parse(token.Value, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, jwt.NewValidationError("unexpected signing method", jwt.ValidationErrorSignatureInvalid)
		}
		return rsaPublicKey, nil
	})

	require.NoError(t, err)
	require.True(t, verified.Valid)

	t.Log("✅ JWT 签名验证成功！")

	// ========== 第 8 步：测试密钥清理 ==========
	t.Log("🗑️  Step 8: 测试密钥清理（删除私钥文件）...")

	// 创建新密钥并立即退役旧密钥
	newKeyResp, err := keyMgmtApp.CreateKey(ctx, keyReq)
	require.NoError(t, err)

	// 强制退役旧密钥
	err = keyManager.ForceRetireKey(ctx, keyResp.Kid)
	require.NoError(t, err)

	// 删除旧密钥的私钥文件
	err = privateKeyStorage.DeletePrivateKey(ctx, keyResp.Kid)
	require.NoError(t, err)

	// 验证文件已删除
	exists, err = privateKeyStorage.KeyExists(ctx, keyResp.Kid)
	require.NoError(t, err)
	assert.False(t, exists, "旧密钥文件应该已删除")

	// 验证新密钥文件仍然存在
	exists, err = privateKeyStorage.KeyExists(ctx, newKeyResp.Kid)
	require.NoError(t, err)
	assert.True(t, exists, "新密钥文件应该存在")

	t.Log("✅ 密钥清理成功")

	// ========== 第 9 步：列出所有密钥文件 ==========
	t.Log("📋 Step 9: 列出文件系统中的所有密钥...")

	kids, err := privateKeyStorage.ListKeys(ctx)
	require.NoError(t, err)

	t.Logf("✅ 文件系统中共有 %d 个密钥文件: %v", len(kids), kids)
	assert.Contains(t, kids, newKeyResp.Kid, "新密钥应该在列表中")
	assert.NotContains(t, kids, keyResp.Kid, "旧密钥不应该在列表中")

	// ========== 测试总结 ==========
	separator := strings.Repeat("=", 60)
	t.Log("\n" + separator)
	t.Log("🎉 生产环境端到端测试全部通过！")
	t.Log(separator)
	t.Log("验证流程:")
	t.Log("  1️⃣  创建密钥 + 自动保存私钥到文件 ✅")
	t.Log("  2️⃣  从文件读取私钥签发 JWT ✅")
	t.Log("  3️⃣  发布 JWKS 公钥集 ✅")
	t.Log("  4️⃣  验证 JWT 签名 ✅")
	t.Log("  5️⃣  密钥清理（删除文件）✅")
	t.Log("  6️⃣  列出所有密钥文件 ✅")
	t.Log(separator)
}

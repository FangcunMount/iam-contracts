package main

import (
	"context"
	"fmt"
	"time"

	jwksApp "github.com/FangcunMount/iam-contracts/internal/apiserver/application/authn/jwks"
	jwksService "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/jwks/service"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/infra/crypto"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/infra/jwt"
	jwksMysql "github.com/FangcunMount/iam-contracts/internal/apiserver/infra/mysql/jwks"
)

// ==================== JWKS Seed 函数 ====================

// seedJWKS 生成 JWKS 密钥对
//
// 业务说明：
// - 生成用于 JWT 签名的 RSA 密钥对
// - 私钥保存到本地文件系统（PEM 格式）
// - 公钥保存到数据库，通过 JWKS 端点对外公开
// - 算法：RS256，密钥长度 2048 位
// - 有效期：1 年
//
// 幂等性：每次执行都会创建新密钥（不影响旧密钥）
func seedJWKS(ctx context.Context, deps *dependencies) error {
	if err := ensureDir(deps.KeysDir); err != nil {
		return fmt.Errorf("ensure keys dir: %w", err)
	}

	keyRepo := jwksMysql.NewKeyRepository(deps.DB)
	privateStorage := crypto.NewPEMPrivateKeyStorage(deps.KeysDir)
	keyGenerator := crypto.NewRSAKeyGeneratorWithStorage(privateStorage)
	keyManager := jwksService.NewKeyManager(keyRepo, keyGenerator)

	keyResolver := crypto.NewPEMPrivateKeyResolver(deps.KeysDir)
	jwtGenerator := jwt.NewGenerator("iam-seed", keyManager, keyResolver)
	_ = jwtGenerator // ensure generator initialised for side effects

	manager := jwksApp.NewKeyManagementAppService(keyManager, deps.Logger)
	now := time.Now()
	req := jwksApp.CreateKeyRequest{
		Algorithm: "RS256", // 使用 RS256 算法
		NotBefore: &now,
		NotAfter:  ptrTime(now.AddDate(1, 0, 0)),
	}

	if _, err := manager.CreateKey(ctx, req); err != nil {
		return fmt.Errorf("create jwks key: %w", err)
	}

	deps.Logger.Infow("✅ JWKS 密钥已生成", "algorithm", "RS256", "validity", "1 year")
	return nil
}

// ==================== 辅助函数 ====================

// ptrTime 返回时间的指针
func ptrTime(t time.Time) *time.Time {
	return &t
}

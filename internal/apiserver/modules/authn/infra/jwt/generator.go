// Package jwt JWT 令牌生成器实现
package jwt

import (
	"context"
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"

	"github.com/FangcunMount/component-base/pkg/util/idutil"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/authentication"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/jwks/port/driven"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/jwks/port/driving"
)

// Generator JWT 令牌生成器（使用 JWKS 的 RSA 密钥签名）
type Generator struct {
	issuer          string                       // 颁发者
	keyMgmt         driving.KeyManagementService // 密钥管理服务
	privKeyResolver driven.PrivateKeyResolver    // 私钥解析器
}

// NewGenerator 创建 JWT 生成器
func NewGenerator(
	issuer string,
	keyMgmt driving.KeyManagementService,
	privKeyResolver driven.PrivateKeyResolver,
) *Generator {
	return &Generator{
		issuer:          issuer,
		keyMgmt:         keyMgmt,
		privKeyResolver: privKeyResolver,
	}
}

// CustomClaims 自定义 JWT Claims
type CustomClaims struct {
	UserID    uint64 `json:"user_id"`
	AccountID uint64 `json:"account_id"`
	jwt.StandardClaims
}

// GenerateAccessToken 生成访问令牌（JWT）
// 使用 JWKS 中的活跃 RSA 密钥进行签名
func (g *Generator) GenerateAccessToken(auth *authentication.Authentication, expiresIn time.Duration) (*authentication.Token, error) {
	ctx := context.Background() // TODO: 从参数传递 context
	now := time.Now()
	tokenID := idutil.NewID(0).String() // 生成唯一 Token ID

	// 获取当前活跃的密钥
	activeKey, err := g.keyMgmt.GetActiveKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get active key: %w", err)
	}

	// 获取私钥
	privKey, err := g.privKeyResolver.ResolveSigningKey(ctx, activeKey.Kid, activeKey.JWK.Alg)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve private key: %w", err)
	}

	// 确保私钥是 RSA 类型
	rsaPrivKey, ok := privKey.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("expected RSA private key, got %T", privKey)
	}

	claims := CustomClaims{
		UserID:    auth.UserID.Uint64(),
		AccountID: idutil.ID(auth.AccountID).Uint64(),
		StandardClaims: jwt.StandardClaims{
			Id:        tokenID,
			Issuer:    g.issuer,
			IssuedAt:  now.Unix(),
			ExpiresAt: now.Add(expiresIn).Unix(),
			NotBefore: now.Unix(),
		},
	}

	// 创建 token，使用 RS256 签名算法
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	// 在 header 中设置 kid（密钥 ID）
	token.Header["kid"] = activeKey.Kid

	// 使用 RSA 私钥签名
	tokenString, err := token.SignedString(rsaPrivKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign token: %w", err)
	}

	return authentication.NewAccessToken(
		tokenID,
		tokenString,
		auth.UserID,
		auth.AccountID,
		expiresIn,
	), nil
}

// ParseAccessToken 解析访问令牌
// 使用 JWKS 公钥验证 RSA 签名
func (g *Generator) ParseAccessToken(tokenValue string) (*authentication.TokenClaims, error) {
	ctx := context.Background() // TODO: 从参数传递 context

	// 解析 token（不验证签名，先提取 kid）
	token, err := jwt.ParseWithClaims(tokenValue, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名方法是 RSA
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// 从 header 获取 kid
		kidInterface, ok := token.Header["kid"]
		if !ok {
			return nil, fmt.Errorf("missing kid in token header")
		}
		kid, ok := kidInterface.(string)
		if !ok {
			return nil, fmt.Errorf("invalid kid type in token header")
		}

		// 从 JWKS 获取密钥
		key, err := g.keyMgmt.GetKeyByKid(ctx, kid)
		if err != nil {
			return nil, fmt.Errorf("failed to get key %s: %w", kid, err)
		}

		// 从 JWK 解析 RSA 公钥用于验签
		// TODO: 实现从 JWK (N, E) 构造 RSA 公钥
		// 目前先返回 nil，后续完善
		_ = key // 避免 unused 警告
		return nil, fmt.Errorf("RSA public key parsing from JWK not implemented yet")
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	// 提取 claims
	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	// 转换为领域模型
	return authentication.NewTokenClaims(
		claims.Id,
		account.NewUserID(claims.UserID),
		account.AccountID(idutil.NewID(claims.AccountID)),
		time.Unix(claims.IssuedAt, 0),
		time.Unix(claims.ExpiresAt, 0),
	), nil
}

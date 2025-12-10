// Package jwt JWT 令牌生成器实现
package jwt

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"math/big"
	"time"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/authentication"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/jwks"
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/token"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
	"github.com/golang-jwt/jwt/v4"
)

// Generator JWT 令牌生成器（使用 JWKS 的 RSA 密钥签名）
type Generator struct {
	issuer          string                  // 颁发者
	keyMgmt         jwks.Manager            // 密钥管理服务
	privKeyResolver jwks.PrivateKeyResolver // 私钥解析器
}

// NewGenerator 创建 JWT 生成器
func NewGenerator(
	issuer string,
	keyMgmt jwks.Manager,
	privKeyResolver jwks.PrivateKeyResolver,
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
func (g *Generator) GenerateAccessToken(ctx context.Context, principal *authentication.Principal, expiresIn time.Duration) (*domain.Token, error) {
	now := time.Now()
	zeroID := meta.FromUint64(0)
	tokenID := zeroID.String() // 生成唯一 Token ID

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
		UserID:    principal.UserID.Uint64(),
		AccountID: principal.AccountID.Uint64(),
		StandardClaims: jwt.StandardClaims{
			Id:        tokenID,
			Subject:   principal.UserID.String(), // 添加 sub 字段，设置为 user_id
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

	return domain.NewAccessToken(
		tokenID,
		tokenString,
		principal.UserID,
		principal.AccountID,
		expiresIn,
	), nil
}

// ParseAccessToken 解析访问令牌
// 使用 JWKS 公钥验证 RSA 签名
func (g *Generator) ParseAccessToken(ctx context.Context, tokenValue string) (*domain.TokenClaims, error) {
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
		if key == nil {
			return nil, fmt.Errorf("key not found for kid %s", kid)
		}

		if key.JWK.Kty != "RSA" {
			return nil, fmt.Errorf("unsupported key kty for verification: %s", key.JWK.Kty)
		}

		if key.JWK.N == nil || key.JWK.E == nil {
			return nil, fmt.Errorf("missing RSA parameters in JWK for kid %s", kid)
		}

		// n and e are base64url encoded (no padding)
		nBytes, err := base64.RawURLEncoding.DecodeString(*key.JWK.N)
		if err != nil {
			return nil, fmt.Errorf("failed to base64url-decode n for kid %s: %w", kid, err)
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(*key.JWK.E)
		if err != nil {
			return nil, fmt.Errorf("failed to base64url-decode e for kid %s: %w", kid, err)
		}

		n := new(big.Int).SetBytes(nBytes)

		// convert exponent bytes to int (big-endian)
		e := 0
		for _, b := range eBytes {
			e = e<<8 + int(b)
		}
		if e == 0 {
			return nil, fmt.Errorf("invalid exponent parsed for kid %s", kid)
		}

		pub := &rsa.PublicKey{N: n, E: e}
		return pub, nil
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
	userID := meta.FromUint64(claims.UserID)
	accountID := meta.FromUint64(claims.AccountID)
	return domain.NewTokenClaims(
		claims.Id,
		userID,
		accountID,
		time.Unix(claims.IssuedAt, 0),
		time.Unix(claims.ExpiresAt, 0),
	), nil
}

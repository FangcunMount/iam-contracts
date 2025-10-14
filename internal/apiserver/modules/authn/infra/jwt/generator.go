// Package jwt JWT 令牌生成器实现
package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/authentication"
	"github.com/fangcun-mount/iam-contracts/pkg/util/idutil"
)

// Generator JWT 令牌生成器
type Generator struct {
	secretKey []byte // JWT 签名密钥
	issuer    string // 颁发者
}

// NewGenerator 创建 JWT 生成器
func NewGenerator(secretKey, issuer string) *Generator {
	return &Generator{
		secretKey: []byte(secretKey),
		issuer:    issuer,
	}
}

// CustomClaims 自定义 JWT Claims
type CustomClaims struct {
	UserID    uint64 `json:"user_id"`
	AccountID uint64 `json:"account_id"`
	jwt.StandardClaims
}

// GenerateAccessToken 生成访问令牌（JWT）
func (g *Generator) GenerateAccessToken(auth *authentication.Authentication, expiresIn time.Duration) (*authentication.Token, error) {
	now := time.Now()
	tokenID := idutil.NewID(0).String() // 生成唯一 Token ID

	claims := CustomClaims{
		UserID:    auth.UserID.Value(),
		AccountID: idutil.ID(auth.AccountID).Value(),
		StandardClaims: jwt.StandardClaims{
			Id:        tokenID,
			Issuer:    g.issuer,
			IssuedAt:  now.Unix(),
			ExpiresAt: now.Add(expiresIn).Unix(),
			NotBefore: now.Unix(),
		},
	}

	// 创建 token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 签名
	tokenString, err := token.SignedString(g.secretKey)
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
func (g *Generator) ParseAccessToken(tokenValue string) (*authentication.TokenClaims, error) {
	// 解析 token
	token, err := jwt.ParseWithClaims(tokenValue, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名方法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return g.secretKey, nil
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

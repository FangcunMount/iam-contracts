// Package auth encrypt and compare password string.
package auth

import (
	"golang.org/x/crypto/bcrypt"
)

// TokenProfileAlgorithm 是 IAM 令牌和 JWKS 密钥支持的唯一 JOSE 算法。
const TokenProfileAlgorithm = "RS256"

// Encrypt 加密明文。
func Encrypt(source string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(source), bcrypt.DefaultCost)
	return string(hashedBytes), err
}

// Compare 比较加密文本与明文是否相同。
func Compare(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// 注意：JWT 签发功能已迁移至 internal/apiserver/infra/token/jwt 包
// 现在统一使用 JWKS (RS256) 非对称签名方案，不再使用 HMAC 对称密钥
// 请使用 jwt.JWSCompactTokenCodec 进行 JWT 签发

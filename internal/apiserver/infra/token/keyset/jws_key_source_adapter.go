package keyset

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"math/big"
	"time"

	jwtinfra "github.com/FangcunMount/iam/v3/internal/apiserver/infra/token/jwt"
	pkgauth "github.com/FangcunMount/iam/v3/pkg/auth"
)

// manager 是密钥管理器。
type manager interface {
	// GetActiveKey 获取活动密钥
	GetActiveKey(ctx context.Context) (*Key, error)
	// GetKeyByKid 根据密钥ID获取密钥
	GetKeyByKid(ctx context.Context, kid string) (*Key, error)
}

// JWSKeySourceAdapter 是 JWS 签名与验签密钥源的适配器。
type JWSKeySourceAdapter struct {
	manager     manager            // 密钥管理器
	keyResolver PrivateKeyResolver // 私钥解析器
}

// NewJWSKeySourceAdapter 创建 JWS 签名与验签密钥源的适配器。
func NewJWSKeySourceAdapter(manager manager, keyResolver PrivateKeyResolver) *JWSKeySourceAdapter {
	return &JWSKeySourceAdapter{
		manager:     manager,
		keyResolver: keyResolver,
	}
}

// ActiveSigningKey 获取活动签名密钥
func (s *JWSKeySourceAdapter) ActiveSigningKey(ctx context.Context) (*jwtinfra.SigningKey, error) {
	// 获取活动密钥
	activeKey, err := s.manager.GetActiveKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get active key: %w", err)
	}
	// 如果活动密钥为空，则返回错误
	if activeKey == nil {
		return nil, fmt.Errorf("active key is nil")
	}
	// 如果活动密钥不可用于签名，则返回错误
	if !activeKey.CanSignAt(time.Now()) {
		return nil, fmt.Errorf("key %s is not eligible for signing", activeKey.Kid)
	}
	// 如果活动密钥算法不支持，则返回错误
	if activeKey.Algorithm != pkgauth.TokenProfileAlgorithm {
		return nil, fmt.Errorf("key %s uses unsupported algorithm %s", activeKey.Kid, activeKey.Algorithm)
	}
	// 解析私钥
	rawKey, err := s.keyResolver.ResolveSigningKey(ctx, activeKey.Kid, activeKey.Algorithm)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve private key: %w", err)
	}
	// 如果私钥类型不匹配，则返回错误
	privateKey, ok := rawKey.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("expected RSA private key, got %T", rawKey)
	}

	// 返回签名密钥
	return &jwtinfra.SigningKey{
		Kid:        activeKey.Kid,
		Algorithm:  activeKey.Algorithm,
		PrivateKey: privateKey,
	}, nil
}

func (s *JWSKeySourceAdapter) VerificationKey(ctx context.Context, kid string) (*jwtinfra.VerificationKey, error) {
	key, err := s.manager.GetKeyByKid(ctx, kid)
	if err != nil {
		return nil, err
	}
	if key == nil {
		return nil, fmt.Errorf("key not found for kid %s", kid)
	}
	if !key.CanVerifyAt(time.Now()) {
		return nil, fmt.Errorf("key %s is not eligible for verification", kid)
	}
	if key.Algorithm != pkgauth.TokenProfileAlgorithm {
		return nil, fmt.Errorf("key %s uses unsupported algorithm %s", kid, key.Algorithm)
	}
	publicKey, err := publicRSAKeyFromJWK(key.JWK)
	if err != nil {
		return nil, err
	}
	return &jwtinfra.VerificationKey{
		Kid:       key.Kid,
		Algorithm: key.Algorithm,
		PublicKey: publicKey,
	}, nil
}

func publicRSAKeyFromJWK(jwk PublicJWK) (*rsa.PublicKey, error) {
	if jwk.Kty != "RSA" {
		return nil, fmt.Errorf("unsupported key kty for verification: %s", jwk.Kty)
	}
	if jwk.N == nil || jwk.E == nil {
		return nil, fmt.Errorf("missing RSA parameters in JWK for kid %s", jwk.Kid)
	}
	nBytes, err := base64.RawURLEncoding.DecodeString(*jwk.N)
	if err != nil {
		return nil, fmt.Errorf("failed to base64url-decode n for kid %s: %w", jwk.Kid, err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(*jwk.E)
	if err != nil {
		return nil, fmt.Errorf("failed to base64url-decode e for kid %s: %w", jwk.Kid, err)
	}
	n := new(big.Int).SetBytes(nBytes)
	e := 0
	for _, b := range eBytes {
		e = e<<8 + int(b)
	}
	if e == 0 {
		return nil, fmt.Errorf("invalid exponent parsed for kid %s", jwk.Kid)
	}
	return &rsa.PublicKey{N: n, E: e}, nil
}

package driven

import (
	"context"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/jwks"
)

// KeyPair 密钥对（私钥 + 公钥）
// 由 KeyGenerator 生成，返回给领域服务
type KeyPair struct {
	// PrivateKey 私钥（crypto.PrivateKey 类型）
	// RSA: *rsa.PrivateKey
	// EC: *ecdsa.PrivateKey
	// OKP: ed25519.PrivateKey
	PrivateKey any

	// PublicJWK 公钥的 JWK 表示
	PublicJWK jwks.PublicJWK
}

// KeyGenerator 密钥生成器接口
// 负责生成密钥对（RSA/EC/OKP）
// 可以有多种实现：RSA 2048/4096、EC P-256/P-384、Ed25519 等
type KeyGenerator interface {
	// GenerateKeyPair 生成密钥对
	// alg: 签名算法（RS256/RS384/RS512/ES256/ES384/ES512/EdDSA）
	// kid: 密钥 ID（由调用方提供，通常是 UUID）
	// 返回：KeyPair（包含私钥和公钥 JWK）
	GenerateKeyPair(ctx context.Context, alg string, kid string) (*KeyPair, error)

	// SupportedAlgorithms 返回支持的算法列表
	// 例如：["RS256", "RS384", "RS512"]
	SupportedAlgorithms() []string
}

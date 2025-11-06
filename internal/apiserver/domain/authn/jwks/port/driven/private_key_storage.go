package driven

import (
	"context"
)

// PrivateKeyStorage 私钥存储接口
// 负责私钥的持久化和删除操作
// 可以有多种实现：PEM 文件、KMS（AWS/GCP/Azure）、HSM 等
type PrivateKeyStorage interface {
	// SavePrivateKey 保存私钥
	// kid: 密钥 ID
	// privateKey: 私钥对象（*rsa.PrivateKey / *ecdsa.PrivateKey / ed25519.PrivateKey）
	// alg: 算法（RS256/ES256/EdDSA 等）
	SavePrivateKey(ctx context.Context, kid string, privateKey any, alg string) error

	// DeletePrivateKey 删除私钥
	// 当密钥被彻底退役时调用
	DeletePrivateKey(ctx context.Context, kid string) error

	// KeyExists 检查私钥是否存在
	KeyExists(ctx context.Context, kid string) (bool, error)
}

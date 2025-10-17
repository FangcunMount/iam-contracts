package driven

import "context"

// PrivateKeyResolver 私钥解析器
// 签名侧拿“私钥句柄”的抽象；开发期可 PEM，生产期 KMS/HSM
type PrivateKeyResolver interface {
	ResolveSigningKey(ctx context.Context, kid, alg string) (any /*priv*/, error)
}

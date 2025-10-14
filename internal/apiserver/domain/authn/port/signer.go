package port

import (
	"context"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/authn/account"
)

// JWTSigner —— 仅负责“签发”Access Token（非对称/含 kid）
type JWTSigner interface {
	// SignAccess：对给定 claims 进行签名并生成 jti/kid
	// 约定最小声明：sub/aid/aud/iss/iat/exp/(sid)
	SignAccess(claims map[string]any) (jwt string, jti string, kid string, err error)
}

// JWTVerifier（可选）—— 用于 /auth/verify 或内省场景
// 资源服务的网关/中间件也可直接基于 JWKS 自行验签；此接口保留给需要在应用内做二次校验的用例。
type JWTVerifier interface {
	// Verify：验签 + 基本声明校验（iat/exp/aud），返回最小声明与 header
	Verify(ctx context.Context, token string, expectedAud string) (claims account.AccessClaims, header map[string]any, err error)
}

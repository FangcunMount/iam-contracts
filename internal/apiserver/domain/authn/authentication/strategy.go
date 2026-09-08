package authentication

import "context"

// IdentityProof 认证凭据接口。
type IdentityProof interface {
	// CredentialKind 返回认证凭据类型
	CredentialKind() CredentialKind
}

// AuthStrategy 身份核验策略（领域服务接口）。
type AuthStrategy interface {
	// Kind 返回身份核验策略类型
	Kind() CredentialKind
	// Authenticate 执行身份核验
	Authenticate(ctx context.Context, proof IdentityProof) (AuthDecision, error)
}

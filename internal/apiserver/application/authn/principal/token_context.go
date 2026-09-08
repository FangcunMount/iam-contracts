package principal

import (
	"github.com/FangcunMount/iam/v4/internal/apiserver/domain/authn/authentication"
	"github.com/FangcunMount/iam/v4/pkg/tenant"
)

// EnsureTokenContext 为签发 JWT 准备授权域 claim；IAM 不写入默认 org_id。
func EnsureTokenContext(p *authentication.Principal) {
	if p == nil {
		return
	}
	if p.TokenContext.TenantDomain == "" {
		p.TokenContext.TenantDomain = tenant.DefaultID
	}
}

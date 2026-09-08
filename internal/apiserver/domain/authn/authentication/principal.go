package authentication

import "github.com/FangcunMount/iam/v4/internal/pkg/meta"

// Principal 是认证成功后的运行时主体表达，是 Login 的领域终点。
type Principal struct {
	UserID          meta.ID
	LoginIdentityID meta.ID

	TenantID meta.ID

	// AuthContext 是认证上下文的权威表达。
	AuthContext AuthenticationContext
}

// ApplyAuthContext 将认证上下文写入 Principal，并同步兼容字段。
func (p *Principal) ApplyAuthContext(ctx AuthenticationContext) {
	if p == nil {
		return
	}
	p.AuthContext = ctx.Clone()
}

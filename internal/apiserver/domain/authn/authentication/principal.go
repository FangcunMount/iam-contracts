package authentication

import "github.com/FangcunMount/iam/v5/internal/pkg/meta"

// Principal 是身份核验成功后确认的运行时主体。
// 它不代表已通过登录准入，也不代表已完成会话建立或令牌颁发。
type Principal struct {
	UserID          meta.ID
	LoginIdentityID meta.ID

	// AuthContext 是认证上下文的权威表达。
	AuthContext AuthenticationContext
}

// ApplyAuthContext 将认证上下文的副本写入 Principal。
func (p *Principal) ApplyAuthContext(ctx AuthenticationContext) {
	if p == nil {
		return
	}
	p.AuthContext = ctx.Clone()
}

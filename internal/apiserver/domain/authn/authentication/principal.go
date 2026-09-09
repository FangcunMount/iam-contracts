package authentication

import "github.com/FangcunMount/iam/v5/internal/pkg/meta"

// Principal 是身份核验成功后确认的运行时主体。
// 它不代表已通过登录准入，也不代表已完成会话建立或令牌颁发。
type Principal struct {
	// ---- 已确认身份 ----
	UserID          meta.ID // 核验成功后确认的 IAM 用户 ID
	LoginIdentityID meta.ID // 本次成功核验的登录身份 ID

	// ---- 本次认证事实 ----
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

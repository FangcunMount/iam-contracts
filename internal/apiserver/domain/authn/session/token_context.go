package session

import "github.com/FangcunMount/iam/v4/internal/pkg/meta"

// TokenContext 表达独立于认证结果、由会话保存并用于令牌签发的业务上下文。
type TokenContext struct {
	TenantDomain string            // 认证域
	OrgID        meta.ID           // 组织ID
	Attributes   map[string]string // 认证属性
}

func (c TokenContext) Clone() TokenContext {
	out := c
	if len(c.Attributes) > 0 {
		out.Attributes = make(map[string]string, len(c.Attributes))
		for key, value := range c.Attributes {
			out.Attributes[key] = value
		}
	}
	return out
}

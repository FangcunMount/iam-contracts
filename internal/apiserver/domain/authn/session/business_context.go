package session

import "github.com/FangcunMount/iam/v5/internal/pkg/meta"

// BusinessContext 保存会话级业务快照，供令牌签发与刷新读取，不是令牌生成结果。
// 当前登录不填充这些字段；保留它们用于延续历史会话的业务声明。
type BusinessContext struct {
	OrgID      meta.ID           // 组织ID
	Attributes map[string]string // 随会话延续的附加属性，签发时仍须遵守 Claims 过滤规则
}

func (c BusinessContext) Clone() BusinessContext {
	out := c
	if len(c.Attributes) > 0 {
		out.Attributes = make(map[string]string, len(c.Attributes))
		for key, value := range c.Attributes {
			out.Attributes[key] = value
		}
	}
	return out
}

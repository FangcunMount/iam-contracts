package session

import "github.com/FangcunMount/iam/v4/internal/pkg/meta"

// CreationContext 是会话建立所需的应用上下文，不属于身份核验结果。
// RequestedTenantID 保留请求选择的租户值，不表示已经核验租户归属。
type CreationContext struct {
	RequestedTenantID meta.ID
	TokenContext      TokenContext
}

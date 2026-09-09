package challenge

import "time"

// AuthChallenge 认证挑战
type AuthChallenge struct {
	// ---- 挑战标识与类型 ----
	ID   string        // 挑战ID
	Type ChallengeType // 挑战类型

	// ---- 使用场景与目标 ----
	Scene  string // 挑战使用场景，用于隔离不同认证流程
	Target string // 挑战目标，如短信手机号或 OAuth 应用 ID

	// ---- 校验材料 ----
	SecretHash []byte // 挑战秘密的校验哈希，不是签名密钥

	// ---- 附加上下文 ----
	Payload map[string]string // 挑战附加上下文，如 OAuth app_id、redirect_uri

	// ---- 生命周期 ----
	ExpiresAt time.Time // 过期时间
	CreatedAt time.Time // 创建时间
}

// IsExpired 验证挑战是否过期
func (c *AuthChallenge) IsExpired(now time.Time) bool {
	return c == nil || !now.Before(c.ExpiresAt)
}

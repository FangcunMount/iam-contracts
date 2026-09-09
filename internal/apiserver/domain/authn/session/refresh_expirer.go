package session

import "time"

// refreshExpirer 刷新过期时间计算器，用于计算下一次 refresh token 的领域过期时间。
type refreshExpirer struct {
	lifetime LifetimePolicy // 会话生命周期策略
}

// newRefreshExpirer 创建刷新过期时间计算器。
func newRefreshExpirer(lifetime LifetimePolicy) *refreshExpirer {
	return &refreshExpirer{lifetime: lifetime}
}

// NewRefreshExpirer 创建刷新过期时间计算器。
func NewRefreshExpirer(lifetime LifetimePolicy) RefreshExpirer {
	return newRefreshExpirer(lifetime)
}

// NextRefreshExpiresAt 计算下一次 refresh token 的领域过期时间
func (r *refreshExpirer) NextRefreshExpiresAt(now time.Time, session *Session) (time.Time, error) {
	return r.lifetime.RefreshTokenExpiresAt(now, session)
}

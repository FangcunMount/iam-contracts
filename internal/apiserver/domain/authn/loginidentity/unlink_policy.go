package loginidentity

import (
	"time"

	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
)

const (
	DefaultRecentAuthWindow = 10 * time.Minute
	DefaultFutureClockSkew  = time.Minute
)

// UnlinkReauthDecision 描述解绑前近期认证检查的结果。
type UnlinkReauthDecision int

const (
	UnlinkReauthOK UnlinkReauthDecision = iota
	UnlinkReauthRequired
	UnlinkReauthInvalidTimestamp
)

// UnlinkReauthRequest 是解绑近期身份核验策略的确定性输入。
type UnlinkReauthRequest struct {
	Identity               *LoginIdentity // 待解绑的登录身份
	CurrentLoginIdentityID meta.ID        // 当前会话使用的登录身份 ID
	AuthenticatedAt        *time.Time     // 最近一次身份核验的时间依据
	Now                    time.Time      // 评估解绑准入的当前时间
}

// UnlinkPolicy 封装解绑安全策略。
type UnlinkPolicy struct {
	RecentAuthWindow time.Duration // 允许使用近期认证的时间窗口
	FutureClockSkew  time.Duration // 允许认证时间领先当前时间的最大偏差
}

// DefaultUnlinkPolicy 返回默认解绑策略。
func DefaultUnlinkPolicy() UnlinkPolicy {
	return UnlinkPolicy{
		RecentAuthWindow: DefaultRecentAuthWindow,
		FutureClockSkew:  DefaultFutureClockSkew,
	}
}

// AssessRecentAuthentication 判断解绑是否满足近期认证要求。
func (p UnlinkPolicy) AssessRecentAuthentication(req UnlinkReauthRequest) UnlinkReauthDecision {
	if req.Identity == nil || !p.requiresRecentAuthentication(req) {
		return UnlinkReauthOK
	}
	if !p.hasRecentAuthentication(req.AuthenticatedAt, req.Now) {
		return UnlinkReauthRequired
	}
	return UnlinkReauthOK
}

func (p UnlinkPolicy) requiresRecentAuthentication(req UnlinkReauthRequest) bool {
	if req.Identity == nil {
		return false
	}
	if !req.CurrentLoginIdentityID.IsZero() && req.Identity.ID == req.CurrentLoginIdentityID {
		return true
	}
	switch req.Identity.Provider {
	case ProviderUsername, ProviderPhone:
		return true
	default:
		return false
	}
}

func (p UnlinkPolicy) hasRecentAuthentication(authenticatedAt *time.Time, now time.Time) bool {
	return (RecentAuthenticationPolicy{Window: p.RecentAuthWindow, FutureClockSkew: p.FutureClockSkew}).Allows(authenticatedAt, now)
}

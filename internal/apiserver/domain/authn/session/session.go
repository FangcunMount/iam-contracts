package session

import (
	"time"

	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authn/authentication"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
)

// Status 表示认证会话的生命周期状态。
type Status string

const (
	// StatusActive 表示会话仍然有效。
	StatusActive Status = "active"
	// StatusRevoked 表示会话已被主动撤销。
	StatusRevoked Status = "revoked"
	// StatusExpired 表示会话已自然过期。
	StatusExpired Status = "expired"
)

// Session 会话，表示一次登录会话。
type Session struct {
	// ---- 会话标识 ----
	SessionID string // 会话ID

	// ---- 主体与登录身份 ----
	UserID          meta.ID // 用户ID
	LoginIdentityID meta.ID // 登录身份ID

	// ---- 原始认证事实 ----
	AuthContext authentication.AuthenticationContext // 原始认证事实，刷新时保留认证方式与认证时间

	// ---- 业务快照 ----
	BusinessContext BusinessContext // 会话业务快照，当前新登录为空

	// ---- 生命周期状态 ----
	Status Status // 会话生命周期状态，实际可用性还需检查 ExpiresAt

	// ---- 生命周期时间 ----
	CreatedAt time.Time // 创建时间
	ExpiresAt time.Time // 过期时间

	// ---- 撤销记录 ----
	RevokedAt    *time.Time // 撤销时间
	RevokeReason string     // 撤销原因
	RevokedBy    string     // 撤销者标识
}

// NewWithContexts 创建包含认证事实与业务快照的会话，并复制可变字段。
func NewWithContexts(sessionID string, userID, loginIdentityID meta.ID, authContext authentication.AuthenticationContext, businessContext BusinessContext, expiresAt time.Time) *Session {
	now := time.Now()
	return &Session{
		SessionID: sessionID, UserID: userID, LoginIdentityID: loginIdentityID,
		AuthContext: authContext.Clone(), BusinessContext: businessContext.Clone(),
		Status: StatusActive, CreatedAt: now, ExpiresAt: expiresAt,
	}
}

// IsActive 返回会话是否仍处于可用状态。
func (s *Session) IsActive() bool {
	if s == nil {
		return false
	}
	if s.IsExpired() {
		return false
	}
	return s.Status == StatusActive
}

// IsExpired 返回会话是否已自然过期。
func (s *Session) IsExpired() bool {
	if s == nil {
		return true
	}
	return time.Now().After(s.ExpiresAt)
}

// RemainingTTL 返回当前会话剩余 TTL。
func (s *Session) RemainingTTL() time.Duration {
	if s == nil || s.IsExpired() {
		return 0
	}
	return time.Until(s.ExpiresAt)
}

// Revoke 将会话置为 revoked。
func (s *Session) Revoke(reason, revokedBy string) {
	if s == nil {
		return
	}
	now := time.Now()
	s.Status = StatusRevoked
	s.RevokedAt = &now
	s.RevokeReason = reason
	s.RevokedBy = revokedBy
}

// Extend 延长会话过期时间，并在必要时把已自然过期的会话重新拉回 active。
func (s *Session) Extend(expiresAt time.Time) {
	if s == nil {
		return
	}
	s.ExpiresAt = expiresAt
	if s.Status == StatusExpired {
		s.Status = StatusActive
	}
}

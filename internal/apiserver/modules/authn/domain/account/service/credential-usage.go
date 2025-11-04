package service

import (
	"time"

	"github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/component-base/pkg/log"
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account/port"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
)

// CredentialUsage 凭据使用记录服务
// 职责：管理凭据的使用状态、成功/失败记录、可用性检查
type CredentialUsage struct{}

// Ensure CredentialUsage implements the port.CredentialUsage interface
var _ port.CredentialUsage = (*CredentialUsage)(nil)

// NewCredentialUsage creates a new CredentialUsage instance
func NewCredentialUsage() *CredentialUsage {
	return &CredentialUsage{}
}

// EnsureUsable 确保凭据可用
// 检查凭据是否处于可用状态（已启用且未锁定）
func (cu *CredentialUsage) EnsureUsable(c *domain.Credential, now time.Time) error {
	if c == nil {
		return errors.WithCode(code.ErrInvalidCredential, "credential cannot be nil")
	}

	// 检查状态是否为启用
	if c.Status != domain.CredStatusEnabled {
		log.Warnw("Credential is not enabled",
			"credentialID", c.ID,
			"status", c.Status.String(),
		)
		return errors.WithCode(code.ErrCredentialDisabled, "credential is disabled")
	}

	// 检查是否被时间锁定
	if c.LockedUntil != nil && now.Before(*c.LockedUntil) {
		log.Warnw("Credential is temporarily locked",
			"credentialID", c.ID,
			"lockedUntil", c.LockedUntil,
		)
		return errors.WithCode(code.ErrCredentialLocked, "credential is locked until %s", c.LockedUntil.Format(time.RFC3339))
	}

	return nil
}

// RecordSuccess 记录认证成功
// 更新最后成功时间，重置失败计数
func (cu *CredentialUsage) RecordSuccess(c *domain.Credential, now time.Time) {
	if c == nil {
		log.Warn("Cannot record success: credential is nil")
		return
	}

	c.LastSuccessAt = &now
	c.FailedAttempts = 0 // 归零失败计数

	log.Infow("Credential authentication succeeded",
		"credentialID", c.ID,
		"accountID", c.AccountID,
		"lastSuccessAt", now,
	)
}

// RecordFailure 记录认证失败
// 增加失败计数，根据锁定策略决定是否锁定凭据
// 返回是否已锁定
func (cu *CredentialUsage) RecordFailure(c *domain.Credential, now time.Time, p domain.LockoutPolicy) (locked bool) {
	if c == nil {
		log.Warn("Cannot record failure: credential is nil")
		return false
	}

	c.LastFailureAt = &now
	c.FailedAttempts++

	log.Infow("Credential authentication failed",
		"credentialID", c.ID,
		"accountID", c.AccountID,
		"failedAttempts", c.FailedAttempts,
		"lastFailureAt", now,
	)

	// 检查是否需要锁定
	if p.Enabled && c.FailedAttempts >= p.Threshold {
		lockUntil := now.Add(p.LockDuration)
		c.LockedUntil = &lockUntil
		// 注意：状态仍然是 Enabled，但通过 LockedUntil 时间来控制锁定

		log.Warnw("Credential locked due to too many failed attempts",
			"credentialID", c.ID,
			"accountID", c.AccountID,
			"failedAttempts", c.FailedAttempts,
			"threshold", p.Threshold,
			"lockedUntil", lockUntil,
		)

		return true
	}

	return false
}

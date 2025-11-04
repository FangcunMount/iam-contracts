package service

import (
	"time"

	"github.com/FangcunMount/component-base/pkg/log"
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account/port"
)

// CredentialLocker 凭据锁定器
// 职责：管理凭据的锁定和解锁（行政动作）
type CredentialLocker struct{}

// Ensure CredentialLocker implements the port.CredentialLocker interface
var _ port.CredentialLocker = (*CredentialLocker)(nil)

// NewCredentialLocker creates a new CredentialLocker instance
func NewCredentialLocker() *CredentialLocker {
	return &CredentialLocker{}
}

// LockUntil 锁定凭据直到指定时间
// 这是行政动作，主要用于 password 类型的凭据
func (cl *CredentialLocker) LockUntil(c *domain.Credential, until time.Time) {
	if c == nil {
		log.Warn("Cannot lock credential: credential is nil")
		return
	}

	c.LockedUntil = &until

	log.Infow("Credential locked administratively",
		"credentialID", c.ID,
		"accountID", c.AccountID,
		"lockedUntil", until,
	)
}

// Unlock 解锁凭据
// 清除锁定时间，重置失败计数
func (cl *CredentialLocker) Unlock(c *domain.Credential) {
	if c == nil {
		log.Warn("Cannot unlock credential: credential is nil")
		return
	}

	c.LockedUntil = nil
	c.FailedAttempts = 0 // 解锁时重置失败计数

	log.Infow("Credential unlocked",
		"credentialID", c.ID,
		"accountID", c.AccountID,
	)
}

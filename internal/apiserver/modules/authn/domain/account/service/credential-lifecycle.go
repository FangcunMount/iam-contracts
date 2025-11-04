package service

import (
	"github.com/FangcunMount/component-base/pkg/log"
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account/port"
)

// CredentialLifecycle 凭据生命周期管理
// 职责：管理凭据的启用和禁用
type CredentialLifecycle struct{}

// Ensure CredentialLifecycle implements the port.CredentialLifecycle interface
var _ port.CredentialLifecycle = (*CredentialLifecycle)(nil)

// NewCredentialLifecycle creates a new CredentialLifecycle instance
func NewCredentialLifecycle() *CredentialLifecycle {
	return &CredentialLifecycle{}
}

// Enable 启用凭据
func (cl *CredentialLifecycle) Enable(c *domain.Credential) {
	if c == nil {
		log.Warn("Cannot enable credential: credential is nil")
		return
	}

	oldStatus := c.Status
	c.Status = domain.CredStatusEnabled

	// 启用时清除锁定状态
	if c.LockedUntil != nil {
		c.LockedUntil = nil
		c.FailedAttempts = 0
	}

	log.Infow("Credential enabled",
		"credentialID", c.ID,
		"accountID", c.AccountID,
		"oldStatus", oldStatus.String(),
		"newStatus", c.Status.String(),
	)
}

// Disable 禁用凭据
func (cl *CredentialLifecycle) Disable(c *domain.Credential) {
	if c == nil {
		log.Warn("Cannot disable credential: credential is nil")
		return
	}

	oldStatus := c.Status
	c.Status = domain.CredStatusDisabled

	log.Infow("Credential disabled",
		"credentialID", c.ID,
		"accountID", c.AccountID,
		"oldStatus", oldStatus.String(),
		"newStatus", c.Status.String(),
	)
}

package service

import (
	"fmt"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/component-base/pkg/log"
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account/port"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
)

// AccountStateMachine 账号状态机
// 状态转换规则：
// - Disabled -> Active, Archived, Deleted
// - Active -> Disabled, Archived, Deleted
// - Archived -> Active, Deleted
// - Deleted -> 不可转换到其他状态（终态）
type AccountStateMachine struct {
	account *domain.Account
	state   IAccountState
}

// AccountStateMachine 实现了 port.AccountStateMachine 接口
var _ port.AccountStateMachine = (*AccountStateMachine)(nil)

// NewAccountStateMachine 创建账号状态机
func NewAccountStateMachine(account *domain.Account) (*AccountStateMachine, error) {
	if account == nil {
		return nil, perrors.WithCode(code.ErrInvalidStateTransition, "account cannot be nil")
	}

	m := &AccountStateMachine{
		account: account,
	}

	// 根据当前状态初始化对应的状态对象
	if err := m.setState(account.Status); err != nil {
		return nil, err
	}

	return m, nil
}

// setState 设置状态机的状态
func (m *AccountStateMachine) setState(status domain.AccountStatus) error {
	switch status {
	case domain.StatusDisabled:
		m.state = &DisabledAccountState{}
	case domain.StatusActive:
		m.state = &ActiveAccountState{}
	case domain.StatusArchived:
		m.state = &ArchivedAccountState{}
	case domain.StatusDeleted:
		m.state = &DeletedAccountState{}
	default:
		return perrors.WithCode(code.ErrInvalidStateTransition, "invalid account status: %d", status)
	}
	return nil
}

// Status 获取当前状态
func (m *AccountStateMachine) Status() domain.AccountStatus {
	return m.account.Status
}

// Account 获取账号对象
func (m *AccountStateMachine) Account() *domain.Account {
	return m.account
}

// Activate 激活账号
func (m *AccountStateMachine) Activate() error {
	return m.state.Activate(m)
}

// Disable 禁用账号
func (m *AccountStateMachine) Disable() error {
	return m.state.Disable(m)
}

// Archive 归档账号
func (m *AccountStateMachine) Archive() error {
	return m.state.Archive(m)
}

// Delete 删除账号
func (m *AccountStateMachine) Delete() error {
	return m.state.Delete(m)
}

// transitionTo 执行状态转换
func (m *AccountStateMachine) transitionTo(newStatus domain.AccountStatus) error {
	oldStatus := m.account.Status

	// 状态未变化，幂等操作
	if oldStatus == newStatus {
		log.Infow("Account status transition (idempotent)",
			"accountID", m.account.ID,
			"from", oldStatus.String(),
			"to", newStatus.String(),
		)
		return nil
	}

	// 更新账号状态
	m.account.Status = newStatus
	if err := m.setState(newStatus); err != nil {
		// 回滚状态
		m.account.Status = oldStatus
		_ = m.setState(oldStatus)
		return err
	}

	// 记录状态转换日志
	log.Infow("Account status transition succeeded",
		"accountID", m.account.ID,
		"from", oldStatus.String(),
		"to", newStatus.String(),
	)

	return nil
}

// invalidTransition 返回非法状态转换错误
func (m *AccountStateMachine) invalidTransition(targetStatus domain.AccountStatus) error {
	log.Warnw("Invalid account status transition",
		"accountID", m.account.ID,
		"from", m.account.Status.String(),
		"to", targetStatus.String(),
	)
	return perrors.WithCode(
		code.ErrInvalidStateTransition,
		"cannot transition from %s to %s",
		m.account.Status.String(),
		targetStatus.String(),
	)
}

// IAccountState 账号状态接口
type IAccountState interface {
	Activate(m *AccountStateMachine) error
	Disable(m *AccountStateMachine) error
	Archive(m *AccountStateMachine) error
	Delete(m *AccountStateMachine) error
}

// DisabledAccountState 已禁用状态
type DisabledAccountState struct{}

// Activate 禁用状态 -> 激活状态 ✓
func (s *DisabledAccountState) Activate(m *AccountStateMachine) error {
	return m.transitionTo(domain.StatusActive)
}

// Disable 禁用状态 -> 禁用状态 ✓ (幂等)
func (s *DisabledAccountState) Disable(m *AccountStateMachine) error {
	return m.transitionTo(domain.StatusDisabled)
}

// Archive 禁用状态 -> 归档状态 ✓
func (s *DisabledAccountState) Archive(m *AccountStateMachine) error {
	return m.transitionTo(domain.StatusArchived)
}

// Delete 禁用状态 -> 删除状态 ✓
func (s *DisabledAccountState) Delete(m *AccountStateMachine) error {
	return m.transitionTo(domain.StatusDeleted)
}

// ActiveAccountState 激活状态
type ActiveAccountState struct{}

// Activate 激活状态 -> 激活状态 ✓ (幂等)
func (s *ActiveAccountState) Activate(m *AccountStateMachine) error {
	return m.transitionTo(domain.StatusActive)
}

// Disable 激活状态 -> 禁用状态 ✓
func (s *ActiveAccountState) Disable(m *AccountStateMachine) error {
	return m.transitionTo(domain.StatusDisabled)
}

// Archive 激活状态 -> 归档状态 ✓
func (s *ActiveAccountState) Archive(m *AccountStateMachine) error {
	return m.transitionTo(domain.StatusArchived)
}

// Delete 激活状态 -> 删除状态 ✓
func (s *ActiveAccountState) Delete(m *AccountStateMachine) error {
	return m.transitionTo(domain.StatusDeleted)
}

// ArchivedAccountState 归档状态
type ArchivedAccountState struct{}

// Activate 归档状态 -> 激活状态 ✓
func (s *ArchivedAccountState) Activate(m *AccountStateMachine) error {
	return m.transitionTo(domain.StatusActive)
}

// Disable 归档状态 -> 禁用状态 ✗ (不允许)
func (s *ArchivedAccountState) Disable(m *AccountStateMachine) error {
	return m.invalidTransition(domain.StatusDisabled)
}

// Archive 归档状态 -> 归档状态 ✓ (幂等)
func (s *ArchivedAccountState) Archive(m *AccountStateMachine) error {
	return m.transitionTo(domain.StatusArchived)
}

// Delete 归档状态 -> 删除状态 ✓
func (s *ArchivedAccountState) Delete(m *AccountStateMachine) error {
	return m.transitionTo(domain.StatusDeleted)
}

// DeletedAccountState 已删除状态（终态）
type DeletedAccountState struct{}

// Activate 删除状态 -> 激活状态 ✗ (不允许，删除是终态)
func (s *DeletedAccountState) Activate(m *AccountStateMachine) error {
	return m.invalidTransition(domain.StatusActive)
}

// Disable 删除状态 -> 禁用状态 ✗ (不允许，删除是终态)
func (s *DeletedAccountState) Disable(m *AccountStateMachine) error {
	return m.invalidTransition(domain.StatusDisabled)
}

// Archive 删除状态 -> 归档状态 ✗ (不允许，删除是终态)
func (s *DeletedAccountState) Archive(m *AccountStateMachine) error {
	return m.invalidTransition(domain.StatusArchived)
}

// Delete 删除状态 -> 删除状态 ✓ (幂等)
func (s *DeletedAccountState) Delete(m *AccountStateMachine) error {
	return m.transitionTo(domain.StatusDeleted)
}

// String 返回状态机的字符串表示（用于调试）
func (m *AccountStateMachine) String() string {
	return fmt.Sprintf("AccountStateMachine{accountID=%s, status=%s}", m.account.ID, m.account.Status.String())
}

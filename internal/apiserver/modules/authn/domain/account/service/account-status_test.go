package service

import (
	"testing"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAccountStateMachine(t *testing.T) {
	validStatuses := []domain.AccountStatus{
		domain.StatusActive,
		domain.StatusDisabled,
		domain.StatusArchived,
		domain.StatusDeleted,
	}

	for _, status := range validStatuses {
		status := status
		t.Run(status.String(), func(t *testing.T) {
			account := domain.NewAccount(
				meta.NewID(1),
				domain.TypeWcMinip,
				"test-external-id",
				domain.WithStatus(status),
			)
			m, err := NewAccountStateMachine(account)
			require.NoError(t, err)
			assert.Equal(t, status, m.Status())
		})
	}

	// 测试 nil account
	_, err := NewAccountStateMachine(nil)
	require.Error(t, err)
	assert.True(t, perrors.IsCode(err, code.ErrInvalidStateTransition))
}

func TestAccountStateMachine_Activate(t *testing.T) {
	tests := []struct {
		name          string
		initialStatus domain.AccountStatus
		wantStatus    domain.AccountStatus
		wantErr       bool
	}{
		{"from-disabled", domain.StatusDisabled, domain.StatusActive, false},
		{"from-archived", domain.StatusArchived, domain.StatusActive, false},
		{"already-active", domain.StatusActive, domain.StatusActive, false},
		{"from-deleted", domain.StatusDeleted, domain.StatusDeleted, true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			m := newMachine(t, tt.initialStatus)
			err := m.Activate()
			if tt.wantErr {
				require.Error(t, err)
				assert.True(t, perrors.IsCode(err, code.ErrInvalidStateTransition))
				assert.Equal(t, tt.initialStatus, m.Status())
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, m.Status())
		})
	}
}

func TestAccountStateMachine_Disable(t *testing.T) {
	tests := []struct {
		name          string
		initialStatus domain.AccountStatus
		wantStatus    domain.AccountStatus
		wantErr       bool
	}{
		{"from-active", domain.StatusActive, domain.StatusDisabled, false},
		{"idempotent-disabled", domain.StatusDisabled, domain.StatusDisabled, false},
		{"from-archived", domain.StatusArchived, domain.StatusArchived, true},
		{"from-deleted", domain.StatusDeleted, domain.StatusDeleted, true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			m := newMachine(t, tt.initialStatus)
			err := m.Disable()
			if tt.wantErr {
				require.Error(t, err)
				assert.True(t, perrors.IsCode(err, code.ErrInvalidStateTransition))
				assert.Equal(t, tt.initialStatus, m.Status())
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, m.Status())
		})
	}
}

func TestAccountStateMachine_Archive(t *testing.T) {
	tests := []struct {
		name          string
		initialStatus domain.AccountStatus
		wantStatus    domain.AccountStatus
		wantErr       bool
	}{
		{"from-active", domain.StatusActive, domain.StatusArchived, false},
		{"from-disabled", domain.StatusDisabled, domain.StatusArchived, false},
		{"idempotent-archived", domain.StatusArchived, domain.StatusArchived, false},
		{"from-deleted", domain.StatusDeleted, domain.StatusDeleted, true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			m := newMachine(t, tt.initialStatus)
			err := m.Archive()
			if tt.wantErr {
				require.Error(t, err)
				assert.True(t, perrors.IsCode(err, code.ErrInvalidStateTransition))
				assert.Equal(t, tt.initialStatus, m.Status())
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, m.Status())
		})
	}
}

func TestAccountStateMachine_Delete(t *testing.T) {
	tests := []struct {
		name          string
		initialStatus domain.AccountStatus
		wantStatus    domain.AccountStatus
	}{
		{"from-active", domain.StatusActive, domain.StatusDeleted},
		{"from-disabled", domain.StatusDisabled, domain.StatusDeleted},
		{"from-archived", domain.StatusArchived, domain.StatusDeleted},
		{"idempotent-deleted", domain.StatusDeleted, domain.StatusDeleted},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			m := newMachine(t, tt.initialStatus)
			err := m.Delete()
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, m.Status())
		})
	}
}

func newMachine(t *testing.T, status domain.AccountStatus) *AccountStateMachine {
	t.Helper()

	account := domain.NewAccount(
		meta.NewID(1),
		domain.TypeWcMinip,
		"test-external-id",
		domain.WithStatus(status),
	)
	m, err := NewAccountStateMachine(account)
	require.NoError(t, err)
	return m
}

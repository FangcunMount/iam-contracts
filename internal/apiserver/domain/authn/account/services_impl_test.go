package account

import (
	"context"
	"testing"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreatorCreateSuccess(t *testing.T) {
	repo := &fakeAccountRepo{
		getByExternalFunc: func(context.Context, ExternalID, AppId) (*Account, error) {
			return nil, nil
		},
	}
	c := NewCreator(repo)

	dto := CreateAccountDTO{
		UserID:      meta.FromUint64(101),
		AccountType: TypeWcMinip,
		ExternalID:  ExternalID("external-123"),
		AppID:       AppId("appid-123"),
	}

	account, err := c.Create(context.Background(), dto)
	require.NoError(t, err)
	require.NotNil(t, account)
	assert.Equal(t, dto.UserID, account.UserID)
	assert.Equal(t, dto.AccountType, account.Type)
	assert.Equal(t, dto.ExternalID, account.ExternalID)
	assert.Equal(t, dto.AppID, account.AppID)
	assert.True(t, account.IsActive())
}

func TestCreatorCreateValidation(t *testing.T) {
	repo := &fakeAccountRepo{}
	c := NewCreator(repo)
	ctx := context.Background()

	valid := CreateAccountDTO{
		UserID:      meta.FromUint64(1),
		AccountType: TypeWcMinip,
		ExternalID:  ExternalID("ext"),
		AppID:       AppId("app"),
	}

	tests := []struct {
		name string
		dto  CreateAccountDTO
	}{
		{
			name: "invalid user",
			dto: CreateAccountDTO{
				UserID:      meta.ID(0),
				AccountType: valid.AccountType,
				ExternalID:  valid.ExternalID,
				AppID:       valid.AppID,
			},
		},
		{
			name: "invalid type",
			dto: CreateAccountDTO{
				UserID:      valid.UserID,
				AccountType: AccountType("unknown"),
				ExternalID:  valid.ExternalID,
				AppID:       valid.AppID,
			},
		},
		{
			name: "missing app",
			dto: CreateAccountDTO{
				UserID:      valid.UserID,
				AccountType: valid.AccountType,
				ExternalID:  valid.ExternalID,
				AppID:       "",
			},
		},
		{
			name: "missing external",
			dto: CreateAccountDTO{
				UserID:      valid.UserID,
				AccountType: valid.AccountType,
				ExternalID:  "",
				AppID:       valid.AppID,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := c.Create(ctx, tc.dto)
			requireErrorCode(t, err, code.ErrInvalidArgument)
		})
	}
}

func TestCreatorCreateDuplicateExternal(t *testing.T) {
	ctx := context.Background()
	dto := CreateAccountDTO{
		UserID:      meta.FromUint64(88),
		AccountType: TypeWcMinip,
		ExternalID:  ExternalID("dup"),
		AppID:       AppId("app"),
	}

	tests := []struct {
		name          string
		existingUser  meta.ID
		expectedError int
	}{
		{
			name:          "same user repeated",
			existingUser:  dto.UserID,
			expectedError: code.ErrExternalExists,
		},
		{
			name:          "belongs to another user",
			existingUser:  meta.FromUint64(999),
			expectedError: code.ErrAccountExists,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeAccountRepo{
				getByExternalFunc: func(context.Context, ExternalID, AppId) (*Account, error) {
					return &Account{UserID: tc.existingUser}, nil
				},
			}
			c := NewCreator(repo)

			_, err := c.Create(ctx, dto)
			requireErrorCode(t, err, tc.expectedError)
		})
	}
}

func TestEditorSetUniqueIDValidation(t *testing.T) {
	repo := &fakeAccountRepo{}
	editor := NewEditor(repo)

	_, err := editor.SetUniqueID(context.Background(), meta.FromUint64(1), "")
	requireErrorCode(t, err, code.ErrInvalidUniqueID)
}

func TestEditorSetUniqueIDNotFound(t *testing.T) {
	repo := &fakeAccountRepo{
		getByIDFunc: func(context.Context, meta.ID) (*Account, error) {
			return nil, nil
		},
	}
	editor := NewEditor(repo)

	_, err := editor.SetUniqueID(context.Background(), meta.FromUint64(2), UnionID("u1"))
	requireErrorCode(t, err, code.ErrNotFoundAccount)
}

func TestEditorSetUniqueIDAlreadySet(t *testing.T) {
	acc := NewAccount(meta.FromUint64(3), TypeWcMinip, ExternalID("ext"))
	acc.UniqueID = UnionID("exists")
	repo := &fakeAccountRepo{
		getByIDFunc: func(context.Context, meta.ID) (*Account, error) {
			return acc, nil
		},
	}
	editor := NewEditor(repo)

	_, err := editor.SetUniqueID(context.Background(), acc.ID, UnionID("new"))
	requireErrorCode(t, err, code.ErrUniqueIDExists)
}

func TestEditorSetUniqueIDSuccess(t *testing.T) {
	acc := NewAccount(meta.FromUint64(4), TypeWcMinip, ExternalID("ext"))
	repo := &fakeAccountRepo{
		getByIDFunc: func(context.Context, meta.ID) (*Account, error) {
			return acc, nil
		},
	}
	editor := NewEditor(repo)

	updated, err := editor.SetUniqueID(context.Background(), acc.ID, UnionID("fresh"))
	require.NoError(t, err)
	require.Equal(t, UnionID("fresh"), updated.UniqueID)
}

func TestEditorUpdateProfileGuards(t *testing.T) {
	deleted := NewAccount(meta.FromUint64(5), TypeWcMinip, ExternalID("ext"))
	deleted.Status = StatusDeleted
	repo := &fakeAccountRepo{
		getByIDFunc: func(context.Context, meta.ID) (*Account, error) {
			return deleted, nil
		},
	}
	editor := NewEditor(repo)

	_, err := editor.UpdateProfile(context.Background(), deleted.ID, map[string]string{"nick": "foo"})
	requireErrorCode(t, err, code.ErrInvalidArgument)
}

func TestEditorUpdateProfileAndMeta(t *testing.T) {
	acc := NewAccount(meta.FromUint64(6), TypeWcMinip, ExternalID("ext"))
	repo := &fakeAccountRepo{
		getByIDFunc: func(context.Context, meta.ID) (*Account, error) {
			return acc, nil
		},
	}
	editor := NewEditor(repo)

	profile := map[string]string{"nick": "foo"}
	metaData := map[string]string{"k1": "v1"}

	updatedProfile, err := editor.UpdateProfile(context.Background(), acc.ID, profile)
	require.NoError(t, err)
	assert.Equal(t, "foo", updatedProfile.Profile["nick"])

	updatedMeta, err := editor.UpdateMeta(context.Background(), acc.ID, metaData)
	require.NoError(t, err)
	assert.Equal(t, "v1", updatedMeta.Meta["k1"])
}

func TestStatusManagerActivate(t *testing.T) {
	acc := &Account{
		ID:     meta.FromUint64(10),
		Status: StatusDisabled,
	}
	repo := &fakeAccountRepo{
		getByIDFunc: func(context.Context, meta.ID) (*Account, error) {
			return acc, nil
		},
	}
	sm := NewStatusManager(repo)

	updated, err := sm.Activate(context.Background(), acc.ID)
	require.NoError(t, err)
	require.Equal(t, StatusActive, updated.Status)
}

func TestStatusManagerDisableFromArchived(t *testing.T) {
	acc := &Account{
		ID:     meta.FromUint64(11),
		Status: StatusArchived,
	}
	repo := &fakeAccountRepo{
		getByIDFunc: func(context.Context, meta.ID) (*Account, error) {
			return acc, nil
		},
	}
	sm := NewStatusManager(repo)

	_, err := sm.Disable(context.Background(), acc.ID)
	requireErrorCode(t, err, code.ErrInvalidStateTransition)
}

func TestAccountStateMachineGuards(t *testing.T) {
	_, err := newAccountStateMachine(nil)
	requireErrorCode(t, err, code.ErrInvalidStateTransition)

	acc := &Account{Status: AccountStatus(99)}
	_, err = newAccountStateMachine(acc)
	requireErrorCode(t, err, code.ErrInvalidStateTransition)
}

func TestAccountStateMachineIdempotentTransition(t *testing.T) {
	acc := &Account{Status: StatusActive}
	machine, err := newAccountStateMachine(acc)
	require.NoError(t, err)

	err = machine.activate()
	require.NoError(t, err)
	assert.Equal(t, StatusActive, acc.Status)
}

func requireErrorCode(t *testing.T, err error, expected int) {
	t.Helper()
	require.Error(t, err)
	coder := perrors.ParseCoder(err)
	require.NotNil(t, coder)
	require.Equal(t, expected, coder.Code())
}

type fakeAccountRepo struct {
	getByIDFunc       func(context.Context, meta.ID) (*Account, error)
	getByExternalFunc func(context.Context, ExternalID, AppId) (*Account, error)
}

func (f *fakeAccountRepo) Create(context.Context, *Account) error {
	return nil
}

func (f *fakeAccountRepo) UpdateUniqueID(context.Context, meta.ID, UnionID) error {
	return nil
}

func (f *fakeAccountRepo) UpdateStatus(context.Context, meta.ID, AccountStatus) error {
	return nil
}

func (f *fakeAccountRepo) UpdateProfile(context.Context, meta.ID, map[string]string) error {
	return nil
}

func (f *fakeAccountRepo) UpdateMeta(context.Context, meta.ID, map[string]string) error {
	return nil
}

func (f *fakeAccountRepo) GetByID(ctx context.Context, id meta.ID) (*Account, error) {
	if f.getByIDFunc == nil {
		return nil, nil
	}
	return f.getByIDFunc(ctx, id)
}

func (f *fakeAccountRepo) GetByUniqueID(context.Context, UnionID) (*Account, error) {
	return nil, nil
}

func (f *fakeAccountRepo) GetByExternalIDAppId(ctx context.Context, externalID ExternalID, appID AppId) (*Account, error) {
	if f.getByExternalFunc == nil {
		return nil, nil
	}
	return f.getByExternalFunc(ctx, externalID, appID)
}

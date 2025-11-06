package user

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

type stubUserValidator struct {
	renameErr         error
	updateContactErr  error
	checkPhoneErr     error
	renameCalls       int
	updateContactCalls int
	checkCalls        int
}

func (s *stubUserValidator) ValidateRegister(context.Context, string, meta.Phone) error {
	return nil
}
func (s *stubUserValidator) ValidateRename(string) error {
	s.renameCalls++
	return s.renameErr
}
func (s *stubUserValidator) ValidateUpdateContact(context.Context, *User, meta.Phone, meta.Email) error {
	s.updateContactCalls++
	return s.updateContactErr
}
func (s *stubUserValidator) CheckPhoneUnique(context.Context, meta.Phone) error {
	s.checkCalls++
	return s.checkPhoneErr
}

func TestProfileEditor_RenameSuccess(t *testing.T) {
	repo := newStubUserRepository()
	userEntity, err := NewUser("old", meta.NewPhone("10086"))
	require.NoError(t, err)
	userEntity.ID = meta.NewID(1)
	repo.usersByID[userEntity.ID.ToUint64()] = userEntity

	validator := &stubUserValidator{}
	editor := NewProfileEditor(repo, validator)

	updated, err := editor.Rename(context.Background(), userEntity.ID, "new-name")

	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, "new-name", updated.Name)
	assert.Equal(t, 1, validator.renameCalls)
	assert.Equal(t, 1, repo.findIDCalls)
}

func TestProfileEditor_RenameValidationError(t *testing.T) {
	repo := newStubUserRepository()
	userEntity, _ := NewUser("old", meta.NewPhone("10086"))
	userEntity.ID = meta.NewID(2)
	repo.usersByID[userEntity.ID.ToUint64()] = userEntity

	validator := &stubUserValidator{renameErr: errors.New("bad name")}
	editor := NewProfileEditor(repo, validator)

	updated, err := editor.Rename(context.Background(), userEntity.ID, "")

	require.Error(t, err)
	assert.Nil(t, updated)
	assert.Equal(t, 1, validator.renameCalls)
	assert.Equal(t, 0, repo.findIDCalls, "repository should not be touched when validation fails")
}

func TestProfileEditor_UpdateContact(t *testing.T) {
	repo := newStubUserRepository()
	userEntity, _ := NewUser("user", meta.NewPhone("10086"))
	userEntity.ID = meta.NewID(3)
	repo.usersByID[userEntity.ID.ToUint64()] = userEntity

	validator := &stubUserValidator{}
	editor := NewProfileEditor(repo, validator)

	newPhone := meta.NewPhone("10010")
	newEmail := meta.NewEmail("user@example.com")

	updated, err := editor.UpdateContact(context.Background(), userEntity.ID, newPhone, newEmail)

	require.NoError(t, err)
	assert.Equal(t, newPhone, updated.Phone)
	assert.Equal(t, newEmail, updated.Email)
	assert.Equal(t, 1, validator.updateContactCalls)
	assert.Equal(t, 1, repo.findIDCalls)
}

func TestProfileEditor_UpdateContactValidationError(t *testing.T) {
	repo := newStubUserRepository()
	userEntity, _ := NewUser("user", meta.NewPhone("10086"))
	userEntity.ID = meta.NewID(4)
	repo.usersByID[userEntity.ID.ToUint64()] = userEntity

	validator := &stubUserValidator{updateContactErr: errors.New("duplicate")}
	editor := NewProfileEditor(repo, validator)

	updated, err := editor.UpdateContact(context.Background(), userEntity.ID, meta.NewPhone("10010"), meta.NewEmail("user@example.com"))

	require.Error(t, err)
	assert.Nil(t, updated)
	assert.Equal(t, 1, validator.updateContactCalls)
	// repository should still be consulted to load the user
	assert.Equal(t, 1, repo.findIDCalls)
}

func TestProfileEditor_UpdateIDCard(t *testing.T) {
	repo := newStubUserRepository()
	userEntity, _ := NewUser("user", meta.NewPhone("10086"))
	userEntity.ID = meta.NewID(5)
	repo.usersByID[userEntity.ID.ToUint64()] = userEntity

	editor := NewProfileEditor(repo, &stubUserValidator{})
	idCard := meta.NewIDCard("tester", "654321")

	updated, err := editor.UpdateIDCard(context.Background(), userEntity.ID, idCard)

	require.NoError(t, err)
	assert.True(t, updated.IDCard.Equal(idCard))
	assert.Equal(t, 1, repo.findIDCalls)
}

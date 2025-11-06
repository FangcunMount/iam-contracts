package user

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

func TestValidator_ValidateRegisterSuccess(t *testing.T) {
	repo := newStubUserRepository()
	v := NewValidator(repo)

	err := v.ValidateRegister(context.Background(), " name ", meta.NewPhone("10086"))

	require.NoError(t, err)
	assert.Equal(t, 1, repo.findPhoneCalls)
}

func TestValidator_ValidateRegister_EmptyInputs(t *testing.T) {
	repo := newStubUserRepository()
	v := NewValidator(repo)

	err := v.ValidateRegister(context.Background(), " ", meta.NewPhone("123"))
	require.Error(t, err)
	assert.Contains(t, fmt.Sprintf("%-v", err), "name cannot be empty")

	err = v.ValidateRegister(context.Background(), "name", meta.Phone{})
	require.Error(t, err)
	assert.Contains(t, fmt.Sprintf("%-v", err), "phone cannot be empty")
}

func TestValidator_ValidateRegister_DuplicatePhone(t *testing.T) {
	repo := newStubUserRepository()
	repo.usersByPhone["10086"] = &User{}
	v := NewValidator(repo)

	err := v.ValidateRegister(context.Background(), "name", meta.NewPhone("10086"))

	require.Error(t, err)
	assert.Contains(t, fmt.Sprintf("%-v", err), "already exists")
	assert.Equal(t, 1, repo.findPhoneCalls)
}

func TestValidator_CheckPhoneUnique_ErrorPropagation(t *testing.T) {
	repo := newStubUserRepository()
	repo.phoneErr = errors.New("db failure")
	v := NewValidator(repo)

	err := v.CheckPhoneUnique(context.Background(), meta.NewPhone("10086"))

	require.Error(t, err)
	assert.Contains(t, fmt.Sprintf("%-v", err), "check user phone")
	assert.Equal(t, 1, repo.findPhoneCalls)
}

func TestValidator_CheckPhoneUnique_EmptyPhone(t *testing.T) {
	repo := newStubUserRepository()
	v := NewValidator(repo)

	err := v.CheckPhoneUnique(context.Background(), meta.Phone{})
	require.Error(t, err)
	assert.Contains(t, fmt.Sprintf("%-v", err), "phone cannot be empty")
}

func TestValidator_ValidateRename(t *testing.T) {
	v := NewValidator(newStubUserRepository())

	assert.NoError(t, v.ValidateRename(" valid "))

	err := v.ValidateRename(" ")
	require.Error(t, err)
	assert.Contains(t, fmt.Sprintf("%-v", err), "name cannot be empty")
}

func TestValidator_ValidateUpdateContact(t *testing.T) {
	repo := newStubUserRepository()
	userEntity, _ := NewUser("user", meta.NewPhone("10086"))
	v := NewValidator(repo)

	// same phone should skip uniqueness check
	err := v.ValidateUpdateContact(context.Background(), userEntity, meta.NewPhone("10086"), meta.NewEmail("a@b.com"))
	require.NoError(t, err)
	assert.Equal(t, 0, repo.findPhoneCalls)

	// changed phone and repository says available
	err = v.ValidateUpdateContact(context.Background(), userEntity, meta.NewPhone("10010"), meta.NewEmail("a@b.com"))
	require.NoError(t, err)
	assert.Equal(t, 1, repo.findPhoneCalls)

	// changed phone but duplicate exists
	repo.usersByPhone["10011"] = &User{}
	err = v.ValidateUpdateContact(context.Background(), userEntity, meta.NewPhone("10011"), meta.NewEmail("a@b.com"))
	require.Error(t, err)
	assert.Contains(t, fmt.Sprintf("%-v", err), "already exists")
	assert.Equal(t, 2, repo.findPhoneCalls)
}

func TestValidator_CheckPhoneUnique_NotFound(t *testing.T) {
	repo := newStubUserRepository()
	v := NewValidator(repo)

	err := v.CheckPhoneUnique(context.Background(), meta.NewPhone("10086"))

	require.NoError(t, err)
	assert.Equal(t, 1, repo.findPhoneCalls)
}

func TestValidator_CheckPhoneUnique_Found(t *testing.T) {
	repo := newStubUserRepository()
	repo.usersByPhone["10086"] = &User{}
	v := NewValidator(repo)

	err := v.CheckPhoneUnique(context.Background(), meta.NewPhone("10086"))

	require.Error(t, err)
	assert.Equal(t, 1, repo.findPhoneCalls)
	assert.Contains(t, fmt.Sprintf("%-v", err), "already exists")
}

func TestValidator_CheckPhoneUnique_RepoReturnsNotFound(t *testing.T) {
	repo := newStubUserRepository()
	// ensure stub returns ErrRecordNotFound
	delete(repo.usersByPhone, "10087")
	v := NewValidator(repo)

	err := v.CheckPhoneUnique(context.Background(), meta.NewPhone("10087"))

	require.NoError(t, err)
	assert.Equal(t, 1, repo.findPhoneCalls)
}

func TestValidator_CheckPhoneUnique_RepoReturnsUnknown(t *testing.T) {
	repo := newStubUserRepository()
	repo.phoneErr = gorm.ErrInvalidDB
	v := NewValidator(repo)

	err := v.CheckPhoneUnique(context.Background(), meta.NewPhone("10088"))

	require.Error(t, err)
	assert.Contains(t, fmt.Sprintf("%-v", err), "failed")
	assert.Equal(t, 1, repo.findPhoneCalls)
}

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

	phone, err := meta.NewPhone("+8613012345678")
	require.NoError(t, err)

	err = v.ValidateRegister(context.Background(), " name ", phone)

	require.NoError(t, err)
	assert.Equal(t, 1, repo.findPhoneCalls)
}

func TestValidator_ValidateRegister_EmptyInputs(t *testing.T) {
	repo := newStubUserRepository()
	v := NewValidator(repo)

	phone, err := meta.NewPhone("+8613012345678")
	require.NoError(t, err)

	err = v.ValidateRegister(context.Background(), " ", phone)
	require.Error(t, err)
	assert.Contains(t, fmt.Sprintf("%-v", err), "name cannot be empty")

	err = v.ValidateRegister(context.Background(), "name", meta.Phone{})
	require.Error(t, err)
	assert.Contains(t, fmt.Sprintf("%-v", err), "phone cannot be empty")
}

func TestValidator_ValidateRegister_DuplicatePhone(t *testing.T) {
	repo := newStubUserRepository()
	phone, err := meta.NewPhone("+8613012345678")
	require.NoError(t, err)
	repo.usersByPhone[phone.String()] = &User{}
	v := NewValidator(repo)

	err = v.ValidateRegister(context.Background(), "name", phone)

	require.Error(t, err)
	assert.Contains(t, fmt.Sprintf("%-v", err), "already exists")
	assert.Equal(t, 1, repo.findPhoneCalls)
}

func TestValidator_CheckPhoneUnique_ErrorPropagation(t *testing.T) {
	repo := newStubUserRepository()
	repo.phoneErr = errors.New("db failure")
	v := NewValidator(repo)

	phone, err := meta.NewPhone("+8613012345678")
	require.NoError(t, err)

	err = v.CheckPhoneUnique(context.Background(), phone)

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
	phone, err := meta.NewPhone("+8613012345678")
	require.NoError(t, err)
	userEntity, _ := NewUser("user", phone)
	v := NewValidator(repo)

	email, err := meta.NewEmail("a@b.com")
	require.NoError(t, err)

	// same phone should skip uniqueness check
	err = v.ValidateUpdateContact(context.Background(), userEntity, phone, email)
	require.NoError(t, err)
	assert.Equal(t, 0, repo.findPhoneCalls)

	// changed phone and repository says available
	newPhone1, err := meta.NewPhone("+8613112345678")
	require.NoError(t, err)
	err = v.ValidateUpdateContact(context.Background(), userEntity, newPhone1, email)
	require.NoError(t, err)
	assert.Equal(t, 1, repo.findPhoneCalls)

	// changed phone but duplicate exists
	newPhone2, err := meta.NewPhone("+8613212345678")
	require.NoError(t, err)
	repo.usersByPhone[newPhone2.String()] = &User{}
	err = v.ValidateUpdateContact(context.Background(), userEntity, newPhone2, email)
	require.Error(t, err)
	assert.Contains(t, fmt.Sprintf("%-v", err), "already exists")
	assert.Equal(t, 2, repo.findPhoneCalls)
}

func TestValidator_CheckPhoneUnique_NotFound(t *testing.T) {
	repo := newStubUserRepository()
	v := NewValidator(repo)

	phone, err := meta.NewPhone("+8613012345678")
	require.NoError(t, err)

	err = v.CheckPhoneUnique(context.Background(), phone)

	require.NoError(t, err)
	assert.Equal(t, 1, repo.findPhoneCalls)
}

func TestValidator_CheckPhoneUnique_Found(t *testing.T) {
	repo := newStubUserRepository()
	phone, err := meta.NewPhone("+8613012345678")
	require.NoError(t, err)
	repo.usersByPhone[phone.String()] = &User{}
	v := NewValidator(repo)

	err = v.CheckPhoneUnique(context.Background(), phone)

	require.Error(t, err)
	assert.Equal(t, 1, repo.findPhoneCalls)
	assert.Contains(t, fmt.Sprintf("%-v", err), "already exists")
}

func TestValidator_CheckPhoneUnique_RepoReturnsNotFound(t *testing.T) {
	repo := newStubUserRepository()
	phone, err := meta.NewPhone("+8613312345678")
	require.NoError(t, err)
	// ensure stub returns ErrRecordNotFound
	delete(repo.usersByPhone, phone.String())
	v := NewValidator(repo)

	err = v.CheckPhoneUnique(context.Background(), phone)

	require.NoError(t, err)
	assert.Equal(t, 1, repo.findPhoneCalls)
}

func TestValidator_CheckPhoneUnique_RepoReturnsUnknown(t *testing.T) {
	repo := newStubUserRepository()
	repo.phoneErr = gorm.ErrInvalidDB
	v := NewValidator(repo)

	phone, err := meta.NewPhone("+8613412345678")
	require.NoError(t, err)

	err = v.CheckPhoneUnique(context.Background(), phone)

	require.Error(t, err)
	assert.Contains(t, fmt.Sprintf("%-v", err), "failed")
	assert.Equal(t, 1, repo.findPhoneCalls)
}

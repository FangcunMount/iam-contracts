package child

import (
	"fmt"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

func TestChildValidator_ValidateRegister(t *testing.T) {
	v := NewValidator(&stubChildRepository{})
	err := v.ValidateRegister(context.Background(), "name", meta.GenderMale, meta.NewBirthday("2010-01-01"))
	assert.NoError(t, err)
}

func TestChildValidator_ValidateRegister_EmptyName(t *testing.T) {
	v := NewValidator(&stubChildRepository{})
	err := v.ValidateRegister(context.Background(), "", meta.GenderMale, meta.NewBirthday("2010-01-01"))
	require.Error(t, err)
	assert.Contains(t, fmt.Sprintf("%-v", err), "name cannot be empty")
}

func TestChildValidator_ValidateRegister_EmptyBirthday(t *testing.T) {
	v := NewValidator(&stubChildRepository{})
	err := v.ValidateRegister(context.Background(), "name", meta.GenderMale, meta.Birthday{})
	require.Error(t, err)
	assert.Contains(t, fmt.Sprintf("%-v", err), "birthday cannot be empty")
}

func TestChildValidator_ValidateRename(t *testing.T) {
	v := NewValidator(&stubChildRepository{})
	assert.NoError(t, v.ValidateRename("valid"))

	err := v.ValidateRename("")
	require.Error(t, err)
	assert.Contains(t, fmt.Sprintf("%-v", err), "name cannot be empty")
}

func TestChildValidator_ValidateUpdateProfile(t *testing.T) {
	v := NewValidator(&stubChildRepository{})
	assert.NoError(t, v.ValidateUpdateProfile(meta.GenderMale, meta.NewBirthday("2010-01-01")))

	err := v.ValidateUpdateProfile(meta.GenderMale, meta.Birthday{})
	require.Error(t, err)
	assert.Contains(t, fmt.Sprintf("%-v", err), "birthday cannot be empty")
}

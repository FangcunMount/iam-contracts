package user

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

func TestLifecycler_StatusTransitions(t *testing.T) {
	repo := newStubUserRepository()
	userEntity, _ := NewUser("user", meta.NewPhone("10086"))
	userEntity.ID = meta.NewID(10)
	repo.usersByID[userEntity.ID.ToUint64()] = userEntity

	lifecycle := NewLifecycler(repo)

	activated, err := lifecycle.Activate(context.Background(), userEntity.ID)
	require.NoError(t, err)
	assert.True(t, activated.IsUsable())

	deactivated, err := lifecycle.Deactivate(context.Background(), userEntity.ID)
	require.NoError(t, err)
	assert.True(t, deactivated.IsInactive())

	blocked, err := lifecycle.Block(context.Background(), userEntity.ID)
	require.NoError(t, err)
	assert.True(t, blocked.IsBlocked())
}

func TestLifecycler_RepoError(t *testing.T) {
	repo := newStubUserRepository()
	repo.findErr = errors.New("db error")
	lifecycle := NewLifecycler(repo)

	activated, err := lifecycle.Activate(context.Background(), meta.NewID(1))
	require.Error(t, err)
	assert.Nil(t, activated)
}

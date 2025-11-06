package guardianship

import (
	"context"
	"testing"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/uc/guardianship"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRepository_Create_DuplicateReturnsBusinessError(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	// Ensure table exists with unique index defined in PO tags
	require.NoError(t, db.AutoMigrate(&GuardianshipPO{}))

	repo := NewRepository(db)
	ctx := context.Background()

	g1 := &guardianship.Guardianship{
		User:  meta.NewID(1),
		Child: meta.NewID(2),
		Rel:   guardianship.RelParent,
	}

	// first create should succeed
	err = repo.Create(ctx, g1)
	require.NoError(t, err)

	// second create with same user+child should be treated as business 'exists' error
	g2 := &guardianship.Guardianship{
		User:  meta.NewID(1),
		Child: meta.NewID(2),
		Rel:   guardianship.RelParent,
	}
	err = repo.Create(ctx, g2)
	require.Error(t, err)

	// We expect the error to be wrapped with the registered business code
	require.True(t, perrors.IsCode(err, code.ErrIdentityGuardianshipExists), "error must be mapped to ErrIdentityGuardianshipExists")
}

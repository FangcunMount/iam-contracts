package guardianship

import (
	"context"
	"testing"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/uc/guardianship"
	testhelpers "github.com/FangcunMount/iam-contracts/internal/apiserver/testhelpers"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
	"github.com/stretchr/testify/require"
)

func TestRepository_Create_DuplicateReturnsBusinessError(t *testing.T) {
	db := testhelpers.SetupTempSQLiteDB(t)
	// Ensure table exists with unique index defined in PO tags
	require.NoError(t, db.AutoMigrate(&GuardianshipPO{}))

	repo := NewRepository(db)
	ctx := context.Background()

	userID1 := meta.FromUint64(1)
	childID2 := meta.FromUint64(2)
	g1 := &guardianship.Guardianship{
		User:  userID1,
		Child: childID2,
		Rel:   guardianship.RelParent,
	}

	// first create should succeed
	err := repo.Create(ctx, g1)
	require.NoError(t, err)

	// second create with same user+child should be treated as business 'exists' error
	userID1_2 := meta.FromUint64(1)
	childID2_2 := meta.FromUint64(2)
	g2 := &guardianship.Guardianship{
		User:  userID1_2,
		Child: childID2_2,
		Rel:   guardianship.RelParent,
	}
	err = repo.Create(ctx, g2)
	require.Error(t, err)

	// We expect the error to be wrapped with the registered business code
	require.True(t, perrors.IsCode(err, code.ErrIdentityGuardianshipExists), "error must be mapped to ErrIdentityGuardianshipExists")
}

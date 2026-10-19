//go:build integration

package accountprovision

import (
	"github.com/FangcunMount/iam/v5/internal/apiserver/infra/crypto"
	fixture "github.com/FangcunMount/iam/v5/internal/apiserver/maintenance/policywrite/testutil"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestM3TransactionMaintenanceReviewerScope(t *testing.T) {
	db, input := reviewerFixture(t)
	writer, sender := fixture.NativeWriter(t, db)
	ctx := t.Context()
	hasher := crypto.NewArgon2Hasher("")
	before, err := ReviewerScopePreflight(ctx, db, input, hasher, 1)
	require.NoError(t, err)
	after, err := ReviewerScopeApply(ctx, db, input, hasher, 1, before.Fingerprint, writer)
	require.NoError(t, err)
	require.Equal(t, "scope_matches_reference", after.State)
	current, err := ReviewerScopePreflight(ctx, db, input, hasher, 1)
	require.NoError(t, err)
	_, err = ReviewerScopeApply(ctx, db, input, hasher, 1, current.Fingerprint, writer)
	require.NoError(t, err)
	require.Equal(t, 1, sender.Calls)
	fixture.AssertCommittedWitnesses(t, db, 1)
	var outbox int64
	require.NoError(t, db.Table("test_scope_outbox").Count(&outbox).Error)
	require.Zero(t, outbox)
}

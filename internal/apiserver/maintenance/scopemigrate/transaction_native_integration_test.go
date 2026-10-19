//go:build integration

package scopemigrate

import (
	nativefixture "github.com/FangcunMount/iam/v5/internal/apiserver/maintenance/policywrite/testutil"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestM3TransactionMaintenanceScope(t *testing.T) {
	db, qs, input := applyFixture(t)
	writer, sender := nativefixture.NativeWriter(t, db)
	ctx := t.Context()
	before, err := Preflight(ctx, db, qs, "scope-v1", input)
	require.NoError(t, err)
	_, err = Apply(ctx, db, qs, writer, "scope-v1", "9", before.Fingerprint, input, true)
	require.NoError(t, err)
	_, err = Apply(ctx, db, qs, writer, "scope-v1", "9", before.Fingerprint, input, true)
	require.NoError(t, err)
	require.Equal(t, 1, sender.Calls)
	nativefixture.AssertCommittedWitnesses(t, db, 1)
	restored, err := Rollback(ctx, db, qs, writer, "scope-v1", "9", before.Fingerprint, true)
	require.NoError(t, err)
	require.Equal(t, "rolled_back", restored.Status)
	nativefixture.AssertCommittedWitnesses(t, db, 2)
}

//go:build integration

package conditionretire

import (
	nativefixture "github.com/FangcunMount/iam/v5/internal/apiserver/maintenance/policywrite/testutil"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestM3TransactionMaintenanceConditionRetirement(t *testing.T) {
	db := fixture(t, false)
	writer, sender := nativefixture.NativeWriter(t, db)
	ctx := t.Context()
	before, err := Preflight(ctx, db)
	require.NoError(t, err)
	applied, err := Apply(ctx, db, writer, before.Fingerprint, true)
	require.NoError(t, err)
	require.Equal(t, "applied", applied.Status)
	_, err = Apply(ctx, db, writer, before.Fingerprint, true)
	require.NoError(t, err)
	require.Equal(t, 1, sender.Calls)
	nativefixture.AssertCommittedWitnesses(t, db, 1)
	restored, err := Rollback(ctx, db, writer, before.Fingerprint, true)
	require.NoError(t, err)
	require.Equal(t, "rolled_back", restored.Status)
	nativefixture.AssertCommittedWitnesses(t, db, 2)
}

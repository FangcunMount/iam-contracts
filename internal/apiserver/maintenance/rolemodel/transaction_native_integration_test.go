//go:build integration

package rolemodel

import (
	nativefixture "github.com/FangcunMount/iam/v5/internal/apiserver/maintenance/policywrite/testutil"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestM3TransactionMaintenanceRoleModel(t *testing.T) {
	db, qs := migrationDB(t)
	writer, sender := nativefixture.NativeWriter(t, db)
	ctx := t.Context()
	before, err := Preflight(ctx, db, qs)
	require.NoError(t, err)
	applied, err := Apply(ctx, db, qs, writer, before.Fingerprint, true)
	require.NoError(t, err)
	require.Equal(t, "applied", applied.Status)
	_, err = Apply(ctx, db, qs, writer, before.Fingerprint, true)
	require.NoError(t, err)
	require.Equal(t, 1, sender.Calls)
	nativefixture.AssertCommittedWitnesses(t, db, 1)
	restored, err := Rollback(ctx, db, writer, before.Fingerprint, true)
	require.NoError(t, err)
	require.Equal(t, "rolled_back", restored.Status)
	nativefixture.AssertCommittedWitnesses(t, db, 2)
}

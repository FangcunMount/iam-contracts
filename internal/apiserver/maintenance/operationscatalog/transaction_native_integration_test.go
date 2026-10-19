//go:build integration

package operationscatalog

import (
	nativefixture "github.com/FangcunMount/iam/v5/internal/apiserver/maintenance/policywrite/testutil"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestM3TransactionMaintenanceOperationsCatalog(t *testing.T) {
	db := fixture(t)
	writer, sender := nativefixture.NativeWriter(t, db)
	ctx := t.Context()
	before, err := Preflight(ctx, db)
	require.NoError(t, err)
	sender.Reject = true
	_, err = Apply(ctx, db, writer, before.Fingerprint, "9", true)
	require.Error(t, err)
	unchanged, err := Preflight(ctx, db)
	require.NoError(t, err)
	require.Equal(t, before.Fingerprint, unchanged.Fingerprint)
	sender.Reject = false
	applied, err := Apply(ctx, db, writer, before.Fingerprint, "9", true)
	require.NoError(t, err)
	require.Equal(t, "configured", applied.State)
	_, err = Apply(ctx, db, writer, applied.Fingerprint, "9", true)
	require.NoError(t, err)
	require.Equal(t, 2, sender.Calls)
	nativefixture.AssertCommittedWitnesses(t, db, 1)
	var outbox int64
	require.NoError(t, db.Table("test_outbox").Count(&outbox).Error)
	require.Zero(t, outbox)
}

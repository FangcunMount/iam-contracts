package conditionretire

import (
	"context"
	planfixture "github.com/FangcunMount/iam/v5/internal/apiserver/maintenance/policywrite/testutil"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPlannedRetirementPreservesReceiptAndRollbackBoundaries(t *testing.T) {
	db := fixture(t, false)
	ctx := context.Background()
	p, err := Preflight(ctx, db)
	require.NoError(t, err)
	probe := &planfixture.Probe{Stager: &stager{}}
	applied, err := Apply(ctx, db, probe, p.Fingerprint, true)
	require.NoError(t, err)
	repeated, err := Apply(ctx, db, probe, p.Fingerprint, true)
	require.NoError(t, err)
	require.Equal(t, applied.AfterHash, repeated.AfterHash)
	require.Equal(t, 1, probe.Writes)
	restored, err := Rollback(ctx, db, probe, p.Fingerprint, true)
	require.NoError(t, err)
	require.Equal(t, "rolled_back", restored.Status)
	require.Equal(t, 2, probe.Writes)
	_, err = Rollback(ctx, db, probe, p.Fingerprint, true)
	require.Error(t, err)
	require.Equal(t, 2, probe.Writes, "legacy repeated rollback remains a conflict")
}

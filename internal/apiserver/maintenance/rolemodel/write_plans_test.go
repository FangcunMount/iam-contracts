package rolemodel

import (
	"context"
	planfixture "github.com/FangcunMount/iam/v5/internal/apiserver/maintenance/policywrite/testutil"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPlannedRoleModelPreviewApplyAndCompensation(t *testing.T) {
	db, qs := migrationDB(t)
	ctx := context.Background()
	p, err := Preflight(ctx, db, qs)
	require.NoError(t, err)
	probe := &planfixture.Probe{Stager: &recordingStager{}}
	applied, err := Apply(ctx, db, qs, probe, p.Fingerprint, true)
	require.NoError(t, err)
	require.Equal(t, "applied", applied.Status)
	repeated, err := Apply(ctx, db, qs, probe, p.Fingerprint, true)
	require.NoError(t, err)
	require.Equal(t, applied.AfterHash, repeated.AfterHash)
	require.Equal(t, 1, probe.Writes)
	restored, err := Rollback(ctx, db, probe, p.Fingerprint, true)
	require.NoError(t, err)
	require.Equal(t, "rolled_back", restored.Status)
	require.Equal(t, 2, probe.Writes)
	_, err = Rollback(ctx, db, probe, p.Fingerprint, true)
	require.NoError(t, err)
	require.Equal(t, 2, probe.Writes)
	require.Equal(t, "maintenance.role_model.rollback", probe.Requests[2].Operation)
}

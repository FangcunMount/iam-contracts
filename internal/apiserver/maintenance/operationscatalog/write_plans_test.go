package operationscatalog

import (
	"context"
	planfixture "github.com/FangcunMount/iam/v5/internal/apiserver/maintenance/policywrite/testutil"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPlannedCatalogPreviewsNoopAndRejectsDrift(t *testing.T) {
	db := fixture(t)
	ctx := context.Background()
	before, err := Preflight(ctx, db)
	require.NoError(t, err)
	probe := &planfixture.Probe{Stager: outbox{}}
	after, err := Apply(ctx, db, probe, before.Fingerprint, "9", true)
	require.NoError(t, err)
	require.Equal(t, "configured", after.State)
	require.Equal(t, 1, probe.Writes)
	repeated, err := Apply(ctx, db, probe, after.Fingerprint, "9", true)
	require.NoError(t, err)
	require.Equal(t, after.Fingerprint, repeated.Fingerprint)
	require.Equal(t, 1, probe.Writes)
	_, err = Apply(ctx, db, probe, before.Fingerprint, "9", true)
	require.Error(t, err)
	require.Equal(t, 1, probe.Writes)
	require.Equal(t, "maintenance.operations_catalog.apply", probe.Requests[0].Operation)
}

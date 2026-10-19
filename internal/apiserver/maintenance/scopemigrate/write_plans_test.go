package scopemigrate

import (
	"context"
	planfixture "github.com/FangcunMount/iam/v5/internal/apiserver/maintenance/policywrite/testutil"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPlannedScopeCopiesMappingAndPreservesCompensation(t *testing.T) {
	iam, qs, input := applyFixture(t)
	ctx := context.Background()
	p, err := Preflight(ctx, iam, qs, "scope-v1", input)
	require.NoError(t, err)
	original := cloneInput(input)
	probe := &planfixture.Probe{Stager: transactionalStager{}, BeforePreview: func() { input.Mappings[0].Targets[0].StoreIDs[0] = "999" }}
	applied, err := Apply(ctx, iam, qs, probe, "scope-v1", "9", p.Fingerprint, input, true)
	require.NoError(t, err)
	require.Equal(t, original, probe.Requests[0].Input.(map[string]any)["mapping"])
	probe.BeforePreview = nil
	repeated, err := Apply(ctx, iam, qs, probe, "scope-v1", "9", p.Fingerprint, original, true)
	require.NoError(t, err)
	require.Equal(t, applied.AfterHash, repeated.AfterHash)
	require.Equal(t, 1, probe.Writes)
	restored, err := Rollback(ctx, iam, qs, probe, "scope-v1", "9", p.Fingerprint, true)
	require.NoError(t, err)
	require.Equal(t, "rolled_back", restored.Status)
	require.Equal(t, 2, probe.Writes)
	_, err = Rollback(ctx, iam, qs, probe, "scope-v1", "9", p.Fingerprint, true)
	require.NoError(t, err)
	require.Equal(t, 2, probe.Writes)
}

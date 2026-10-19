package accountprovision

import (
	"context"
	crypto "github.com/FangcunMount/iam/v5/internal/apiserver/infra/crypto"
	fixture "github.com/FangcunMount/iam/v5/internal/apiserver/maintenance/policywrite/testutil"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPlannedReviewerScopeRetainsCredentialsOutsideFrozenInput(t *testing.T) {
	db, input := reviewerFixture(t)
	ctx := context.Background()
	hasher := crypto.NewArgon2Hasher("")
	before, err := ReviewerScopePreflight(ctx, db, input, hasher, 1)
	require.NoError(t, err)
	probe := &fixture.Probe{Stager: scopeTestStager{}}
	after, err := ReviewerScopeApply(ctx, db, input, hasher, 1, before.Fingerprint, probe)
	require.NoError(t, err)
	require.Equal(t, "scope_matches_reference", after.State)
	request := probe.Requests[0].Input.(map[string]any)
	require.NotContains(t, request, "password")
	require.NotContains(t, request, "credential")
	current, err := ReviewerScopePreflight(ctx, db, input, hasher, 1)
	require.NoError(t, err)
	_, err = ReviewerScopeApply(ctx, db, input, hasher, 1, current.Fingerprint, probe)
	require.NoError(t, err)
	require.Equal(t, 1, probe.Writes)
	_, err = ReviewerScopeApply(ctx, db, input, hasher, 1, before.Fingerprint, probe)
	require.Error(t, err)
	require.Equal(t, 1, probe.Writes)
}

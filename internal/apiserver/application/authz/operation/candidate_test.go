package operation

import (
	"bytes"
	"testing"
	"time"

	change "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/policychange"
	"github.com/FangcunMount/reliable-messaging/transaction"
	"github.com/FangcunMount/reliable-messaging/transport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func candidateFixture(t *testing.T, key string, changed bool) Candidate {
	t.Helper()
	command, err := change.NewCommandIntent("role.create", "user:7", "global", key, map[string]string{"name": "test"})
	require.NoError(t, err)
	id := uuid.NewString()
	var plan *transaction.FrozenPlan
	if changed {
		physical, err := transport.PhysicalTopicV2("iam.authz.version.v2", "TRANSACTION", "iam.mysql", "m3-1")
		require.NoError(t, err)
		frozen, err := change.FreezeVersionPlan(command, 1, id, uuid.NewString(), time.Now().UTC(), "iam.authz.version.v2", physical)
		require.NoError(t, err)
		plan = &frozen
	}
	result, err := NewCandidate(command, id, transaction.Binding{Owner: "iam.mysql", StoreID: "primary", WriterEpoch: "m3-1"}, 1, []Permission{{Resource: "iam:authz:collection:roles", Action: "create"}}, plan)
	require.NoError(t, err)
	return result
}
func TestCandidatePreservesWinnerWireAndSeparatesGeneratedArtifactsFromInput(t *testing.T) {
	first := candidateFixture(t, "same", true)
	second := candidateFixture(t, "same", true)
	require.True(t, first.SameRequest(second))
	require.Equal(t, first.RequestKey(), second.RequestKey())
	require.NotEqual(t, first.Fingerprint(), second.Fingerprint())
	decoded, err := DecodeCandidate(first.Encode())
	require.NoError(t, err)
	require.Equal(t, first.Encode(), decoded.Encode())
	original, err := first.Plan()
	require.NoError(t, err)
	restored, err := decoded.Plan()
	require.NoError(t, err)
	require.Equal(t, original.Input().Branches[0].Wire, restored.Input().Branches[0].Wire)
	raw := first.Encode()
	raw[0] = 'x'
	require.True(t, first.Valid())
	permissions := first.Permissions()
	permissions[0].Action = "delete"
	require.Equal(t, "create", first.Permissions()[0].Action)
	require.False(t, first.SameRequest(candidateFixture(t, "different", true)))
}
func TestCandidateSupportsDurableNoopWithoutCreatingFakeMessage(t *testing.T) {
	candidate := candidateFixture(t, "noop", false)
	require.False(t, candidate.Changed())
	require.Equal(t, "user:7", candidate.Principal())
	require.Equal(t, "global", candidate.Scope())
	require.Equal(t, "role.create", candidate.Name())
	_, err := candidate.Plan()
	require.ErrorIs(t, err, ErrInvalid)
	decoded, err := DecodeCandidate(candidate.Encode())
	require.NoError(t, err)
	require.False(t, decoded.Changed())
}
func TestCandidateRejectsNoncanonicalOrConflictingEvidence(t *testing.T) {
	candidate := candidateFixture(t, "first", true)
	raw := candidate.Encode()
	for _, bad := range [][]byte{append(bytes.Clone(raw), ' '), bytes.Replace(raw, []byte(`"schema":1`), []byte(`"schema":2`), 1), bytes.Replace(raw, []byte(`"schema":1`), []byte(`"schema":1,"schema":1`), 1), bytes.Replace(raw, []byte(`"expected_policy_version":1`), []byte(`"expected_policy_version":2`), 1), append([]byte(`{"unknown":1,`), raw[1:]...)} {
		_, err := DecodeCandidate(bad)
		require.ErrorIs(t, err, ErrInvalid)
	}
	data := candidate.data
	data.Permissions = nil
	_, err := encode(data)
	require.ErrorIs(t, err, ErrInvalid, "human result query needs original permissions")
	data = candidate.data
	data.ID = uuid.NewString()
	_, err = encode(data)
	require.ErrorIs(t, err, ErrInvalid, "journal identity must bind the original SDK operation")
	data = candidate.data
	data.Binding.Owner = "qs.mysql"
	_, err = encode(data)
	require.ErrorIs(t, err, ErrInvalid)
}

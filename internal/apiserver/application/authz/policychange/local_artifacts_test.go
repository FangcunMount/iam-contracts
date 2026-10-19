package policychange

import (
	"bytes"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"github.com/FangcunMount/reliable-messaging/transaction"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestLocalArtifactsFreezeIDsWithoutChangingBusinessWireOrInputHash(t *testing.T) {
	intent, err := NewCommandIntent("role.create", "user:7", "global", "key", map[string]string{"name": "role"})
	require.NoError(t, err)
	at := time.Date(2026, 10, 19, 14, 0, 0, 0, time.UTC)
	key := ArtifactKey("role", "role")
	a := LocalArtifacts{AuditAt: at, IDs: map[string]meta.ID{key: 77}}
	plain, err := FreezeVersionPlan(intent, 41, "operation", "event", at, "logical", "physical")
	require.NoError(t, err)
	frozen, err := FreezeVersionPlanWithArtifacts(intent, 41, "operation", "event", at, "logical", "physical", a)
	require.NoError(t, err)
	require.Equal(t, plain.Input().Branches[0].Wire, frozen.Input().Branches[0].Wire)
	require.Equal(t, transaction.RequestFor(plain), transaction.RequestFor(frozen))
	require.NotEqual(t, plain.Digest(), frozen.Digest())
	a.IDs[key] = 88
	encoded, err := transaction.EncodePlan(frozen)
	require.NoError(t, err)
	decoded, err := transaction.DecodePlan(encoded)
	require.NoError(t, err)
	plan, err := DecodeVersionPlan(decoded)
	require.NoError(t, err)
	require.Equal(t, int64(42), plan.Target)
	require.Equal(t, []byte(`{"version":42}`), plan.EventPayload())
	local, ok := plan.Artifacts()
	require.True(t, ok)
	require.Equal(t, meta.ID(77), local.IDs[key])
	local.IDs[key] = 99
	next, _ := plan.Artifacts()
	require.Equal(t, meta.ID(77), next.IDs[key])
	require.True(t, bytes.Contains(decoded.Input().Branches[0].Message.Input().Payload, []byte(`"77"`)))
	require.NoError(t, MatchCommand(decoded, intent))
}
func TestLocalArtifactsRejectInvalidIdentityAndClock(t *testing.T) {
	at := time.Now().UTC()
	for _, a := range []LocalArtifacts{{}, {AuditAt: at, IDs: map[string]meta.ID{"key": 0}}, {AuditAt: at, IDs: map[string]meta.ID{"one": 7, "two": 7}}, {AuditAt: at, IDs: map[string]meta.ID{"": 7}}} {
		require.False(t, a.valid())
	}
	builder := NewArtifactBuilder(at)
	err := AllocateKeys("two", "one", "one")(builder)
	require.NoError(t, err)
	a, err := builder.Freeze()
	require.NoError(t, err)
	require.Len(t, a.IDs, 2)
	require.NotEqual(t, a.IDs["one"], a.IDs["two"])
}

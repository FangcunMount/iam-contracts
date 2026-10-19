package policychange

import (
	"bytes"
	"math"
	"testing"
	"time"

	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/scope"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"github.com/FangcunMount/reliable-messaging/transaction"
	"github.com/stretchr/testify/require"
)

func TestCommandIntentSeparatesSubjectScopeOperationAndStableInput(t *testing.T) {
	makeIntent := func(op, subject, scope, key string, input any) CommandIntent {
		c, e := NewCommandIntent(op, subject, scope, key, input)
		require.NoError(t, e)
		return c
	}
	a := makeIntent("role.update", "user:7", "org:9", "same-key", map[string]any{"id": "42", "display_name": "首次名称"})
	b := makeIntent("role.update", "user:7", "org:9", "same-key", map[string]any{"display_name": "首次名称", "id": "42"})
	require.Equal(t, a.Key(), b.Key())
	require.Equal(t, a.InputHash(), b.InputHash())
	for _, c := range []CommandIntent{
		makeIntent("role.update", "user:8", "org:9", "same-key", map[string]string{"id": "42"}),
		makeIntent("role.update", "user:7", "org:8", "same-key", map[string]string{"id": "42"}),
		makeIntent("role.delete", "user:7", "org:9", "same-key", map[string]string{"id": "42"}),
	} {
		require.NotEqual(t, a.Key(), c.Key())
		require.NotEqual(t, a.InputHash(), c.InputHash())
	}
	changed := makeIntent("role.update", "user:7", "org:9", "same-key", map[string]string{"id": "43"})
	require.Equal(t, a.Key(), changed.Key())
	require.NotEqual(t, a.InputHash(), changed.InputHash())
	copy := a.Input()
	copy[0] = 0
	require.True(t, bytes.HasPrefix(a.Input(), []byte("{")))
	for _, input := range []any{nil, []string{"missing object"}, math.NaN()} {
		_, e := NewCommandIntent("role.update", "user:7", "global", "key", input)
		require.Error(t, e)
	}
	_, e := NewCommandIntent("role.update", "user:7", "global", " key ", map[string]string{})
	require.Error(t, e)
}

func TestPolicyPlanWinnerRetainsFirstIdentityVersionTimeAndLegacyWire(t *testing.T) {
	c, e := NewCommandIntent("role.update", "user:7", "org:9", "same-key", map[string]string{"id": "42", "name": "新名称"})
	require.NoError(t, e)
	at, e := time.Parse(time.RFC3339Nano, "2026-10-19T12:00:00.123456789+08:00")
	require.NoError(t, e)
	first, e := FreezeVersionPlan(c, 41, "original-op", "first-event", at, "iam.authz.version.v2", "physical.transaction")
	require.NoError(t, e)
	candidate, e := FreezeVersionPlan(c, 88, "new-op", "new-event", at.Add(time.Hour), "iam.authz.version.v2", "physical.transaction")
	require.NoError(t, e)
	require.NotEqual(t, first.Digest(), candidate.Digest())
	require.NoError(t, MatchCommand(first, c))
	raw, e := transaction.EncodePlan(first)
	require.NoError(t, e)
	restored, e := transaction.DecodePlan(raw)
	require.NoError(t, e)
	winner, e := DecodeVersionPlan(restored)
	require.NoError(t, e)
	require.Equal(t, int64(41), winner.Expected)
	require.Equal(t, int64(42), winner.Target)
	require.Equal(t, "first-event", winner.Message.Input().ID)
	require.Equal(t, at.Format(time.RFC3339Nano), winner.Message.Input().OccurredAt)
	want := `{"type":"component-base.messaging.message.v1","uuid":"first-event","metadata":{"aggregate_id":"42","aggregate_type":"PolicyVersion","event_type":"iam.authz.version_changed.v2","source":"iam-outbox-relay"},"payload":"eyJ2ZXJzaW9uIjo0Mn0="}`
	require.Equal(t, want, string(restored.Input().Branches[0].Wire))
	for _, expected := range []int64{-1, 0, math.MaxInt64} {
		_, e := FreezeVersionPlan(c, expected, "op", "event", at, "logical", "physical")
		require.Error(t, e)
	}
	last, e := FreezeVersionPlan(c, math.MaxInt64-1, "op", "event", at, "logical", "physical")
	require.NoError(t, e)
	v, e := DecodeVersionPlan(last)
	require.NoError(t, e)
	require.Equal(t, int64(math.MaxInt64), v.Target)
	different, e := NewCommandIntent("role.update", "user:7", "org:9", "same-key", map[string]string{"id": "42", "name": "changed"})
	require.NoError(t, e)
	require.Error(t, MatchCommand(first, different))
	for _, kind := range []string{"owner", "wire", "revision"} {
		t.Run(kind, func(t *testing.T) {
			in := first.Input()
			switch kind {
			case "owner":
				in.Owner = "wrong"
			case "wire":
				in.Branches[0].Wire = []byte("{}")
			case "revision":
				in.Branches[0].RouteRevision = "m3-2"
			}
			p, e := transaction.NewPlan(in)
			require.NoError(t, e)
			_, e = DecodeVersionPlan(p)
			require.Error(t, e)
		})
	}
}

func TestOpaqueAssignmentScopeCannotDisappearFromTheInputHash(t *testing.T) {
	s, err := scope.New(meta.FromUint64(7), scope.Kind("stores"), []meta.ID{meta.FromUint64(42)})
	require.NoError(t, err)
	_, err = NewCommandIntent("assignment.replace", "user:7", "org:7", "key", struct{ Scope scope.Scope }{s})
	require.ErrorIs(t, err, ErrInvalidChangePlan, "private domain fields would be silently omitted by json.Marshal")
	a, err := NewCommandIntent("assignment.replace", "user:7", "org:7", "key", map[string]any{"org_id": s.OrgID().String(), "kind": s.Kind(), "store_ids": []string{"42"}})
	require.NoError(t, err)
	b, err := NewCommandIntent("assignment.replace", "user:7", "org:7", "key", map[string]any{"org_id": s.OrgID().String(), "kind": s.Kind(), "store_ids": []string{"43"}})
	require.NoError(t, err)
	require.Equal(t, a.Key(), b.Key())
	require.NotEqual(t, a.InputHash(), b.InputHash())
}

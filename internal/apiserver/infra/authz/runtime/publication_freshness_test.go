package runtime_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/policypublication"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/authorization"
	authzruntime "github.com/FangcunMount/iam/v5/internal/apiserver/infra/authz/runtime"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
	"github.com/stretchr/testify/require"
)

// Notifications select a target; only authoritative reads may renew the proof.
// This deterministic boundary test supplements, rather than replaces, Broker
// fanout acceptance of independently subscribed application instances.
func TestPublicationTwoRuntimesOutOfOrderDoesNotRenewFreshness(t *testing.T) {
	now := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)
	sources := make([]*durableSource, 2)
	runtimes := make([]*authzruntime.Runtime, 2)
	handlers := make([]*policypublication.Service, 2)
	for i := range runtimes {
		sources[i] = &durableSource{dataset: assessmentDataset(t), version: 9}
		var err error
		runtimes[i], err = authzruntime.NewRuntime(t.Context(), sources[i], authorization.NewEvaluator(), authzruntime.WithClock(func() time.Time { return now }))
		require.NoError(t, err)
		handlers[i] = policypublication.NewService(runtimes[i], runtimes[i])
	}
	request := checkRequest(t, 2, "retry", "adhoc")
	for _, r := range runtimes {
		decision, err := r.Check(t.Context(), request)
		require.NoError(t, err)
		require.True(t, decision.Allowed)
	}
	now = now.Add(59 * time.Second)
	for _, index := range []int{1, 0} {
		// An older and a repeated event cannot turn a stale snapshot into fresh
		// authority. A future target with a lagging source cannot renew it either.
		for _, version := range []int{8, 9, 10, 9, 10} {
			require.NoError(t, handlers[index].Handle(context.Background(), []byte(fmt.Sprintf(`{"version":%d}`, version)), ""))
		}
		require.False(t, runtimes[index].PolicyVersionLoaded(10))
	}
	now = now.Add(time.Second)
	for _, r := range runtimes {
		_, err := r.Check(t.Context(), request)
		require.True(t, perrors.IsCode(err, code.ErrAuthorizationPolicyUnavailable))
	}
	for i, r := range runtimes {
		sources[i].dataset.Grants = nil
		sources[i].dataset.Version = 10
		sources[i].version = 10
		require.NoError(t, r.Reconcile(t.Context())) // recovery even if next hint is lost
		decision, err := r.Check(t.Context(), request)
		require.NoError(t, err)
		require.False(t, decision.Allowed)
		require.EqualValues(t, 10, decision.PolicyVersion)
		require.NoError(t, handlers[i].Handle(t.Context(), []byte(`{"version":9}`), ""))
		require.True(t, r.PolicyVersionLoaded(10))
	}
}

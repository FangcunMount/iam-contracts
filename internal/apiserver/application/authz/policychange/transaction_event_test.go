package policychange

import (
	"context"
	"testing"
	"time"

	domain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/policy"
	"github.com/FangcunMount/iam/v5/pkg/event"
	"github.com/FangcunMount/reliable-messaging/transaction"
	"github.com/stretchr/testify/require"
)

type frozenEventRecorder struct{ events []event.DomainEvent }

func (r *frozenEventRecorder) Stage(_ context.Context, events ...event.DomainEvent) error {
	r.events = append(r.events, events...)
	return nil
}

func TestOriginalExecutorStagesTheWinnersFrozenEventOnly(t *testing.T) {
	intent, err := NewCommandIntent("role.update", "user:7", "org:9", "key", map[string]string{"id": "42"})
	require.NoError(t, err)
	at := time.Date(2026, 10, 19, 12, 0, 0, 123456789, time.FixedZone("original", 8*3600))
	plan, err := FreezeVersionPlan(intent, 41, "operation", "original-event", at, "logical", "physical")
	require.NoError(t, err)
	e := transaction.Execution{Plan: plan, Fence: 1, WriterEpoch: "m3-1", Token: [32]byte{1}, LeaseUntil: at.Add(time.Minute)}
	ctx, err := WithExecutionPlan(context.Background(), e, intent)
	require.NoError(t, err)
	recorder := new(frozenEventRecorder)
	for range 2 {
		require.NoError(t, StagePolicyVersionChanged(ctx, recorder, &domain.PolicyVersion{Version: 42}))
	}
	require.Len(t, recorder.events, 2)
	for _, event := range recorder.events {
		require.Equal(t, "original-event", event.EventID())
		require.True(t, at.Equal(event.OccurredAt()))
		require.Equal(t, "42", event.AggregateID())
		require.Equal(t, domain.VersionChangedPayload{Version: 42}, event.Payload())
	}
	require.ErrorIs(t, StagePolicyVersionChanged(ctx, recorder, &domain.PolicyVersion{Version: 43}), ErrInvalidChangePlan)
	require.NoError(t, StagePolicyVersionChanged(ctx, recorder, nil))
	require.Len(t, recorder.events, 2, "no-op or stale version manufactured a notification")
	observed := e
	observed.Token = [32]byte{}
	_, err = WithExecutionPlan(context.Background(), observed, intent)
	require.ErrorIs(t, err, transaction.ErrStaleExecution)
	_, err = WithExecutionPlan(nil, e, intent)
	require.Error(t, err)
	changed, err := NewCommandIntent("role.update", "user:7", "org:9", "key", map[string]string{"id": "43"})
	require.NoError(t, err)
	_, err = WithExecutionPlan(context.Background(), e, changed)
	require.Error(t, err)
}

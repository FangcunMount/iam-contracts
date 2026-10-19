package policychange

import (
	"context"
	"strconv"
	"time"

	domain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/policy"
	"github.com/FangcunMount/iam/v5/internal/apiserver/eventing"
	"github.com/FangcunMount/iam/v5/pkg/event"
	"github.com/FangcunMount/reliable-messaging/transaction"
)

type executionPlanKey struct{}

// WithExecutionPlan is used by the host executor after acquiring the original
// root guard. Recovered/observed executions have no write capability and fail.
// It does not itself prove a guard, half preparation or root commitment.
func WithExecutionPlan(ctx context.Context, e transaction.Execution, command CommandIntent) (context.Context, error) {
	if ctx == nil || !e.Valid() {
		return nil, transaction.ErrStaleExecution
	}
	if err := MatchCommand(e.Plan, command); err != nil {
		return nil, err
	}
	plan, err := DecodeVersionPlan(e.Plan)
	if err != nil {
		return nil, err
	}
	return context.WithValue(ctx, executionPlanKey{}, plan), nil
}
func ExecutionPlan(ctx context.Context) (VersionPlan, bool) {
	if ctx == nil {
		return VersionPlan{}, false
	}
	plan, ok := ctx.Value(executionPlanKey{}).(VersionPlan)
	return plan, ok
}
func (p VersionPlan) event() (event.DomainEvent, error) {
	in := p.Message.Input()
	at, err := time.Parse(time.RFC3339Nano, in.OccurredAt)
	if err != nil || p.Target < 2 || p.Expected != p.Target-1 {
		return nil, ErrInvalidChangePlan
	}
	return event.Event[domain.VersionChangedPayload]{BaseEvent: event.BaseEvent{ID: in.ID, EventTypeValue: eventing.AuthzVersionChanged, OccurredAtValue: at, AggregateTypeValue: "PolicyVersion", AggregateIDValue: strconv.FormatInt(p.Target, 10)}, Data: domain.VersionChangedPayload{Version: p.Target}}, nil
}

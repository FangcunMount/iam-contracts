package uow

import (
	"context"
	"errors"
	"testing"

	change "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/policychange"
	policy "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/policy"
	"github.com/stretchr/testify/require"
)

type planningVersion struct{ value int64 }

func (p planningVersion) GetCurrent(context.Context) (*policy.PolicyVersion, error) {
	return &policy.PolicyVersion{Version: p.value}, nil
}
func (p planningVersion) GetOrCreate(context.Context) (*policy.PolicyVersion, error) {
	panic("planning cannot initialize policy")
}
func (p planningVersion) Increment(context.Context, string, string) (*policy.PolicyVersion, error) {
	panic("planning cannot mutate policy")
}

type planningRead struct {
	version int64
	calls   int
}

func (p *planningRead) ReadPlan(ctx context.Context, fn func(context.Context, TxRepositories) error) error {
	p.calls++
	return fn(ctx, TxRepositories{PolicyVersions: planningVersion{p.version}})
}

type writeRoute struct {
	planned, legacy int
	failure         error
}

func (p *writeRoute) WithinTx(context.Context, func(context.Context, TxRepositories) error) error {
	p.legacy++
	return nil
}
func (p *writeRoute) WithinPlannedTx(context.Context, WriteRequest, func(context.Context, TxRepositories) error) error {
	p.planned++
	return p.failure
}
func TestReadPlanningPinsPermissionVersionAndPreservesNoopPrecondition(t *testing.T) {
	reader := &planningRead{version: 42}
	request := WriteRequest{Operation: "assignment.replace", Scope: "org:7", Preview: func(ctx context.Context, _ TxRepositories) (bool, error) {
		require.NoError(t, change.RequireDecisionVersion(ctx, 42))
		require.ErrorIs(t, change.RequireDecisionVersion(ctx, 41), policy.ErrStaleVersion)
		require.ErrorIs(t, CheckExpectedVersion(42, 41), policy.ErrStaleVersion)
		_, execution := change.ExecutionPlan(ctx)
		require.False(t, execution, "read evidence manufactured write capability")
		return false, nil
	}}
	snapshot, err := PrepareChange(context.Background(), reader, request)
	require.NoError(t, err)
	require.Equal(t, WriteSnapshot{Expected: 42, Changed: false}, snapshot)
	require.Equal(t, 1, reader.calls)
	require.NoError(t, CheckExpectedVersion(42, 0))
	require.ErrorIs(t, CheckExpectedVersion(42, -1), policy.ErrInvalidVersionPlan)
	request.Preview = func(context.Context, TxRepositories) (bool, error) { return false, policy.ErrStaleVersion }
	snapshot, err = PrepareChange(context.Background(), reader, request)
	require.ErrorIs(t, err, policy.ErrStaleVersion)
	require.Equal(t, WriteSnapshot{}, snapshot)
}
func TestConfiguredPlannerFailureCannotFallBackToLegacyUnitOfWork(t *testing.T) {
	failure := errors.New("half preparation unavailable")
	route := &writeRoute{failure: failure}
	err := WithinWrite(context.Background(), route, WriteRequest{Operation: "role.update", Scope: "global"}, nil)
	require.ErrorIs(t, err, failure)
	require.Equal(t, 1, route.planned)
	require.Zero(t, route.legacy)
	require.ErrorIs(t, WithinWrite(context.Background(), nil, WriteRequest{}, nil), ErrWritePlanRequired)
}

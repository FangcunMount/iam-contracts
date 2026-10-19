package messaging

import (
	"context"
	"errors"
	"testing"

	"github.com/FangcunMount/reliable-messaging/transaction"
	"github.com/stretchr/testify/require"
)

type senderFunc func(context.Context, transaction.Execution, transaction.LocalExecutor) (transaction.Result, error)

func (s senderFunc) Execute(ctx context.Context, e transaction.Execution, fn transaction.LocalExecutor) (transaction.Result, error) {
	return s(ctx, e, fn)
}
func TestExecuteJoiningLocalRetainsResultOwnershipAfterCanceledSnapshot(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	type observed struct {
		completion LocalCompletion
		err        error
	}
	returned := make(chan observed)
	sdkReturned := make(chan struct{})
	sdk := senderFunc(func(ctx context.Context, e transaction.Execution, local transaction.LocalExecutor) (transaction.Result, error) {
		go func() { _, _ = local(ctx, e) }()
		<-entered
		close(sdkReturned)
		return transaction.Result{Local: transaction.LocalUnknown}, context.Canceled
	})
	go func() {
		_, completion, err := ExecuteJoiningLocal(context.Background(), sdk, transaction.Execution{}, func(context.Context, transaction.Execution) (transaction.LocalOutcome, error) {
			close(entered)
			<-release
			return transaction.LocalCommitted, nil
		})
		returned <- observed{completion, err}
	}()
	<-sdkReturned
	select {
	case <-returned:
		t.Fatal("returned while host callback retained mutable results")
	default:
	}
	close(release)
	observation := <-returned
	require.ErrorIs(t, observation.err, context.Canceled)
	completion := observation.completion
	require.True(t, completion.Started)
	require.Equal(t, transaction.LocalCommitted, completion.Outcome)
}
func TestExecuteJoiningLocalRejectsLateCallbackAndNeverConfusesNilWithCommit(t *testing.T) {
	var captured transaction.LocalExecutor
	sdk := senderFunc(func(_ context.Context, _ transaction.Execution, local transaction.LocalExecutor) (transaction.Result, error) {
		captured = local
		return transaction.Result{Local: transaction.LocalUnknown}, context.Canceled
	})
	_, completion, err := ExecuteJoiningLocal(context.Background(), sdk, transaction.Execution{}, func(context.Context, transaction.Execution) (transaction.LocalOutcome, error) {
		t.Fatal("late business root started")
		return transaction.LocalCommitted, nil
	})
	require.ErrorIs(t, err, context.Canceled)
	require.False(t, completion.Started)
	state, err := captured(context.Background(), transaction.Execution{})
	require.Equal(t, transaction.LocalUnknown, state)
	require.ErrorIs(t, err, transaction.ErrStaleExecution)
	sdk = senderFunc(func(ctx context.Context, e transaction.Execution, local transaction.LocalExecutor) (transaction.Result, error) {
		_, err := local(ctx, e)
		return transaction.Result{Local: transaction.LocalUnknown}, err
	})
	result, completion, err := ExecuteJoiningLocal(context.Background(), sdk, transaction.Execution{}, func(context.Context, transaction.Execution) (transaction.LocalOutcome, error) {
		return transaction.LocalUnknown, nil
	})
	require.NoError(t, err)
	require.True(t, completion.Started)
	require.Equal(t, transaction.LocalUnknown, result.Local)
	require.Equal(t, transaction.LocalUnknown, completion.Outcome)
}
func TestExecuteJoiningLocalPreservesLocalFailure(t *testing.T) {
	failure := errors.New("business CAS conflict")
	sdk := senderFunc(func(ctx context.Context, e transaction.Execution, local transaction.LocalExecutor) (transaction.Result, error) {
		_, err := local(ctx, e)
		return transaction.Result{Local: transaction.LocalAborted}, err
	})
	_, completion, err := ExecuteJoiningLocal(context.Background(), sdk, transaction.Execution{}, func(context.Context, transaction.Execution) (transaction.LocalOutcome, error) {
		return transaction.LocalAborted, failure
	})
	require.ErrorIs(t, err, failure)
	require.ErrorIs(t, completion.Err, failure)
}

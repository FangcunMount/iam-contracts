package messaging

import (
	"context"
	"errors"
	"sync"

	"github.com/FangcunMount/reliable-messaging/transaction"
)

// LocalCompletion describes the actual host callback, independently of an early
// canceled SDK result snapshot. Only the guarded host executor can prove its
// own root commit ACK; generic callers must not treat a callback's nil as proof.
type LocalCompletion struct {
	Started bool
	Outcome transaction.LocalOutcome
	Err     error
}

// ExecuteJoiningLocal closes callback admission when Execute returns and joins
// an already admitted local callback before returning host-owned result memory.
// Broker RPCs still belong to the SDK lifecycle. A deadline does not free a
// running callback or permit a late callback to start another business root.
func ExecuteJoiningLocal(ctx context.Context, sender transaction.Sender, e transaction.Execution, local transaction.LocalExecutor) (transaction.Result, LocalCompletion, error) {
	if ctx == nil || sender == nil || local == nil {
		return transaction.InitialResult(e), LocalCompletion{}, errors.New("transaction sender, context and original local executor required")
	}
	var mu sync.Mutex
	completion := LocalCompletion{Outcome: transaction.LocalNotStarted}
	closed := false
	done := make(chan struct{})
	result, err := sender.Execute(ctx, e, func(localCtx context.Context, execution transaction.Execution) (outcome transaction.LocalOutcome, localErr error) {
		mu.Lock()
		if closed || completion.Started {
			mu.Unlock()
			return transaction.LocalUnknown, transaction.ErrStaleExecution
		}
		completion.Started = true
		completion.Outcome = transaction.LocalUnknown
		mu.Unlock()
		outcome = transaction.LocalUnknown
		defer func() { mu.Lock(); completion.Outcome = outcome; completion.Err = localErr; mu.Unlock(); close(done) }()
		return local(localCtx, execution)
	})
	mu.Lock()
	closed = true
	started := completion.Started
	mu.Unlock()
	if started {
		<-done
	}
	mu.Lock()
	final := completion
	mu.Unlock()
	return result, final, err
}

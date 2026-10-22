package process

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/FangcunMount/reliable-messaging/transaction"
)

// The final transaction host drains real request handlers while Checker and
// original MySQL remain alive. Only this mode changes legacy close ordering.
func runTransactionShutdownSequence(deps shutdownSequenceDeps) error {
	for _, step := range []struct {
		name string
		run  func() error
	}{
		{"HTTP", deps.closeHTTP}, {"gRPC", deps.closeGRPC},
	} {
		if step.run != nil {
			if err := step.run(); err != nil {
				return fmt.Errorf("transaction shutdown %s: %w", step.name, err)
			}
		}
	}
	if deps.stopPolicySync != nil {
		if err := deps.stopPolicySync(context.Background()); err != nil {
			return err
		}
	}
	var failures []error
	deps.lifecycle.Run(func(_ string, err error) { failures = append(failures, err) })
	if err := errors.Join(failures...); err != nil {
		return err
	}
	if deps.stopReliable != nil {
		if err := deps.stopReliable(context.Background()); err != nil {
			return err
		}
	}
	for _, step := range []struct {
		name string
		run  func() error
	}{
		{"suggest", deps.suggestCleanup}, {"identity", deps.identityCleanup}, {"MySQL", deps.closeDatabase},
	} {
		if step.run != nil {
			if err := step.run(); err != nil {
				return fmt.Errorf("transaction shutdown %s: %w", step.name, err)
			}
		}
	}
	return nil
}

type transactionShutdown struct {
	deps      shutdownSequenceDeps
	once      sync.Once
	requested atomic.Bool
	done      chan struct{}
	result    error
}

func newTransactionShutdown(deps shutdownSequenceDeps) *transactionShutdown {
	return &transactionShutdown{deps: deps, done: make(chan struct{})}
}
func (s *transactionShutdown) Close(ctx context.Context) error {
	s.once.Do(func() {
		s.requested.Store(true)
		if s.deps.beginDrain != nil {
			s.deps.beginDrain()
		}
		deps := s.deps
		deps.beginDrain = nil
		go func() { defer close(s.done); s.result = runShutdownSequence(deps) }()
	})
	return s.Wait(ctx)
}
func (s *transactionShutdown) Wait(ctx context.Context) error {
	if !s.requested.Load() {
		return errors.New("transaction host shutdown not requested")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-s.done:
		return s.result
	default:
	}
	select {
	case <-s.done:
		return s.result
	case <-ctx.Done():
		select {
		case <-s.done:
			return s.result
		default:
		}
		return &transaction.ShutdownIncomplete{RuntimePending: true, Cause: ctx.Err()}
	}
}

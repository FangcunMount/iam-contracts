package operation

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrQueryAdmissionClosed = errors.New("IAM operation query admission closed")
var ErrQueryCapacity = errors.New("IAM operation query capacity exceeded")

type QueryConfig struct {
	Concurrency int
	Timeout     time.Duration
}
type queryLifecycle struct {
	mu                 sync.Mutex
	active, limit      int
	closed, doneClosed bool
	done               chan struct{}
}

func (l *queryLifecycle) enter() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return ErrQueryAdmissionClosed
	}
	if l.active >= l.limit {
		return ErrQueryCapacity
	}
	l.active++
	return nil
}
func (l *queryLifecycle) leave() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.active--
	if l.closed && l.active == 0 && !l.doneClosed {
		close(l.done)
		l.doneClosed = true
	}
}
func (s *QueryService) StopAdmission() {
	s.life.mu.Lock()
	defer s.life.mu.Unlock()
	s.life.closed = true
	if s.life.active == 0 && !s.life.doneClosed {
		close(s.life.done)
		s.life.doneClosed = true
	}
}

type QueryShutdownIncomplete struct {
	Pending int
	Cause   error
}

func (*QueryShutdownIncomplete) Error() string   { return "IAM operation queries have not drained" }
func (e *QueryShutdownIncomplete) Unwrap() error { return e.Cause }
func (s *QueryService) Wait(ctx context.Context) error {
	if ctx == nil {
		return ErrInvalid
	}
	s.life.mu.Lock()
	closed := s.life.closed
	s.life.mu.Unlock()
	if !closed {
		return ErrQueryAdmissionClosed
	}
	select {
	case <-s.life.done:
		return nil
	default:
	}
	select {
	case <-s.life.done:
		return nil
	case <-ctx.Done():
		s.life.mu.Lock()
		pending := s.life.active
		s.life.mu.Unlock()
		if pending == 0 {
			return nil
		}
		return &QueryShutdownIncomplete{Pending: pending, Cause: ctx.Err()}
	}
}

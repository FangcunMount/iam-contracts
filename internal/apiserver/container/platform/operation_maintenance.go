package platform

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/FangcunMount/reliable-messaging/transaction"
)

type OperationMaintenanceStats struct {
	Aborted, Errors int64
	Running         bool
}

// One synchronous DB call owns its capacity until it actually returns. Context
// cancellation cannot make a blocked driver disappear from shutdown accounting.
type operationMaintenance struct {
	mu               sync.Mutex
	interval, budget time.Duration
	batch            int
	abort            func(context.Context, int) (int, error)
	cancel           context.CancelFunc
	done             chan struct{}
	stopped          bool
	stats            OperationMaintenanceStats
}

func (m *operationMaintenance) start() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.stopped || m.cancel != nil || m.abort == nil || m.batch < 1 || m.batch > 100 || m.interval <= 0 || m.budget <= 0 {
		return errors.New("invalid operation maintenance lifecycle")
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.done = make(chan struct{})
	m.stats.Running = true
	go func() {
		defer close(m.done)
		defer func() { m.mu.Lock(); m.stats.Running = false; m.mu.Unlock() }()
		ticker := time.NewTicker(m.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
			if ctx.Err() != nil {
				return
			}
			check, finish := context.WithTimeout(ctx, m.budget)
			n, err := m.abort(check, m.batch)
			finish()
			m.mu.Lock()
			m.stats.Aborted += int64(n)
			if err != nil {
				m.stats.Errors++
			}
			m.mu.Unlock()
		}
	}()
	return nil
}
func (m *operationMaintenance) close(ctx context.Context) error {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	m.stopped = true
	cancel, done := m.cancel, m.done
	m.mu.Unlock()
	if cancel == nil {
		return nil
	}
	cancel()
	select {
	case <-done:
		return nil
	default:
	}
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return &transaction.ShutdownIncomplete{RuntimePending: true, Cause: ctx.Err()}
	}
}
func (m *operationMaintenance) snapshot() OperationMaintenanceStats {
	if m == nil {
		return OperationMaintenanceStats{}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stats
}

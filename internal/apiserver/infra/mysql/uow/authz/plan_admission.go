package authz

import (
	"context"
	"github.com/FangcunMount/reliable-messaging/transaction"
	"sync"
)

// planAdmission also counts lookup/read-only planning before SDK admission.
// Closing the SDK alone cannot truthfully account for these host operations.
type planAdmission struct {
	mu         sync.Mutex
	active     int
	stopping   bool
	slots      chan struct{}
	stop, done chan struct{}
}

func newPlanAdmission(limit int) *planAdmission {
	return &planAdmission{slots: make(chan struct{}, limit), stop: make(chan struct{}), done: make(chan struct{})}
}
func (p *planAdmission) enter(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case <-p.stop:
		return ErrWriteAdmissionClosed
	case <-ctx.Done():
		return ctx.Err()
	case p.slots <- struct{}{}:
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stopping {
		<-p.slots
		return ErrWriteAdmissionClosed
	}
	p.active++
	return nil
}
func (p *planAdmission) leave() {
	p.mu.Lock()
	defer p.mu.Unlock()
	<-p.slots
	p.active--
	if p.stopping && p.active == 0 {
		close(p.done)
	}
}
func (p *planAdmission) stopAdmission() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stopping {
		return
	}
	p.stopping = true
	close(p.stop)
	if p.active == 0 {
		close(p.done)
	}
}
func (p *planAdmission) wait(ctx context.Context) error {
	select {
	case <-p.done:
		return nil
	case <-ctx.Done():
		p.mu.Lock()
		defer p.mu.Unlock()
		return &transaction.ShutdownIncomplete{ActiveSends: p.active, RuntimePending: true, Cause: ctx.Err()}
	}
}

// StopAdmission leaves owner Checker/RPC lifecycle running. The owner first
// waits for these host operations, then closes its SDK, then its database pool.
func (u *PlannedUnitOfWork) StopAdmission()                 { u.admission.stopAdmission() }
func (u *PlannedUnitOfWork) Wait(ctx context.Context) error { return u.admission.wait(ctx) }

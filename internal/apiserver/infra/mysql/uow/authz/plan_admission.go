package authz

import (
	"context"
	"errors"
	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/operation"
	"github.com/FangcunMount/reliable-messaging/transaction"
	"sync"
)

// planAdmission also counts lookup/read-only planning before SDK admission.
// Closing the SDK alone cannot truthfully account for these host operations.
type planAdmission struct {
	mu         sync.Mutex
	queries    []*operation.QueryService
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
	for _, query := range p.queries {
		query.StopAdmission()
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
	default:
	}
	select {
	case <-p.done:
		return nil
	case <-ctx.Done():
		p.mu.Lock()
		defer p.mu.Unlock()
		if p.stopping && p.active == 0 {
			return nil
		}
		return &transaction.ShutdownIncomplete{ActiveSends: p.active, RuntimePending: true, Cause: ctx.Err()}
	}
}

// StopAdmission leaves owner Checker/RPC lifecycle running. The owner first
// waits for these host operations, then closes its SDK, then its database pool.
func (u *PlannedUnitOfWork) StopAdmission() { u.admission.stopAdmission() }
func (u *PlannedUnitOfWork) Wait(ctx context.Context) error {
	if err := u.admission.wait(ctx); err != nil {
		return err
	}
	u.admission.mu.Lock()
	queries := append([]*operation.QueryService(nil), u.admission.queries...)
	u.admission.mu.Unlock()
	var result error
	for _, query := range queries {
		result = errors.Join(result, query.Wait(ctx))
	}
	if result != nil {
		return &transaction.ShutdownIncomplete{RuntimePending: true, Cause: result}
	}
	return nil
}

// NewOperationQuery shares the writer's real shutdown accounting across the
// application UOW clone. Registration occurs during module composition only.
func (u *PlannedUnitOfWork) NewOperationQuery(access operation.AccessPolicy) (*operation.QueryService, error) {
	if u.config.Journal == nil {
		return nil, nil
	}
	u.admission.mu.Lock()
	defer u.admission.mu.Unlock()
	if u.admission.stopping || len(u.admission.queries) != 0 {
		return nil, operation.ErrQueryAdmissionClosed
	}
	query, err := operation.NewQueryService(u.config.Journal, u.config.Journal, access, operation.QueryConfig{Concurrency: u.config.Sends, Timeout: u.config.ProofBudget})
	if err != nil {
		return nil, err
	}
	u.admission.queries = append(u.admission.queries, query)
	return query, nil
}

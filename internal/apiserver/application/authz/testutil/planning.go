package testutil

import (
	"context"

	app "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/uow"
)

// PlanningProbe verifies host entry planning contracts on source fixtures. It
// has no SDK, half messages or witness and cannot prove a production transaction.
// Native MySQL tests separately enforce the physical read-only/root boundaries.
type PlanningProbe struct {
	Delegate      app.UnitOfWork
	Requests      []app.WriteRequest
	Snapshots     []app.WriteSnapshot
	Writes        int
	BeforePreview func()
}

func (p *PlanningProbe) WithinTx(ctx context.Context, fn func(context.Context, app.TxRepositories) error) error {
	return p.Delegate.WithinTx(ctx, fn)
}
func (p *PlanningProbe) ReadPlan(ctx context.Context, fn func(context.Context, app.TxRepositories) error) error {
	return p.Delegate.WithinTx(ctx, func(ctx context.Context, repos app.TxRepositories) error { repos.Events = nil; return fn(ctx, repos) })
}
func (p *PlanningProbe) WithinPlannedTx(ctx context.Context, request app.WriteRequest, fn func(context.Context, app.TxRepositories) error) error {
	p.Requests = append(p.Requests, request)
	if p.BeforePreview != nil {
		p.BeforePreview()
	}
	snapshot, err := app.PrepareChange(ctx, p, request)
	if err != nil {
		return err
	}
	p.Snapshots = append(p.Snapshots, snapshot)
	if !snapshot.Changed {
		return nil
	}
	p.Writes++
	return p.Delegate.WithinTx(ctx, fn)
}

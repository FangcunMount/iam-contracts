package uow

import (
	"context"
	"errors"

	change "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/policychange"
	policy "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/policy"
)

var ErrWritePlanRequired = errors.New("explicit authorization write plan required")

// WriteRequest is a host business plan, not an SDK callback or a retry permit.
// Preview runs only against an enforced read-only snapshot. It must recheck the
// original permission and business conditions; false means an authorized no-op.
type WriteRequest struct {
	Operation, Scope, ChangedBy string
	PermissionResource          string
	PermissionActions           []string
	Input                       any
	Preview                     func(context.Context, TxRepositories) (bool, error)
}
type WriteSnapshot struct {
	Expected int64
	Changed  bool
}
type ReadOnlyPlanner interface {
	ReadPlan(context.Context, func(context.Context, TxRepositories) error) error
}
type PlannedUnitOfWork interface {
	WithinPlannedTx(context.Context, WriteRequest, func(context.Context, TxRepositories) error) error
}

// WithinWrite preserves the original UOW until the host explicitly installs the
// planned implementation. A configured planned UOW never falls back on errors.
func WithinWrite(ctx context.Context, u UnitOfWork, request WriteRequest, fn func(context.Context, TxRepositories) error) error {
	if u == nil {
		return ErrWritePlanRequired
	}
	if planned, ok := u.(PlannedUnitOfWork); ok {
		return planned.WithinPlannedTx(ctx, request, fn)
	}
	return u.WithinTx(ctx, fn)
}
func PrepareChange(ctx context.Context, reader ReadOnlyPlanner, request WriteRequest) (WriteSnapshot, error) {
	var snapshot WriteSnapshot
	if ctx == nil || reader == nil || request.Operation == "" || request.Scope == "" || request.Preview == nil {
		return snapshot, change.ErrInvalidChangePlan
	}
	err := reader.ReadPlan(ctx, func(readCtx context.Context, repos TxRepositories) error {
		if repos.PolicyVersions == nil {
			return change.ErrInvalidChangePlan
		}
		current, err := repos.PolicyVersions.GetCurrent(readCtx)
		if err != nil {
			return err
		}
		if current == nil || current.Version < 1 {
			return change.ErrInvalidChangePlan
		}
		readCtx, err = change.WithReadVersion(readCtx, current.Version)
		if err != nil {
			return err
		}
		snapshot = WriteSnapshot{Expected: current.Version, Changed: true}
		if request.Preview != nil {
			snapshot.Changed, err = request.Preview(readCtx, repos)
		}
		return err
	})
	if err != nil {
		return WriteSnapshot{}, err
	}
	return snapshot, nil
}

// CheckExpectedVersion is also used before no-op detection for commands carrying
// an explicit policy-version precondition. Zero retains the legacy optional form.
func CheckExpectedVersion(current, expected int64) error {
	if expected < 0 {
		return policy.ErrInvalidVersionPlan
	}
	if expected != 0 && expected != current {
		return policy.ErrStaleVersion
	}
	return nil
}

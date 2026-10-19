package authz

import (
	"context"
	"database/sql"
	"errors"
	"time"

	operation "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/operation"
	change "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/policychange"
	app "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/uow"
	policy "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/policy"
	"github.com/FangcunMount/iam/v5/internal/apiserver/infra/messaging"
	journal "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/authzoperation"
	"github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/messageprotocol"
	database "github.com/FangcunMount/iam/v5/internal/pkg/database/mysql"
	"github.com/FangcunMount/reliable-messaging/transaction"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func originalSQLRoot(tx *gorm.DB) (*sql.Tx, error) {
	if tx == nil || tx.Statement == nil {
		return nil, transaction.ErrTransactionRequired
	}
	switch pool := tx.Statement.ConnPool.(type) {
	case *sql.Tx:
		if pool != nil {
			return pool, nil
		}
	case *gorm.PreparedStmtTX:
		if raw, ok := pool.Tx.(*sql.Tx); ok && raw != nil {
			return raw, nil
		}
	}
	return nil, transaction.ErrTransactionRequired
}
func (u *PlannedUnitOfWork) withBusinessOperation(ctx context.Context, command change.CommandIntent, request app.WriteRequest, fn func(context.Context, app.TxRepositories) error) error {
	if !request.Result.Valid() || request.Preview == nil || request.AuthorizeKnown == nil {
		return app.ErrWritePlanRequired
	}
	found, err := u.config.Journal.Lookup(ctx, command)
	if err == nil {
		return u.restoreBusinessOperation(ctx, found, request)
	}
	if !errors.Is(err, operation.ErrNotFound) {
		return err
	}
	// Pre-journal provisional SDK records cannot authorize another execution or
	// be silently promoted to a new typed business receipt.
	lookup := transaction.Request{Owner: "iam.mysql", Scope: change.PolicyScope, Key: command.Key(), InputHash: command.InputHash()}
	previous, err := u.original.store.Lookup(ctx, lookup)
	if err == nil {
		operation.ReportReference(ctx, previous.Execution.Plan.Input().OperationID)
		return u.known(previous)
	}
	if !errors.Is(err, transaction.ErrNotFound) {
		return err
	}
	planning, permissions, err := change.CapturePermissions(ctx)
	if err != nil {
		return err
	}
	var auditAt time.Time
	preview := request.Preview
	request.Preview = func(readCtx context.Context, repos app.TxRepositories) (bool, error) {
		changed, err := preview(readCtx, repos)
		if err != nil || !changed {
			return changed, err
		}
		tx, err := database.RequireTx(readCtx)
		if err != nil {
			return false, err
		}
		if err = tx.Statement.ConnPool.QueryRowContext(readCtx, "SELECT CURRENT_TIMESTAMP").Scan(&auditAt); err != nil {
			return false, err
		}
		if auditAt.IsZero() {
			return false, change.ErrInvalidChangePlan
		}
		return true, nil
	}
	snapshot, err := app.PrepareChange(planning, u.original, request)
	if err != nil {
		return err
	}
	required, err := permissions()
	if err != nil {
		return err
	}
	id := uuid.NewString()
	var frozen *transaction.FrozenPlan
	var noop [][]byte
	if snapshot.Changed {
		builder := change.NewArtifactBuilder(auditAt)
		if request.Freeze != nil {
			if err = request.Freeze(builder); err != nil {
				return err
			}
		}
		artifacts, err := builder.Freeze()
		if err != nil {
			return err
		}
		plan, err := change.FreezeVersionPlanWithArtifacts(command, snapshot.Expected, id, uuid.NewString(), time.Now().UTC(), u.config.LogicalTopic, u.config.PhysicalTopic, artifacts)
		if err != nil {
			return err
		}
		frozen = &plan
	} else {
		result, err := request.Result.Capture()
		if err != nil {
			return err
		}
		noop = [][]byte{result}
	}
	candidate, err := operation.NewCandidate(command, id, u.original.store.Binding(), snapshot.Expected, required, frozen, noop...)
	if err != nil {
		return err
	}
	reserved, err := u.config.Journal.Reserve(ctx, candidate, u.config.OperationLease)
	if err != nil {
		return err
	}
	if reserved.Claim == nil {
		return u.restoreBusinessOperation(ctx, reserved.Snapshot, request)
	}
	candidate = reserved.Snapshot.Candidate
	operation.ReportReference(ctx, candidate.ID())
	if !candidate.Changed() {
		return u.executeNoopOperation(ctx, reserved, request)
	}
	// The host winner owns the exact complete SDK plan. No new identity/wire is
	// generated after this point, including on a failed reservation or send.
	plan, err := candidate.Plan()
	if err != nil {
		return err
	}
	sdk, err := u.sender.Reserve(ctx, transaction.RequestFor(plan), plan)
	if err != nil {
		return u.resolveOperation(ctx, reserved, request.Result, err)
	}
	if !sdk.Acquired || sdk.Execution.Plan.Digest() != plan.Digest() {
		return u.resolveOperation(ctx, reserved, request.Result, operation.ErrProtocolConflict)
	}
	executor := u.original.LocalExecutor(command, func(active context.Context, repos app.TxRepositories) error {
		tx, err := database.RequireTx(active)
		if err != nil {
			return err
		}
		root, err := originalSQLRoot(tx)
		if err != nil {
			return err
		}
		guard, err := u.config.Journal.Bind(active, root, reserved.Claim)
		if err != nil {
			return err
		}
		if err = fn(active, repos); err != nil {
			return err
		}
		result, err := request.Result.Capture()
		if err != nil {
			return err
		}
		return guard.Commit(active, result, candidate.ExpectedVersion()+1)
	})
	_, completion, sendErr := messaging.ExecuteJoiningLocal(ctx, u.sender, sdk.Execution, executor)
	if completion.Started && completion.Outcome == transaction.LocalCommitted {
		return nil
	}
	cause := sendErr
	if completion.Err != nil {
		cause = completion.Err
	}
	return u.resolveOperation(ctx, reserved, request.Result, cause)
}
func (u *PlannedUnitOfWork) executeNoopOperation(ctx context.Context, reserved journal.Reservation, request app.WriteRequest) error {
	candidate := reserved.Snapshot.Candidate
	// Preview has already run in an enforced READ ONLY root. This receipt-only
	// root never reruns Preview/business. The policy row lock verifies that the
	// original version/authorization snapshot is unchanged before saving its result.
	err := u.original.base.within(ctx, func(active context.Context, _ app.TxRepositories) error {
		tx, err := database.RequireTx(active)
		if err != nil {
			return err
		}
		root, err := originalSQLRoot(tx)
		if err != nil {
			return err
		}
		guard, err := u.config.Journal.Bind(active, root, reserved.Claim)
		if err != nil {
			return err
		}
		if err = messageprotocol.RequireTransaction(active, tx, candidate.Binding(), "m3-1", true); err != nil {
			return err
		}
		var version int64
		if err = root.QueryRowContext(active, "SELECT policy_version FROM authz_policy_versions WHERE id=1 FOR UPDATE").Scan(&version); err != nil {
			return err
		}
		if version != candidate.ExpectedVersion() {
			return policy.ErrStaleVersion
		}
		return guard.Commit(active, candidate.NoopResult(), version)
	})
	if err == nil {
		return nil
	}
	return u.resolveOperation(ctx, reserved, request.Result, err)
}
func (u *PlannedUnitOfWork) resolveOperation(_ context.Context, reserved journal.Reservation, codec *operation.ResultCodec, cause error) error {
	// A durable abort competes with the same original-row lock. It may discover a
	// committed root instead. Cancellation or a returned error alone is no proof.
	proof, cancel := context.WithTimeout(context.Background(), u.config.ProofBudget)
	defer cancel()
	outcome, err := u.config.Journal.Abort(proof, reserved.Claim)
	if err == nil && outcome.State == journal.Committed {
		if err = codec.Restore(outcome.Result); err == nil {
			return nil
		}
	}
	local := transaction.LocalUnknown
	if err == nil && outcome.State == journal.Aborted {
		local = transaction.LocalAborted
	}
	return &operation.OutcomeError{OperationID: reserved.Snapshot.Candidate.ID(), Local: local, Cause: errors.Join(cause, err)}
}
func (u *PlannedUnitOfWork) restoreBusinessOperation(ctx context.Context, record journal.Snapshot, request app.WriteRequest) error {
	candidate := record.Candidate
	// Empty permissions are possible only for explicitly trusted maintenance or
	// privileged service contexts. Other service replay needs its own admission
	// contract; it cannot acquire authority from an old success alone.
	if len(candidate.Permissions()) == 0 && candidate.Principal() != "service:admin" && candidate.Principal() != u.config.MaintenancePrincipal {
		return operation.ErrAccessDenied
	}
	err := u.original.ReadPlan(ctx, func(active context.Context, repos app.TxRepositories) error {
		current, err := repos.PolicyVersions.GetCurrent(active)
		if err != nil {
			return err
		}
		if current == nil {
			return change.ErrInvalidChangePlan
		}
		checked, err := change.WithReadVersion(active, current.Version)
		if err != nil {
			return err
		}
		return request.AuthorizeKnown(checked, candidate.Permissions())
	})
	if err != nil {
		return err
	}
	operation.ReportReference(ctx, candidate.ID())
	switch record.State {
	case journal.Committed:
		return request.Result.Restore(record.Result)
	case journal.Aborted:
		return &operation.OutcomeError{OperationID: candidate.ID(), Local: transaction.LocalAborted}
	case journal.Prepared:
		if !candidate.Changed() {
			return &operation.OutcomeError{OperationID: candidate.ID(), Local: transaction.LocalUnknown}
		}
		protocol, err := u.config.Journal.ReadProtocol(ctx, candidate)
		if err == nil && protocol.Local == transaction.LocalCommitted {
			refreshed, err := u.config.Journal.Read(ctx, candidate.ID())
			if err == nil && refreshed.State == journal.Committed {
				return request.Result.Restore(refreshed.Result)
			}
		}
		local := transaction.LocalUnknown
		if err == nil && protocol.Local == transaction.LocalAborted {
			local = transaction.LocalAborted
		}
		return &operation.OutcomeError{OperationID: candidate.ID(), Local: local, Cause: err}
	default:
		return operation.ErrInvalid
	}
}

package authz

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	change "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/policychange"
	app "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/uow"
	assignment "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/assignment"
	"github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/messageprotocol"
	database "github.com/FangcunMount/iam/v5/internal/pkg/database/mysql"
	"github.com/FangcunMount/iam/v5/pkg/event"
	txmysql "github.com/FangcunMount/reliable-messaging/storage/transaction/mysql"
	"github.com/FangcunMount/reliable-messaging/transaction"
	"gorm.io/gorm"
)

// TransactionUnitOfWork borrows the host primary pool. It performs no I/O or DDL
// during construction; the host explicitly installs and validates its binding.
// A local executor may only be passed to Sender.Execute, after every half ACK.
type TransactionUnitOfWork struct {
	db    *gorm.DB
	store *txmysql.Store
	base  *unitOfWork
}

func NewTransactionUnitOfWork(db *gorm.DB, resolver assignment.SubjectResolver, binding transaction.Binding) (*TransactionUnitOfWork, error) {
	if binding.Owner != "iam.mysql" {
		return nil, transaction.ErrBinding
	}
	if db == nil || db.Error != nil || db.Statement == nil || db.Dialector == nil || db.Dialector.Name() != "mysql" {
		return nil, transaction.ErrTransactionRequired
	}
	pool := db.Statement.ConnPool
	if prepared, ok := pool.(*gorm.PreparedStmtDB); ok {
		pool = prepared.ConnPool
	}
	raw, ok := pool.(*sql.DB)
	if !ok || raw == nil {
		return nil, transaction.ErrTransactionRequired
	}
	store, err := txmysql.New(raw, binding)
	if err != nil {
		return nil, err
	}
	base := NewUnitOfWork(db, resolver, transactionEventStager{}).(*unitOfWork)
	return &TransactionUnitOfWork{db: db, store: store, base: base}, nil
}
func (u *TransactionUnitOfWork) Store() *txmysql.Store { return u.store }
func (u *TransactionUnitOfWork) ReadPlan(ctx context.Context, fn func(context.Context, app.TxRepositories) error) error {
	return u.base.ReadPlan(ctx, func(readCtx context.Context, repos app.TxRepositories) error {
		tx, err := database.RequireTx(readCtx)
		if err != nil {
			return err
		}
		if err = messageprotocol.RequireTransaction(readCtx, tx, u.store.Binding(), "m3-1", false); err != nil {
			return err
		}
		return fn(readCtx, repos)
	})
}

// LocalExecutor is single-use, including on errors. A nil business callback or a
// borrowed caller transaction is rejected before opening any local root. A root
// commit error is Unknown; returning nil from the callback is never the proof.
func (u *TransactionUnitOfWork) LocalExecutor(command change.CommandIntent, fn func(context.Context, app.TxRepositories) error) transaction.LocalExecutor {
	var used atomic.Bool
	return func(ctx context.Context, e transaction.Execution) (transaction.LocalOutcome, error) {
		if !used.CompareAndSwap(false, true) {
			return transaction.LocalUnknown, transaction.ErrStaleExecution
		}
		if u == nil || u.base == nil || fn == nil || ctx == nil {
			return transaction.LocalNotStarted, app.ErrWritePlanRequired
		}
		if _, active := database.TxFromContext(ctx); active {
			return transaction.LocalNotStarted, transaction.ErrTransactionRequired
		}
		if err := change.MatchCommand(e.Plan, command); err != nil {
			return transaction.LocalNotStarted, err
		}
		if !e.Valid() {
			return transaction.LocalNotStarted, transaction.ErrStaleExecution
		}
		base := u.base
		plan, planErr := change.DecodeVersionPlan(e.Plan)
		if planErr != nil {
			return transaction.LocalNotStarted, planErr
		}
		if auditAt, ok := plan.AuditTime(); ok {
			base = &unitOfWork{base: database.NewUnitOfWork(u.db.Session(&gorm.Session{NowFunc: func() time.Time { return auditAt }})), events: u.base.events, subjectResolver: u.base.subjectResolver}
		}
		var callbackErr error
		entered := false
		err := base.within(ctx, func(txCtx context.Context, repos app.TxRepositories) error {
			entered = true
			callbackErr = func() error {
				tx, err := database.RequireTx(txCtx)
				if err != nil {
					return err
				}
				// The guard's conditional write precedes permissions, facts and witness writes.
				guard, err := u.store.BindGORM(txCtx, tx, e)
				if err != nil {
					return err
				}
				if err = messageprotocol.RequireTransaction(txCtx, tx, u.store.Binding(), "m3-1", true); err != nil {
					return err
				}
				// A root cannot bypass Sender's all-halves gate. The exact
				// identities are already locked and verified by BindGORM.
				key := transaction.ExecutionKey(e.Plan.Input().Owner, e.Plan.Input().OperationID, e.WriterEpoch, e.Fence)
				var prepared int64
				if err = tx.WithContext(txCtx).Table("rm_tx_witness w").Joins("JOIN rm_tx_execution e ON e.execution_key=w.execution_key").Where("w.execution_key=? AND e.prepare_started=TRUE AND w.state='prepared' AND w.half_outcome='prepared'", key[:]).Count(&prepared).Error; err != nil {
					return err
				}
				if prepared != int64(len(e.References())) {
					return change.ErrInvalidChangePlan
				}
				txCtx, err = change.WithExecutionPlan(txCtx, e, command)
				if err != nil {
					return err
				}
				plan, _ := change.ExecutionPlan(txCtx)
				proof := &eventProof{plan: plan, active: true}
				defer proof.close()
				txCtx = context.WithValue(txCtx, eventProofKey{}, proof)
				// The original root is owned by this executor. Maintenance SQL
				// borrows it directly, so its hooks must inherit the same frozen context.
				tx.Statement.Context = txCtx
				if err = fn(txCtx, repos); err != nil {
					return err
				}
				if !proof.exactlyOnce() {
					return change.ErrInvalidChangePlan
				}
				current, err := repos.PolicyVersions.GetCurrent(txCtx)
				if err != nil {
					return err
				}
				if current == nil || current.Version != plan.Target {
					return change.ErrInvalidChangePlan
				}
				return guard.CommitWitness(txCtx)
			}()
			return callbackErr
		})
		if err == nil {
			return transaction.LocalCommitted, nil
		} // actual host root commit ACK
		if !entered {
			return transaction.LocalNotStarted, err
		}
		if callbackErr != nil {
			return transaction.LocalAborted, callbackErr
		} // no COMMIT was issued; SDK still needs a durable abort fence
		return transaction.LocalUnknown, err // callback ended; COMMIT may have reached the server
	}
}

type eventProofKey struct{}
type eventProof struct {
	mu     sync.Mutex
	plan   change.VersionPlan
	count  int
	active bool
}

func (p *eventProof) close() { p.mu.Lock(); p.active = false; p.mu.Unlock() }
func (p *eventProof) exactlyOnce() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.active && p.count == 1
}

type transactionEventStager struct{}

func (transactionEventStager) Stage(ctx context.Context, events ...event.DomainEvent) error {
	if ctx == nil {
		return transaction.ErrTransactionRequired
	}
	p, _ := ctx.Value(eventProofKey{}).(*eventProof)
	if p == nil {
		return transaction.ErrTransactionRequired
	}
	if _, err := database.RequireTx(ctx); err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.active || p.count != 0 || len(events) != 1 || events[0] == nil {
		return change.ErrInvalidChangePlan
	}
	e := events[0]
	in := p.plan.Message.Input()
	at, err := time.Parse(time.RFC3339Nano, in.OccurredAt)
	if err != nil {
		return change.ErrInvalidChangePlan
	}
	body, err := json.Marshal(e.Payload())
	if err != nil || e.EventID() != in.ID || e.EventType() != in.EventType || !e.OccurredAt().Equal(at) || e.AggregateType() != "PolicyVersion" || e.AggregateID() != strconv.FormatInt(p.plan.Target, 10) || !bytes.Equal(body, p.plan.EventPayload()) {
		return change.ErrInvalidChangePlan
	}
	p.count++
	return nil
}

var _ event.Stager = transactionEventStager{}
var _ app.ReadOnlyPlanner = (*TransactionUnitOfWork)(nil)

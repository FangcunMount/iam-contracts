// Package policywrite declares the original-transaction boundary of offline
// authorization tools. It owns neither a Broker client nor a database pool.
package policywrite

import (
	"context"
	"errors"
	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/operation"

	change "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/policychange"
	app "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/uow"
	database "github.com/FangcunMount/iam/v5/internal/pkg/database/mysql"
	"github.com/FangcunMount/iam/v5/pkg/event"
	"gorm.io/gorm"
)

// Request contains only reviewed business inputs; passwords and before-images
// must never be included. Preview revalidates the original maintenance contract
// in a fresh read-only root and returns false for an already completed operation.
type Request struct {
	Operation, ChangedBy string
	Input                any
	Result               *operation.ResultCodec
	Preview              func(context.Context, *gorm.DB) (bool, error)
	Freeze               func(*change.ArtifactBuilder) error
}

// Writer is installed explicitly by the host's transaction composition. Its
// failure cannot silently fall back to the legacy Outbox transaction.
type Writer interface {
	WithinPolicyWrite(context.Context, *gorm.DB, Request, func(context.Context) error) error
}

func Within(ctx context.Context, db *gorm.DB, stager event.Stager, request Request, fn func(context.Context) error) error {
	if ctx == nil || db == nil || stager == nil || request.Operation == "" || request.Preview == nil || fn == nil {
		return errors.New("maintenance policy write requires its explicit plan and original database")
	}
	if writer, ok := stager.(Writer); ok {
		return writer.WithinPolicyWrite(ctx, db, request, fn)
	}
	return database.NewUnitOfWork(db).WithinTransaction(ctx, fn)
}

// AuthorizationRequest borrows the root supplied by the host read-only planner;
// it does not open a replacement connection or a nested transaction.
func (r Request) AuthorizationRequest() app.WriteRequest {
	return app.WriteRequest{Result: r.Result, AuthorizeKnown: func(context.Context, []operation.Permission) error { return nil }, Freeze: r.Freeze, Operation: r.Operation, Scope: "global", ChangedBy: r.ChangedBy, Input: r.Input, Preview: func(ctx context.Context, _ app.TxRepositories) (bool, error) {
		tx, err := database.RequireTx(ctx)
		if err != nil {
			return false, err
		}
		if r.Preview == nil {
			return false, app.ErrWritePlanRequired
		}
		return r.Preview(ctx, tx)
	}}
}

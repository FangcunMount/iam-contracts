// Package authzaudit applies the host plan's immutable original-transaction
// audit values. It never allocates an ID or clock for planned writes.
package authzaudit

import (
	"context"
	"time"

	change "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/policychange"
	base "github.com/FangcunMount/iam/v5/internal/pkg/database/mysql"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
)

func Create(ctx context.Context, key string, fields *base.AuditFields) (bool, error) {
	plan, planned := change.ExecutionPlan(ctx)
	if !planned {
		return false, nil
	}
	id, ok := plan.CreatedID(key)
	if !ok || fields == nil {
		return true, change.ErrInvalidChangePlan
	}
	at, ok := plan.AuditTime()
	if !ok {
		return true, change.ErrInvalidChangePlan
	}
	actor := base.UserIDOrZero(ctx)
	fields.ID = id
	fields.CreatedAt = at
	fields.UpdatedAt = at
	fields.CreatedBy = actor
	fields.UpdatedBy = actor
	fields.DeletedBy = 0
	fields.Version = base.InitialVersion
	return true, nil
}
func Update(ctx context.Context, fields *base.AuditFields) (bool, error) {
	plan, planned := change.ExecutionPlan(ctx)
	if !planned {
		return false, nil
	}
	at, ok := plan.AuditTime()
	if !ok || fields == nil {
		return true, change.ErrInvalidChangePlan
	}
	fields.UpdatedAt = at
	fields.UpdatedBy = base.UserIDOrZero(ctx)
	return true, nil
}
func Time(ctx context.Context) (time.Time, error) {
	if plan, ok := change.ExecutionPlan(ctx); ok {
		at, valid := plan.AuditTime()
		if !valid {
			return time.Time{}, change.ErrInvalidChangePlan
		}
		return at, nil
	}
	return time.Now(), nil
}
func ID(ctx context.Context, key string) (meta.ID, error) {
	if plan, ok := change.ExecutionPlan(ctx); ok {
		id, valid := plan.CreatedID(key)
		if !valid {
			return 0, change.ErrInvalidChangePlan
		}
		return id, nil
	}
	return meta.New(), nil
}

package authz

import (
	"context"

	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/management"
	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/operation"
	change "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/policychange"
	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/uow"
)

// The current REST authorization writers operate in global scope. An org claim
// is not membership/permission proof and cannot open a scoped gRPC receipt here.
type operationAccess struct {
	reader uow.ReadOnlyPlanner
	guard  management.Guard
}

func (a operationAccess) Principal(ctx context.Context) (string, bool) {
	return management.AuthenticatedPrincipal(ctx)
}
func (a operationAccess) Require(ctx context.Context, scope string, permissions []operation.Permission) error {
	if a.reader == nil || scope != "global" || len(permissions) == 0 {
		return operation.ErrAccessDenied
	}
	return a.reader.ReadPlan(ctx, func(active context.Context, repos uow.TxRepositories) error {
		if repos.PolicyVersions == nil {
			return operation.ErrAccessDenied
		}
		version, err := repos.PolicyVersions.GetCurrent(active)
		if err != nil {
			return err
		}
		if version == nil {
			return operation.ErrAccessDenied
		}
		checked, err := change.WithReadVersion(active, version.Version)
		if err != nil {
			return err
		}
		for _, permission := range permissions {
			if err := a.guard.RequireOperation(checked, permission.Resource, permission.Action); err != nil {
				// Runtime policy availability errors remain failures; never use a
				// stale allow or expose driver/authorization internals in transport.
				return operation.ErrAccessDenied
			}
		}
		return nil
	})
}

// NewOperationAccessPolicy performs no I/O. QueryService owns the bounded
// request lifetime; this policy borrows the host's primary read boundary.
func NewOperationAccessPolicy(reader uow.ReadOnlyPlanner, guard management.Guard) operation.AccessPolicy {
	return operationAccess{reader: reader, guard: guard}
}

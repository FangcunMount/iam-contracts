package authz

import (
	"context"
	"testing"

	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/management"
	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/operation"
	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/uow"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/authorization"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/policy"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/subject"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"github.com/stretchr/testify/require"
)

type operationAccessReader struct {
	calls   int
	version int64
}

func (r *operationAccessReader) ReadPlan(ctx context.Context, fn func(context.Context, uow.TxRepositories) error) error {
	r.calls++
	return fn(ctx, uow.TxRepositories{PolicyVersions: operationVersion{version: r.version}})
}

type operationVersion struct {
	policy.Repository
	version int64
}

func (r operationVersion) GetCurrent(context.Context) (*policy.PolicyVersion, error) {
	return &policy.PolicyVersion{Version: r.version}, nil
}

type operationChecker func(context.Context, authorization.Request) (authorization.Decision, error)

func (f operationChecker) Check(ctx context.Context, r authorization.Request) (authorization.Decision, error) {
	return f(ctx, r)
}

func TestOperationAccessUsesFreshPrimaryVersionAndEveryOriginalPermission(t *testing.T) {
	user, err := subject.NewUserRef(meta.ID(7))
	require.NoError(t, err)
	ctx := management.WithAuthenticatedUser(t.Context(), user)
	permissions := []operation.Permission{{Resource: "iam:authz:collection:roles", Action: "update"}, {Resource: "iam:authz:management:protected-roles", Action: "manage_protected"}}
	for _, tc := range []struct {
		allowed bool
		cached  int64
		pass    bool
	}{{true, 42, true}, {true, 41, false}, {false, 42, false}} {
		reader := &operationAccessReader{version: 42}
		calls := 0
		guard := management.NewGuard(operationChecker(func(context.Context, authorization.Request) (authorization.Decision, error) {
			calls++
			return authorization.Decision{Allowed: tc.allowed, PolicyVersion: tc.cached}, nil
		}))
		access := operationAccess{reader: reader, guard: guard}
		if tc.pass {
			require.NoError(t, access.Require(ctx, "global", permissions))
			require.Equal(t, 2, calls)
		} else {
			require.ErrorIs(t, access.Require(ctx, "global", permissions), operation.ErrAccessDenied)
		}
		require.Equal(t, 1, reader.calls)
		require.ErrorIs(t, access.Require(ctx, "org:7", permissions), operation.ErrAccessDenied)
		require.Equal(t, 1, reader.calls, "scope mismatch queried DB")
		require.ErrorIs(t, access.Require(ctx, "global", nil), operation.ErrAccessDenied)
	}
}

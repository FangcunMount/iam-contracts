package management

import (
	"context"
	"testing"
	"time"

	change "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/policychange"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/authorization"
	policy "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/policy"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/role"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/subject"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"github.com/FangcunMount/reliable-messaging/transaction"
	"github.com/stretchr/testify/require"
)

func TestFrozenWriteRejectsStaleCachedAllowForOperationAndProtectedRole(t *testing.T) {
	user, err := subject.NewUserRef(meta.FromUint64(7))
	require.NoError(t, err)
	original := WithAuthenticatedUser(context.Background(), user)
	intent, err := change.NewCommandIntent("role.update", user.String(), "global", "key", map[string]string{"id": "42"})
	require.NoError(t, err)
	plan, err := change.FreezeVersionPlan(intent, 41, "op", "event", time.Now(), "logical", "physical")
	require.NoError(t, err)
	ctx, err := change.WithExecutionPlan(original, transaction.Execution{Plan: plan, Fence: 1, WriterEpoch: "m3-1", Token: [32]byte{1}, LeaseUntil: time.Now().Add(time.Minute)}, intent)
	require.NoError(t, err)
	for _, version := range []int64{40, 41, 42} {
		guard := NewGuard(checkerFunc(func(context.Context, authorization.Request) (authorization.Decision, error) {
			return authorization.Decision{Allowed: true, PolicyVersion: version}, nil
		}))
		operation := guard.RequireOperation(ctx, "iam:authz:collection:roles", "update")
		protection := guard.Require(ctx, &role.Role{ManagementProtection: role.ManagementProtected})
		if version == 41 {
			require.NoError(t, operation)
			require.NoError(t, protection)
		} else {
			require.ErrorIs(t, operation, policy.ErrStaleVersion)
			require.ErrorIs(t, protection, policy.ErrStaleVersion)
		}
		require.NoError(t, guard.RequireOperation(original, "iam:authz:collection:roles", "update"), "legacy context policy freshness is not redefined by this helper")
	}
	owner, ok := AuthenticatedPrincipal(ctx)
	require.True(t, ok)
	require.Equal(t, "user:7", owner)
	owner, ok = AuthenticatedPrincipal(WithAuthenticatedService(context.Background(), "qs-apiserver.svc"))
	require.True(t, ok)
	require.Equal(t, "service:qs-apiserver.svc", owner)
	_, ok = AuthenticatedPrincipal(context.WithValue(context.Background(), struct{}{}, "user:7"))
	require.False(t, ok)
}

func TestOperationStatusPolicyCapturesOnlyAuthorizedCurrentPermissions(t *testing.T) {
	user, err := subject.NewUserRef(meta.ID(7))
	require.NoError(t, err)
	parent := WithAuthenticatedUser(t.Context(), user)
	for _, allowed := range []bool{false, true} {
		ctx, read, err := change.CapturePermissions(parent)
		require.NoError(t, err)
		ctx, err = change.WithReadVersion(ctx, 2)
		require.NoError(t, err)
		guard := NewGuard(checkerFunc(func(context.Context, authorization.Request) (authorization.Decision, error) {
			return authorization.Decision{Allowed: allowed, PolicyVersion: 2}, nil
		}))
		opErr := guard.RequireOperation(ctx, "iam:authz:collection:roles", "update")
		protectedErr := guard.Require(ctx, &role.Role{ManagementProtection: role.ManagementProtected})
		permissions, err := read()
		require.NoError(t, err)
		if allowed {
			require.NoError(t, opErr)
			require.NoError(t, protectedErr)
			require.ElementsMatch(t, []change.Permission{{Resource: "iam:authz:collection:roles", Action: "update"}, {Resource: role.ManageProtectedResource, Action: role.ManageProtectedAction}}, permissions)
		} else {
			require.Error(t, opErr)
			require.Error(t, protectedErr)
			require.Empty(t, permissions)
		}
	}
	ctx, read, err := change.CapturePermissions(parent)
	require.NoError(t, err)
	ctx, err = change.WithReadVersion(ctx, 2)
	require.NoError(t, err)
	stale := NewGuard(checkerFunc(func(context.Context, authorization.Request) (authorization.Decision, error) {
		return authorization.Decision{Allowed: true, PolicyVersion: 1}, nil
	}))
	require.ErrorIs(t, stale.RequireOperation(ctx, "iam:authz:collection:roles", "update"), policy.ErrStaleVersion)
	captured, err := read()
	require.NoError(t, err)
	require.Empty(t, captured)
}

package management

import (
	"context"
	admission "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/assignmentadmission"
	"testing"

	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/authorization"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/role"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/subject"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"github.com/stretchr/testify/require"
)

type checkerFunc func(context.Context, authorization.Request) (authorization.Decision, error)

func (f checkerFunc) Check(ctx context.Context, r authorization.Request) (authorization.Decision, error) {
	return f(ctx, r)
}

func TestGuardUsesAuthenticatedIdentityAndExplicitPermission(t *testing.T) {
	protected := &role.Role{ManagementProtection: role.ManagementProtected}
	standard := &role.Role{ManagementProtection: role.ManagementStandard}
	g := NewGuard(nil)
	require.NoError(t, g.Require(context.Background(), standard))
	require.Error(t, g.Require(context.Background(), protected))
	require.Error(t, g.Require(WithAuthenticatedService(context.Background(), "qs-apiserver.svc"), protected))
	require.NoError(t, g.Require(WithAuthenticatedService(context.Background(), "admin"), protected))
	sub, err := subject.NewUserRef(meta.ID(42))
	require.NoError(t, err)
	ctx := WithAuthenticatedUser(context.Background(), sub)
	for _, allowed := range []bool{false, true} {
		called := false
		g = NewGuard(checkerFunc(func(_ context.Context, r authorization.Request) (authorization.Decision, error) {
			called = true
			require.Equal(t, sub, r.Subject)
			require.Equal(t, role.ManageProtectedResource, r.ResourceKey.String())
			require.Equal(t, role.ManageProtectedAction, r.Action.String())
			return authorization.Decision{Allowed: allowed}, nil
		}))
		err := g.Require(ctx, protected)
		if allowed {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
		}
		require.True(t, called)
	}
}

func TestGuardValidatesAllRolesAfterProtectedRole(t *testing.T) {
	ctx := WithAuthenticatedService(context.Background(), "admin")
	g := NewGuard(nil)
	protected := &role.Role{ManagementProtection: role.ManagementProtected}
	require.Error(t, g.Require(ctx, protected, nil))
	require.Error(t, g.Require(ctx, protected, &role.Role{ManagementProtection: "invalid"}))
}

func TestProtectedPermissionDoesNotReplaceOriginalOperationPermission(t *testing.T) {
	sub, err := subject.NewUserRef(meta.ID(42))
	require.NoError(t, err)
	ctx := WithAuthenticatedUser(context.Background(), sub)
	g := NewGuard(checkerFunc(func(_ context.Context, r authorization.Request) (authorization.Decision, error) {
		return authorization.Decision{Allowed: r.Action.String() == role.ManageProtectedAction}, nil
	}))
	require.NoError(t, g.Require(ctx, &role.Role{ManagementProtection: role.ManagementProtected}))
	require.Error(t, g.RequireOperation(ctx, role.ManageProtectedResource, "delete"))
	require.Error(t, g.RequireOperation(context.Background(), role.ManageProtectedResource, "delete"))
}

func TestAssignmentGuardRebuildsServiceManagedSet(t *testing.T) {
	policy, err := admission.New(admission.Config{DefaultPolicy: "deny", Services: map[string]admission.ServiceConstraint{
		"qs-apiserver.svc": {SubjectTypes: []string{"user"}, Roles: []string{"qs:evaluator"}, RequireDelegatedActorOnGrant: true},
	}})
	require.NoError(t, err)
	g := NewGuard(nil, policy)
	sub, err := subject.NewUserRef(meta.ID(42))
	require.NoError(t, err)
	ctx := WithAuthenticatedService(context.Background(), "qs-apiserver.svc")
	require.NoError(t, g.RequireReplacement(ctx, sub, []string{"qs:evaluator"}, []string{"qs:evaluator"}, "user:9"))
	require.Error(t, g.RequireReplacement(ctx, sub, nil, []string{"qs:evaluator", "iam_admin"}, "user:9"))
	require.Error(t, g.RequireReplacement(ctx, sub, nil, nil, "user:9"))
	name, err := role.NewName("iam_admin")
	require.NoError(t, err)
	require.Error(t, g.RequireAssignment(ctx, sub, &role.Role{Name: name, ManagementProtection: role.ManagementStandard}, admission.OperationGrant, "user:9"))
	require.Error(t, g.RequireReplacement(WithAuthenticatedService(context.Background(), "unknown"), sub, nil, []string{"qs:evaluator"}, "user:9"))
}

package management

import (
	"context"
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

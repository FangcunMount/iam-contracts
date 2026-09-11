package role_test

import (
	"context"
	"testing"

	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/management"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	roleApp "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/role"
	authztestutil "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/testutil"
	permissiongrantDomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/permissiongrant"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/resource"
	roleDomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/role"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
	"github.com/stretchr/testify/require"
)

func TestDeleteRoleRejectsRoleWithActiveGrant(t *testing.T) {
	fixture := authztestutil.NewFixture(t, nil)
	roles := fixture.Roles
	grants := fixture.PermissionGrants
	role, err := roleDomain.NewRole("qs:evaluator", "Evaluator")
	require.NoError(t, err)
	require.NoError(t, roles.Create(management.WithAuthenticatedService(context.Background(), "admin"), &role))
	grant, err := permissiongrantDomain.New(
		role.ID, resource.NewResourceID(91), "qs:evaluation:collection:assessments", "retry", "operator",
	)
	require.NoError(t, err)
	require.NoError(t, grants.Create(management.WithAuthenticatedService(context.Background(), "admin"), &grant))
	catalog := roleApp.NewRoleCatalog(fixture.UnitOfWork, nil, management.NewGuard(nil))

	err = catalog.DeleteRole(management.WithAuthenticatedService(context.Background(), "admin"), roleApp.DeleteRoleCommand{ID: role.ID, ChangedBy: "operator"})

	require.Error(t, err)
	require.True(t, perrors.IsCode(err, code.ErrRoleInUse))
	persisted, err := roles.FindByID(management.WithAuthenticatedService(context.Background(), "admin"), role.ID)
	require.NoError(t, err)
	require.NotNil(t, persisted)
}

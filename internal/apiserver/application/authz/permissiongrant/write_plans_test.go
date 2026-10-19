package permissiongrant_test

import (
	"context"
	"testing"

	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/management"
	app "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/permissiongrant"
	fixture "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/testutil"
	uow "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/uow"
	"github.com/stretchr/testify/require"
)

func TestGrantPlansPreserveHistoricalRevocationNoopWithoutRestoringConditions(t *testing.T) {
	f, _, events := setupPermissionGrantService(t)
	ctx := management.WithAuthenticatedService(context.Background(), "admin")
	require.NoError(t, f.UnitOfWork.WithinTx(ctx, func(ctx context.Context, repos uow.TxRepositories) error {
		_, e := repos.PolicyVersions.GetOrCreate(ctx)
		return e
	}))
	role := seedRole(t, f.Roles, "qs:planned")
	resource := seedResource(t, f.Resources)
	probe := &fixture.PlanningProbe{Delegate: f.UnitOfWork}
	service := app.NewService(probe, f.PermissionGrants, nil, management.NewGuard(nil))
	grant, err := service.Create(ctx, app.CreateCommand{RoleID: role.ID, ResourceID: resource.ID, Action: "retry", GrantedBy: "operator"})
	require.NoError(t, err)
	require.Equal(t, "grant.create", probe.Requests[0].Operation)
	require.Len(t, events.events, 1)
	_, err = service.Create(ctx, app.CreateCommand{RoleID: role.ID, ResourceID: resource.ID, Action: "retry", GrantedBy: "operator"})
	require.Error(t, err)
	require.Equal(t, 1, probe.Writes)
	require.NoError(t, service.Revoke(ctx, app.RevokeCommand{GrantID: grant.ID, RevokedBy: "operator"}))
	require.Len(t, events.events, 2)
	f.SetHistoricalGrantConditions(t, grant.ID.Uint64(), `{"version":1,"all_of":[{"key":"object.origin_type","operator":"eq","value":{"type":"string","string":"adhoc"}}]}`)
	_, err = f.PermissionGrants.FindByID(ctx, grant.ID)
	require.Error(t, err)
	require.NoError(t, service.Revoke(ctx, app.RevokeCommand{GrantID: grant.ID, RevokedBy: "operator"}))
	require.Equal(t, 2, probe.Writes)
	require.Len(t, events.events, 2)
	require.False(t, probe.Snapshots[len(probe.Snapshots)-1].Changed)
}

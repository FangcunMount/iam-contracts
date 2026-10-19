package role_test

import (
	"context"
	"testing"

	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/management"
	app "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/role"
	fixture "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/testutil"
	uow "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/uow"
	"github.com/FangcunMount/iam/v5/pkg/event"
	"github.com/stretchr/testify/require"
)

type plannedRoleEvents struct{ events []event.DomainEvent }

func (p *plannedRoleEvents) Stage(_ context.Context, events ...event.DomainEvent) error {
	p.events = append(p.events, events...)
	return nil
}
func TestRoleEntriesDeclarePlansAndNoopDoesNotStageAVersion(t *testing.T) {
	events := new(plannedRoleEvents)
	f := fixture.NewFixture(t, events)
	ctx := management.WithAuthenticatedService(context.Background(), "admin")
	require.NoError(t, f.UnitOfWork.WithinTx(ctx, func(ctx context.Context, repos uow.TxRepositories) error {
		_, e := repos.PolicyVersions.GetOrCreate(ctx)
		return e
	}))
	probe := &fixture.PlanningProbe{Delegate: f.UnitOfWork}
	service := app.NewRoleCatalog(probe, nil, management.NewGuard(nil))
	command, err := app.NewCreateRoleCommand("qs:planned-role", "Original name", "original description")
	require.NoError(t, err)
	command.ChangedBy = "operator"
	created, err := service.CreateRole(ctx, command)
	require.NoError(t, err)
	require.Equal(t, "role.create", probe.Requests[0].Operation)
	require.Equal(t, "global", probe.Requests[0].Scope)
	require.Equal(t, 1, probe.Writes)
	require.Len(t, events.events, 1)
	same := "Original name"
	description := "original description"
	probe.BeforePreview = func() { same = "external mutation"; description = "external description" }
	result, err := service.UpdateRole(ctx, app.UpdateRoleCommand{ID: created.ID, ChangedBy: "operator", DisplayName: &same, Description: &description})
	require.NoError(t, err)
	require.Equal(t, "Original name", result.DisplayName)
	require.Equal(t, "original description", result.Description)
	require.False(t, probe.Snapshots[1].Changed)
	require.Equal(t, int64(2), probe.Snapshots[1].Expected)
	require.Equal(t, 1, probe.Writes)
	require.Len(t, events.events, 1, "no-op manufactured an event")
	probe.BeforePreview = nil
	renamed := "New name"
	_, err = service.UpdateRole(ctx, app.UpdateRoleCommand{ID: created.ID, ChangedBy: "operator", DisplayName: &renamed})
	require.NoError(t, err)
	require.Equal(t, 2, probe.Writes)
	require.Len(t, events.events, 2)
	require.NoError(t, service.DeleteRole(ctx, app.DeleteRoleCommand{ID: created.ID, ChangedBy: "operator"}))
	require.Equal(t, "role.delete", probe.Requests[3].Operation)
	require.Equal(t, 3, probe.Writes)
	require.Len(t, events.events, 3)
}

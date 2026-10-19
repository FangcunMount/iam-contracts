package resource_test

import (
	"context"
	"testing"

	app "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/resource"
	fixture "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/testutil"
	uow "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/uow"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/subject"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"github.com/stretchr/testify/require"
)

func TestResourceEntriesPlanNoopAndOwnCallerPointersAndActions(t *testing.T) {
	events := new(recordingStager)
	f := fixture.NewFixture(t, events)
	ctx := context.Background()
	require.NoError(t, f.UnitOfWork.WithinTx(ctx, func(ctx context.Context, repos uow.TxRepositories) error {
		_, e := repos.PolicyVersions.GetOrCreate(ctx)
		return e
	}))
	probe := &fixture.PlanningProbe{Delegate: f.UnitOfWork}
	service := app.NewResourceCatalog(probe, nil, allowCatalogWriter{})
	actor, err := subject.NewUserRef(meta.FromUint64(7))
	require.NoError(t, err)
	command, err := app.NewCreateResourceCommand("qs:test:collection:planned", "Original", "", "", "", []string{"read", "update"}, "Original description")
	require.NoError(t, err)
	command.Actor = actor
	command.ChangedBy = "operator"
	created, err := service.CreateResource(ctx, command)
	require.NoError(t, err)
	require.Equal(t, "resource.create", probe.Requests[0].Operation)
	require.Len(t, events.events, 1)
	same := "Original"
	description := "Original description"
	actions := []string{"read", "update"}
	probe.BeforePreview = func() { same = "external"; description = "external"; actions[0] = "delete" }
	value, err := service.UpdateResource(ctx, app.UpdateResourceCommand{Actor: actor, ChangedBy: "operator", ID: created.ID, DisplayName: &same, Description: &description, Actions: actions})
	require.NoError(t, err)
	require.Equal(t, "Original", value.DisplayName)
	require.Equal(t, "Original description", value.Description)
	require.Equal(t, []string{"read", "update"}, value.ActionStrings())
	require.False(t, probe.Snapshots[1].Changed)
	require.Equal(t, 1, probe.Writes)
	require.Len(t, events.events, 1)
	probe.BeforePreview = nil
	require.NoError(t, service.DeleteResource(ctx, app.DeleteResourceCommand{Actor: actor, ChangedBy: "operator", ID: created.ID}))
	require.Equal(t, "resource.delete", probe.Requests[2].Operation)
	require.Equal(t, 2, probe.Writes)
	require.Len(t, events.events, 2)
}

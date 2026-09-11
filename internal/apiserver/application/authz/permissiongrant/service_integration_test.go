package permissiongrant_test

import (
	"context"
	"testing"

	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/management"

	permissionGrantApp "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/permissiongrant"
	authztestutil "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/testutil"
	permissiongrantDomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/permissiongrant"
	resourceDomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/resource"
	roleDomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/role"
	"github.com/FangcunMount/iam/v5/pkg/event"
	"github.com/stretchr/testify/require"
)

func TestPermissionGrantRevokeAlreadyRevokedIsIdempotentWithoutVersionBump(t *testing.T) {
	fixture, _, stager := setupPermissionGrantService(t)
	reloader := &recordingReloader{}
	service := permissionGrantApp.NewService(fixture.UnitOfWork, fixture.PermissionGrants, reloader, management.NewGuard(nil))
	role := seedRole(t, fixture.Roles, "qs:evaluator")
	resource := seedResource(t, fixture.Resources)
	grant := seedGrant(t, fixture.PermissionGrants, role, resource)

	require.NoError(t, service.Revoke(management.WithAuthenticatedService(context.Background(), "admin"), permissionGrantApp.RevokeCommand{
		GrantID: grant.ID, RevokedBy: "operator-1",
	}))
	require.Len(t, stager.events, 1)
	require.EqualValues(t, 1, fixture.PolicyVersionCount(t))

	require.NoError(t, service.Revoke(management.WithAuthenticatedService(context.Background(), "admin"), permissionGrantApp.RevokeCommand{
		GrantID: grant.ID, RevokedBy: "operator-1",
	}))
	require.Len(t, stager.events, 1, "duplicate revoke must not publish another policy version")
	require.EqualValues(t, 1, fixture.PolicyVersionCount(t))
	require.Equal(t, 1, reloader.calls, "duplicate revoke must not reload an unchanged runtime policy")
}

func setupPermissionGrantService(t *testing.T) (*authztestutil.Fixture, *permissionGrantApp.Service, *recordingStager) {
	t.Helper()
	recording := &recordingStager{}
	fixture := authztestutil.NewFixture(t, recording)
	service := permissionGrantApp.NewService(fixture.UnitOfWork, fixture.PermissionGrants, nil, management.NewGuard(nil))
	return fixture, service, recording
}

func seedRole(t *testing.T, repository roleDomain.Repository, name string) roleDomain.Role {
	t.Helper()
	role, err := roleDomain.NewRole(name, name)
	require.NoError(t, err)
	require.NoError(t, repository.Create(management.WithAuthenticatedService(context.Background(), "admin"), &role))
	return role
}

func seedResource(t *testing.T, repository resourceDomain.Repository) resourceDomain.Resource {
	t.Helper()
	resource, err := resourceDomain.NewResource(
		"qs:evaluation:collection:assessments",
		[]string{"retry"},
		resourceDomain.WithDisplayName("Assessments"),
	)
	require.NoError(t, err)
	require.NoError(t, repository.Create(management.WithAuthenticatedService(context.Background(), "admin"), &resource))
	return resource
}

func seedGrant(
	t *testing.T,
	repository permissiongrantDomain.Repository,
	role roleDomain.Role,
	resource resourceDomain.Resource,

) permissiongrantDomain.Grant {
	t.Helper()
	grant, err := permissiongrantDomain.New(
		role.ID, resource.ID, resource.KeyString(), "retry", "operator-1",
	)
	require.NoError(t, err)
	require.NoError(t, repository.Create(management.WithAuthenticatedService(context.Background(), "admin"), &grant))
	return grant
}

type recordingStager struct{ events []event.DomainEvent }

func (s *recordingStager) Stage(_ context.Context, events ...event.DomainEvent) error {
	s.events = append(s.events, events...)
	return nil
}

type recordingReloader struct{ calls int }

func (r *recordingReloader) LoadPolicy(context.Context) error {
	r.calls++
	return nil
}

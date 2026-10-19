package authz_test

import (
	"context"
	"testing"

	app "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/assignment"
	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/management"
	change "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/policychange"
	fixture "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/testutil"
	uow "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/uow"
	domain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/assignment"
	policy "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/policy"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/role"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/scope"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/subject"
	"github.com/FangcunMount/iam/v5/internal/apiserver/infra/authz/subjectresolver"
	assignmentRepo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/assignment"
	policyRepo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/policy"
	roleRepo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/role"
	host "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/uow/authz"
	"github.com/FangcunMount/iam/v5/internal/apiserver/testhelpers"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"github.com/stretchr/testify/require"
)

func TestPlannedAssignmentEntriesKeepNamedIdentityNoopAndCompanyScope(t *testing.T) {
	db := testhelpers.SetupTempSQLiteDB(t)
	require.NoError(t, db.AutoMigrate(&roleRepo.RolePO{}, &assignmentRepo.AssignmentPO{}, &policyRepo.PolicyVersionPO{}))
	ctx := management.WithAuthenticatedService(context.Background(), "admin")
	roles := roleRepo.NewRoleRepository(db)
	assignments := assignmentRepo.NewRepository(db)
	roleByName := seedRoles(t, ctx, roles, "qs:staff", "qs:evaluator")
	resolver := subjectresolver.NewUserSubjectResolver(existingUserResolver{})
	events := new(eventStager)
	base := host.NewUnitOfWork(db, resolver, events)
	require.NoError(t, base.WithinTx(ctx, func(ctx context.Context, repos uow.TxRepositories) error {
		_, e := repos.PolicyVersions.GetOrCreate(ctx)
		return e
	}))
	probe := &fixture.PlanningProbe{Delegate: base}
	service := app.NewCommandService(domain.NewValidator(roles, resolver), roles, probe, nil, management.NewGuard(nil))
	sub, err := subject.NewUserRef(meta.FromUint64(100))
	require.NoError(t, err)
	version, err := service.GrantByRoleName(ctx, app.GrantByRoleNameCommand{Subject: sub, RoleName: "qs:staff", GrantedBy: "operator"})
	require.NoError(t, err)
	require.Equal(t, int64(2), version)
	require.Equal(t, "assignment.grant_named", probe.Requests[0].Operation)
	input := probe.Requests[0].Input.(map[string]any)
	require.Equal(t, "qs:staff", input["role_name"])
	require.NotContains(t, input, "role_id")
	version, err = service.RevokeByRoleName(ctx, app.RevokeByRoleNameCommand{Subject: sub, RoleName: "qs:staff", ChangedBy: "operator"})
	require.NoError(t, err)
	require.Equal(t, int64(3), version)
	version, err = service.RevokeByRoleName(ctx, app.RevokeByRoleNameCommand{Subject: sub, RoleName: "qs:staff", ChangedBy: "operator"})
	require.NoError(t, err)
	require.Equal(t, int64(3), version)
	require.Equal(t, 2, probe.Writes)
	require.Equal(t, 2, events.Count(), "empty revoke manufactured a new version")
	stores, err := scope.New(meta.FromUint64(1), scope.Stores, []meta.ID{meta.FromUint64(10), meta.FromUint64(11)})
	require.NoError(t, err)
	name, err := role.NewName("qs:staff")
	require.NoError(t, err)
	command := app.ReplaceManagedAssignmentsCommand{Subject: sub, OrgID: meta.FromUint64(1), ScopedRoles: []domain.ScopedRoleGrant{{RoleName: name, Scope: stores}}, ManagedRoleNames: []string{"qs:staff", "qs:evaluator"}, ChangedBy: "operator", ExpectedPolicyVersion: 3}
	result, err := service.ReplaceManagedAssignments(ctx, command)
	require.NoError(t, err)
	require.True(t, result.Changed)
	require.Equal(t, int64(4), result.PolicyVersion)
	company, err := scope.New(meta.FromUint64(2), scope.AllStores, nil)
	require.NoError(t, err)
	other := command
	other.OrgID = meta.FromUint64(2)
	other.ScopedRoles = []domain.ScopedRoleGrant{{RoleName: name, Scope: company}}
	other.ExpectedPolicyVersion = 4
	result, err = service.ReplaceManagedAssignments(ctx, other)
	require.NoError(t, err)
	require.True(t, result.Changed)
	require.Equal(t, int64(5), result.PolicyVersion)
	_, err = service.ReplaceManagedAssignments(ctx, command)
	require.ErrorIs(t, err, policy.ErrStaleVersion, "explicit stale version must fail even when desired state is equal")
	command.ExpectedPolicyVersion = 5
	replacement, err := scope.New(meta.FromUint64(1), scope.Stores, []meta.ID{meta.FromUint64(99)})
	require.NoError(t, err)
	probe.BeforePreview = func() { command.ManagedRoleNames[0] = "qs:external"; command.ScopedRoles[0].Scope = replacement }
	result, err = service.ReplaceManagedAssignments(ctx, command)
	require.NoError(t, err)
	require.False(t, result.Changed)
	require.Equal(t, int64(5), result.PolicyVersion)
	require.Equal(t, 4, events.Count())
	require.Equal(t, 4, probe.Writes)
	latest := probe.Requests[len(probe.Requests)-1]
	require.Equal(t, "org:1", latest.Scope)
	intent, err := change.NewCommandIntent(latest.Operation, "service:admin", latest.Scope, "same-key", latest.Input)
	require.NoError(t, err)
	require.Contains(t, string(intent.Input()), `"store_ids":["10","11"]`)
	rows, err := assignments.ListBySubject(ctx, domain.SubjectType(sub.Type), sub.ID)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	for _, a := range rows {
		require.Equal(t, roleByName["qs:staff"].ID, a.RoleID)
		value, ok := a.Scope()
		require.True(t, ok)
		if value.OrgID() == meta.FromUint64(1) {
			require.Equal(t, []meta.ID{meta.FromUint64(10), meta.FromUint64(11)}, value.StoreIDs())
		} else {
			require.Equal(t, meta.FromUint64(2), value.OrgID())
			require.Equal(t, scope.AllStores, value.Kind())
		}
	}
}

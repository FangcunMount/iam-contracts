//go:build integration

package migration

import (
	"context"
	"errors"
	"testing"
	"time"

	assignmentapp "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/assignment"
	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/management"
	grantapp "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/permissiongrant"
	change "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/policychange"
	resourceapp "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/resource"
	roleapp "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/role"
	assignment "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/assignment"
	policy "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/policy"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/subject"
	grantrepo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/permissiongrant"
	policyrepo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/policy"
	rolerepo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/role"
	hostuow "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/uow/authz"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	txmysql "github.com/FangcunMount/reliable-messaging/storage/transaction/mysql"
	"github.com/FangcunMount/reliable-messaging/transaction"
	"github.com/FangcunMount/reliable-messaging/transport"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// nativeStoredHalfSender deliberately replaces Broker I/O only. It exercises
// real released Store/reservation/guard/witness and actual host app entrypoints.
// This is original-database atomicity evidence, never a Broker failure test.
type nativeStoredHalfSender struct {
	store       *txmysql.Store
	calls       int
	reject      bool
	beforeLocal func() error
}

func (s *nativeStoredHalfSender) Reserve(ctx context.Context, r transaction.Request, p transaction.FrozenPlan) (transaction.Reservation, error) {
	return s.store.Reserve(ctx, r, p, transaction.ReserveOptions{Lease: time.Minute})
}
func (s *nativeStoredHalfSender) Execute(ctx context.Context, e transaction.Execution, local transaction.LocalExecutor) (transaction.Result, error) {
	s.calls++
	result := transaction.InitialResult(e)
	if err := s.store.BeginPrepare(ctx, e); err != nil {
		return result, err
	}
	if s.reject {
		_, err := s.store.Abort(ctx, e)
		result.Local = transaction.LocalAborted
		return result, errors.Join(err, errors.New("explicit fixture half rejection"))
	}
	for i, ref := range e.References() {
		if err := s.store.RecordHalf(ctx, e, ref, transaction.HalfReceipt{Outcome: transaction.HalfPrepared, BrokerMessageID: "stored-fixture-only", BrokerTransactionID: "stored-fixture-only"}); err != nil {
			return result, err
		}
		result.Branches[i].Half = transaction.HalfPrepared
		result.Branches[i].Final = transaction.FinalUnknown
	}
	if s.beforeLocal != nil {
		if err := s.beforeLocal(); err != nil {
			return result, err
		}
	}
	localCtx, cancel := context.WithDeadline(ctx, e.LeaseUntil)
	defer cancel()
	outcome, err := local(localCtx, e)
	if outcome == transaction.LocalAborted {
		_, abortErr := s.store.Abort(ctx, e)
		err = errors.Join(err, abortErr)
	}
	evidence := make([]*transaction.Evidence, 0, len(e.References()))
	for _, ref := range e.References() {
		value, readErr := s.store.Evidence(ctx, ref)
		if readErr != nil {
			return result, readErr
		}
		evidence = append(evidence, value)
	}
	result.Local = transaction.DecideLocal(e.References(), evidence)
	if result.Local == transaction.LocalCommitted {
		return result, errors.New("fixture final confirmation unavailable")
	}
	return result, err
}

type nativeSubjects struct{}

func (nativeSubjects) Supports(kind subject.Type) bool            { return kind == subject.TypeUser }
func (nativeSubjects) Resolve(context.Context, subject.Ref) error { return nil } // identity resolution is outside this atomicity fixture

type nativeCatalogWriter struct{}

func (nativeCatalogWriter) RequireCatalogWrite(context.Context, subject.Ref, string) error {
	return nil
} // permissions have separate T05/T08 gates

func testM3PlannedHostEntries(t *testing.T, db *gorm.DB, binding transaction.Binding) {
	t.Helper()
	local, err := hostuow.NewTransactionUnitOfWork(db, nativeSubjects{}, binding)
	require.NoError(t, err)
	sender := &nativeStoredHalfSender{store: local.Store()}
	logical := "iam.authz.version.v2"
	physical, err := transport.PhysicalTopicV2(logical, transport.TransactionMessageType, binding.Owner, "m3-1")
	require.NoError(t, err)
	planned, err := hostuow.NewPlannedUnitOfWork(local, sender, hostuow.PlanConfig{LogicalTopic: logical, PhysicalTopic: physical, Sends: 8, ProofBudget: 2 * time.Second, Admission: func() bool { return true }})
	require.NoError(t, err)
	ctx := management.WithAuthenticatedService(t.Context(), "admin")
	var beforeOutbox int64
	require.NoError(t, db.Table("rm_outbox").Count(&beforeOutbox).Error)
	roles := roleapp.NewRoleCatalog(planned, nil, management.NewGuard(nil))
	keyCtx, err := change.WithRequestKey(ctx, "native-first-role")
	require.NoError(t, err)
	command, err := roleapp.NewCreateRoleCommand("m3:planned", "Native plan", "")
	require.NoError(t, err)
	command.ChangedBy = "native-operator"
	role, err := roles.CreateRole(keyCtx, command)
	require.NoError(t, err)
	require.Positive(t, role.ID.Int64())
	require.Equal(t, 1, sender.calls)
	_, err = roles.CreateRole(keyCtx, command)
	var known *hostuow.KnownOperation
	require.ErrorAs(t, err, &known)
	require.Equal(t, transaction.LocalCommitted, known.Local)
	require.Equal(t, 1, sender.calls)
	different := command
	different.DisplayName = "different input"
	_, err = roles.CreateRole(keyCtx, different)
	require.ErrorIs(t, err, transaction.ErrInputConflict)
	require.Equal(t, 1, sender.calls)
	resources := resourceapp.NewResourceCatalog(planned, nil, nativeCatalogWriter{})
	actor, err := subject.NewUserRef(meta.ID(7))
	require.NoError(t, err)
	resourceCmd, err := resourceapp.NewCreateResourceCommand("m3:test:collection:native", "Native resource", "", "", "", []string{"read"}, "")
	require.NoError(t, err)
	resourceCmd.Actor = actor
	resourceCmd.ChangedBy = "native-operator"
	resource, err := resources.CreateResource(ctx, resourceCmd)
	require.NoError(t, err)
	require.Positive(t, resource.ID.Uint64())
	grants := grantapp.NewService(planned, grantrepo.NewRepository(db), nil, management.NewGuard(nil))
	grant, err := grants.Create(ctx, grantapp.CreateCommand{RoleID: role.ID, ResourceID: resource.ID, Action: "read", GrantedBy: "native-operator"})
	require.NoError(t, err)
	require.Positive(t, grant.ID.Int64())
	roleRepository := rolerepo.NewRoleRepository(db)
	assignments := assignmentapp.NewCommandService(assignment.NewValidatorWithSubjectResolver(roleRepository, nativeSubjects{}), roleRepository, planned, nil, management.NewGuard(nil))
	created, err := assignments.Grant(ctx, assignmentapp.GrantCommand{SubjectType: assignment.SubjectTypeUser, SubjectID: meta.ID(7), RoleID: role.ID, GrantedBy: "native-operator"})
	require.NoError(t, err)
	require.Positive(t, created.ID.Uint64())
	require.NoError(t, assignments.Revoke(ctx, assignmentapp.RevokeCommand{SubjectType: assignment.SubjectTypeUser, SubjectID: meta.ID(7), RoleID: role.ID, ChangedBy: "native-operator"}))
	calls := sender.calls
	require.NoError(t, assignments.Revoke(ctx, assignmentapp.RevokeCommand{SubjectType: assignment.SubjectTypeUser, SubjectID: meta.ID(7), RoleID: role.ID, ChangedBy: "native-operator"}))
	require.Equal(t, calls, sender.calls, "authorized no-op prepared a new half")
	require.NoError(t, grants.Revoke(ctx, grantapp.RevokeCommand{GrantID: grant.ID, RevokedBy: "native-operator"}))
	before, err := policyrepo.NewPolicyVersionRepository(db).GetCurrent(ctx)
	require.NoError(t, err)
	sender.reject = true
	rejected, err := roleapp.NewCreateRoleCommand("m3:rejected", "Rejected", "")
	require.NoError(t, err)
	rejected.ChangedBy = "native-operator"
	_, err = roles.CreateRole(ctx, rejected)
	require.Error(t, err)
	var count int64
	require.NoError(t, db.Table("authz_roles").Where("name=?", "m3:rejected").Count(&count).Error)
	require.Zero(t, count)
	after, err := policyrepo.NewPolicyVersionRepository(db).GetCurrent(ctx)
	require.NoError(t, err)
	require.Equal(t, before.Version, after.Version)
	sender.reject = false
	sender.beforeLocal = func() error {
		return db.Exec("UPDATE authz_policy_versions SET policy_version=policy_version+1 WHERE id=1").Error
	}
	conflicted, err := roleapp.NewCreateRoleCommand("m3:conflicted", "Conflicted", "")
	require.NoError(t, err)
	conflicted.ChangedBy = "native-operator"
	_, err = roles.CreateRole(ctx, conflicted)
	require.ErrorIs(t, err, policy.ErrStaleVersion)
	require.NoError(t, db.Table("authz_roles").Where("name=?", "m3:conflicted").Count(&count).Error)
	require.Zero(t, count)
	var afterOutbox int64
	require.NoError(t, db.Table("rm_outbox").Count(&afterOutbox).Error)
	require.Equal(t, beforeOutbox, afterOutbox, "new host writes used a publishing Outbox")
	planned.StopAdmission()
	require.NoError(t, planned.Wait(t.Context()))
	_, err = roles.CreateRole(ctx, rejected)
	require.ErrorIs(t, err, hostuow.ErrWriteAdmissionClosed)
}

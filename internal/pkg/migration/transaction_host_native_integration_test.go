//go:build integration

package migration

import (
	"context"
	"errors"
	"testing"
	"time"

	assignmentapp "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/assignment"
	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/management"
	operation "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/operation"
	grantapp "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/permissiongrant"
	change "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/policychange"
	resourceapp "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/resource"
	roleapp "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/role"
	assignment "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/assignment"
	policy "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/policy"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/subject"
	journal "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/authzoperation"
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

// Actual host app entry + released SDK + business journal in one original root.
// Only Broker I/O is an explicitly stored-half fixture; this is not Broker E2E.
func testM3JournalHostEntries(t *testing.T, db *gorm.DB, binding transaction.Binding) {
	local, err := hostuow.NewTransactionUnitOfWork(db, nativeSubjects{}, binding)
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	records, err := journal.New(pool, binding)
	require.NoError(t, err)
	sender := &nativeStoredHalfSender{store: local.Store()}
	topic := "iam.authz.version.v2"
	physical, err := transport.PhysicalTopicV2(topic, transport.TransactionMessageType, binding.Owner, "m3-1")
	require.NoError(t, err)
	writer, err := hostuow.NewPlannedUnitOfWork(local, sender, hostuow.PlanConfig{LogicalTopic: topic, PhysicalTopic: physical, Sends: 8, ProofBudget: 2 * time.Second, Admission: func() bool { return true }, Journal: records, OperationLease: time.Minute})
	require.NoError(t, err)
	service := roleapp.NewRoleCatalog(writer, nil, management.NewGuard(nil))
	ctx := management.WithAuthenticatedService(t.Context(), "admin")
	var reference string
	ctx, err = operation.WithReferenceSink(ctx, func(id string) { reference = id })
	require.NoError(t, err)
	create, err := roleapp.NewCreateRoleCommand("m3:journal-idempotency", "First result", "")
	require.NoError(t, err)
	create.ChangedBy = "native-operator"
	keyed, err := change.WithRequestKey(ctx, "journal-create-role")
	require.NoError(t, err)
	first, err := service.CreateRole(keyed, create)
	require.NoError(t, err)
	require.NotEmpty(t, reference)
	originalID, originalRef := first.ID, reference
	duplicate, err := service.CreateRole(keyed, create)
	require.NoError(t, err)
	require.Equal(t, originalID, duplicate.ID)
	require.Equal(t, originalRef, reference)
	require.Equal(t, 1, sender.calls)
	conflict := create
	conflict.DisplayName = "different input"
	_, err = service.CreateRole(keyed, conflict)
	require.ErrorIs(t, err, operation.ErrConflict)
	require.Equal(t, 1, sender.calls)
	original, err := records.Read(t.Context(), originalRef)
	require.NoError(t, err)
	require.Equal(t, journal.Committed, original.State)
	protocol, err := records.ReadProtocol(t.Context(), original.Candidate)
	require.NoError(t, err)
	require.Equal(t, transaction.LocalCommitted, protocol.Local)
	require.Equal(t, transaction.FinalUnknown, protocol.Final)
	// An authorized no-op creates a durable result but no fake policy event.
	same := "First result"
	update, err := roleapp.NewUpdateRoleCommand(originalID, &same, nil)
	require.NoError(t, err)
	update.ChangedBy = "native-operator"
	noop, err := change.WithRequestKey(ctx, "journal-noop-role")
	require.NoError(t, err)
	before, err := policyrepo.NewPolicyVersionRepository(db).GetCurrent(ctx)
	require.NoError(t, err)
	unchanged, err := service.UpdateRole(noop, update)
	require.NoError(t, err)
	require.Equal(t, "First result", unchanged.DisplayName)
	require.Equal(t, 1, sender.calls)
	noopRef := reference
	receipt, err := records.Read(t.Context(), noopRef)
	require.NoError(t, err)
	require.Equal(t, journal.Committed, receipt.State)
	require.False(t, receipt.Candidate.Changed())
	require.Equal(t, before.Version, receipt.PolicyVersion)
	// Replay restores the original result after later facts change; it cannot
	// resolve the current object and accidentally execute another update.
	require.NoError(t, db.Exec("UPDATE authz_roles SET display_name='later' WHERE id=?", originalID).Error)
	require.NoError(t, db.Exec("UPDATE authz_policy_versions SET policy_version=policy_version+1 WHERE id=1").Error)
	restored, err := service.UpdateRole(noop, update)
	require.NoError(t, err)
	require.Equal(t, "First result", restored.DisplayName)
	require.Equal(t, noopRef, reference)
	require.Equal(t, 1, sender.calls)
	var actual string
	require.NoError(t, db.Raw("SELECT display_name FROM authz_roles WHERE id=?", originalID).Row().Scan(&actual))
	require.Equal(t, "later", actual)

	// All typed domain results restore their original IDs without reentering
	// business, including private ResourceKey and Assignment Scope values.
	resources := resourceapp.NewResourceCatalog(writer, nil, nativeCatalogWriter{})
	actor, err := subject.NewUserRef(meta.ID(7))
	require.NoError(t, err)
	resourceCmd, err := resourceapp.NewCreateResourceCommand("m3:journal:collection:resource", "Resource", "", "", "", []string{"read"}, "")
	require.NoError(t, err)
	resourceCmd.Actor = actor
	resourceCmd.ChangedBy = "native-operator"
	resourceKey, err := change.WithRequestKey(ctx, "journal-create-resource")
	require.NoError(t, err)
	resource, err := resources.CreateResource(resourceKey, resourceCmd)
	require.NoError(t, err)
	resourceCalls := sender.calls
	resourceAgain, err := resources.CreateResource(resourceKey, resourceCmd)
	require.NoError(t, err)
	require.Equal(t, resource.ID, resourceAgain.ID)
	require.Equal(t, resource.KeyString(), resourceAgain.KeyString())
	require.Equal(t, resourceCalls, sender.calls)
	grants := grantapp.NewService(writer, grantrepo.NewRepository(db), nil, management.NewGuard(nil))
	grantKey, err := change.WithRequestKey(ctx, "journal-create-grant")
	require.NoError(t, err)
	grantCmd := grantapp.CreateCommand{RoleID: originalID, ResourceID: resource.ID, Action: "read", GrantedBy: "native-operator"}
	grant, err := grants.Create(grantKey, grantCmd)
	require.NoError(t, err)
	grantCalls := sender.calls
	grantAgain, err := grants.Create(grantKey, grantCmd)
	require.NoError(t, err)
	require.Equal(t, grant.ID, grantAgain.ID)
	require.Equal(t, grant.GrantedAt, grantAgain.GrantedAt)
	require.Equal(t, grant.GrantKey, grantAgain.GrantKey)
	require.Equal(t, grantCalls, sender.calls)
	roles := rolerepo.NewRoleRepository(db)
	assignments := assignmentapp.NewCommandService(assignment.NewValidatorWithSubjectResolver(roles, nativeSubjects{}), roles, writer, nil, management.NewGuard(nil))
	assignmentKey, err := change.WithRequestKey(ctx, "journal-create-assignment")
	require.NoError(t, err)
	assignmentCmd := assignmentapp.GrantCommand{SubjectType: assignment.SubjectTypeUser, SubjectID: meta.ID(7), RoleID: originalID, GrantedBy: "native-operator"}
	assigned, err := assignments.Grant(assignmentKey, assignmentCmd)
	require.NoError(t, err)
	assignmentCalls := sender.calls
	assignedAgain, err := assignments.Grant(assignmentKey, assignmentCmd)
	require.NoError(t, err)
	require.Equal(t, assigned.ID, assignedAgain.ID)
	require.Equal(t, assignmentCalls, sender.calls)
	sender.reject = true
	rejected, err := roleapp.NewCreateRoleCommand("m3:journal-rejected", "Rejected", "")
	require.NoError(t, err)
	rejected.ChangedBy = "native-operator"
	rejectKey, err := change.WithRequestKey(ctx, "journal-reject-role")
	require.NoError(t, err)
	_, err = service.CreateRole(rejectKey, rejected)
	var outcome *operation.OutcomeError
	require.ErrorAs(t, err, &outcome)
	require.Equal(t, transaction.LocalAborted, outcome.Local)
	aborted, err := records.Read(t.Context(), reference)
	require.NoError(t, err)
	require.Equal(t, journal.Aborted, aborted.State)
	var absent int64
	require.NoError(t, db.Table("authz_roles").Where("name=?", "m3:journal-rejected").Count(&absent).Error)
	require.Zero(t, absent)
	writer.StopAdmission()
	require.NoError(t, writer.Wait(t.Context()))
}

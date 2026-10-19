package assignment

import (
	"context"
	"sort"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	admission "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/assignmentadmission"
	change "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/policychange"
	uow "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/uow"
	domain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/assignment"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/subject"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
)

const assignmentOperations = "iam:authz:collection:assignments"

func (s *CommandService) previewGrant(cmd GrantCommand) uow.WriteRequest {
	return uow.WriteRequest{Operation: "assignment.grant", Freeze: change.AllocateKeys(change.ArtifactKey("assignment", string(cmd.SubjectType), cmd.SubjectID.String(), cmd.RoleID.String(), "0")), Scope: "global", ChangedBy: cmd.GrantedBy, PermissionResource: assignmentOperations, PermissionActions: []string{"grant"}, Input: map[string]any{"subject_type": cmd.SubjectType, "subject_id": cmd.SubjectID.String(), "role_id": cmd.RoleID.String(), "granted_by": cmd.GrantedBy}, Preview: func(ctx context.Context, repos uow.TxRepositories) (bool, error) {
		validator := domain.NewValidatorWithSubjectResolver(repos.Roles, repos.SubjectResolver)
		if err := validator.CheckSubjectExists(ctx, cmd.SubjectType, cmd.SubjectID); err != nil {
			return false, err
		}
		role, err := repos.Roles.FindByID(ctx, cmd.RoleID)
		if err != nil {
			return false, err
		}
		if err = s.guard.RequireAssignment(ctx, subject.Ref{Type: subject.Type(cmd.SubjectType), ID: cmd.SubjectID}, role, admission.OperationGrant, cmd.GrantedBy); err != nil {
			return false, err
		}
		reader, ok := repos.Assignments.(domain.PlanningReader)
		if !ok {
			return false, change.ErrInvalidChangePlan
		}
		count, err := reader.CountTarget(ctx, cmd.SubjectType, cmd.SubjectID, cmd.RoleID, true)
		if err != nil {
			return false, err
		}
		if count > 0 {
			return false, perrors.WithCode(code.ErrAssignmentAlreadyExists, "binding already exists")
		}
		return true, nil
	}}
}
func (s *CommandService) previewRevoke(cmd RevokeCommand) uow.WriteRequest {
	return uow.WriteRequest{Operation: "assignment.revoke", Scope: "global", ChangedBy: cmd.ChangedBy, PermissionResource: assignmentOperations, PermissionActions: []string{"revoke"}, Input: map[string]any{"subject_type": cmd.SubjectType, "subject_id": cmd.SubjectID.String(), "role_id": cmd.RoleID.String(), "changed_by": cmd.ChangedBy, "reason": revokeReason(cmd.Reason)}, Preview: func(ctx context.Context, repos uow.TxRepositories) (bool, error) {
		role, err := repos.Roles.FindByID(ctx, cmd.RoleID)
		if err != nil {
			return false, err
		}
		if err = s.guard.RequireAssignment(ctx, subject.Ref{Type: subject.Type(cmd.SubjectType), ID: cmd.SubjectID}, role, admission.OperationRevoke, cmd.ChangedBy); err != nil {
			return false, err
		}
		reader, ok := repos.Assignments.(domain.PlanningReader)
		if !ok {
			return false, change.ErrInvalidChangePlan
		}
		count, err := reader.CountTarget(ctx, cmd.SubjectType, cmd.SubjectID, cmd.RoleID, false)
		return count > 0, err
	}}
}
func (s *CommandService) previewRevokeID(cmd RevokeByIDCommand) uow.WriteRequest {
	return uow.WriteRequest{Operation: "assignment.revoke_id", Scope: "global", ChangedBy: cmd.ChangedBy, PermissionResource: assignmentOperations, PermissionActions: []string{"revoke"}, Input: map[string]any{"assignment_id": cmd.AssignmentID.String(), "changed_by": cmd.ChangedBy, "reason": revokeReason(cmd.Reason)}, Preview: func(ctx context.Context, repos uow.TxRepositories) (bool, error) {
		existing, err := repos.Assignments.FindByID(ctx, cmd.AssignmentID)
		if err != nil {
			return false, err
		}
		role, err := repos.Roles.FindByID(ctx, existing.RoleID)
		if err != nil {
			return false, err
		}
		if err = s.guard.RequireAssignment(ctx, subject.Ref{Type: existing.SubjectType, ID: existing.SubjectID}, role, admission.OperationRevoke, cmd.ChangedBy); err != nil {
			return false, err
		}
		reader, ok := repos.Assignments.(domain.PlanningReader)
		if !ok {
			return false, change.ErrInvalidChangePlan
		}
		return reader.ActiveID(ctx, cmd.AssignmentID)
	}}
}
func replacementInput(cmd ReplaceManagedAssignmentsCommand) map[string]any {
	names, managed := append([]string{}, cmd.RoleNames...), append([]string{}, cmd.ManagedRoleNames...)
	sort.Strings(names)
	sort.Strings(managed)
	scoped := append([]domain.ScopedRoleGrant(nil), cmd.ScopedRoles...)
	sort.Slice(scoped, func(i, j int) bool { return scoped[i].RoleName.String() < scoped[j].RoleName.String() })
	targets := make([]map[string]any, 0, len(scoped))
	for _, target := range scoped {
		stores := make([]string, 0, len(target.Scope.StoreIDs()))
		for _, id := range target.Scope.StoreIDs() {
			stores = append(stores, id.String())
		}
		targets = append(targets, map[string]any{"role_name": target.RoleName.String(), "org_id": target.Scope.OrgID().String(), "scope_kind": string(target.Scope.Kind()), "store_ids": stores})
	}
	return map[string]any{"subject": cmd.Subject.String(), "org_id": cmd.OrgID.String(), "expected_policy_version": cmd.ExpectedPolicyVersion, "role_names": names, "managed_role_names": managed, "scoped_roles": targets, "changed_by": cmd.ChangedBy, "reason": cmd.Reason}
}
func (s *CommandService) previewReplacement(cmd ReplaceManagedAssignmentsCommand, result *ReplaceManagedAssignmentsResult) uow.WriteRequest {
	scope := "global"
	if cmd.OrgID > 0 {
		scope = "org:" + cmd.OrgID.String()
	}
	var createKeys []string
	return uow.WriteRequest{Operation: "assignment.replace_managed", Freeze: func(b *change.ArtifactBuilder) error { return change.AllocateKeys(createKeys...)(b) }, Scope: scope, ChangedBy: cmd.ChangedBy, PermissionResource: assignmentOperations, PermissionActions: []string{"grant", "revoke"}, Input: replacementInput(cmd), Preview: func(ctx context.Context, repos uow.TxRepositories) (bool, error) {
		if err := s.guard.RequireReplacement(ctx, cmd.Subject, cmd.RoleNames, cmd.ManagedRoleNames, cmd.ChangedBy); err != nil {
			return false, err
		}
		managed, plan, err := s.loadReplacement(ctx, cmd, repos, false)
		if err != nil {
			return false, err
		}
		createKeys = nil
		for _, name := range plan.Grants {
			target := managed[name.String()]
			if target == nil {
				return false, change.ErrInvalidChangePlan
			}
			createKeys = append(createKeys, change.ArtifactKey("assignment", string(cmd.Subject.Type), cmd.Subject.ID.String(), target.ID.String(), cmd.OrgID.String()))
		}
		result.DirectRoles = make([]string, 0, len(plan.DirectRoles))
		for _, name := range plan.DirectRoles {
			result.DirectRoles = append(result.DirectRoles, name.String())
		}
		sort.Strings(result.DirectRoles)
		result.Changed = plan.Changed
		result.PolicyVersion, _ = change.ExpectedVersion(ctx)
		return plan.Changed, nil
	}}
}

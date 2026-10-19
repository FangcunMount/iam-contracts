// Package assignment contains the native role-assignment write use cases.
package assignment

import (
	"context"
	admission "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/assignmentadmission"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
	"sort"
	"strings"

	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/management"

	"github.com/FangcunMount/component-base/pkg/errors"
	policychange "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/policychange"
	authzuow "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/uow"
	assignmentDomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/assignment"
	roleDomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/role"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/subject"
)

type GrantByRoleNameCommand struct {
	Subject             subject.Ref
	RoleName, GrantedBy string
}

type RevokeByRoleNameCommand struct {
	Subject                     subject.Ref
	RoleName, ChangedBy, Reason string
}

type CommandService struct {
	guard     management.Guard
	validator assignmentDomain.Validator
	roles     roleDomain.Repository
	uow       authzuow.UnitOfWork
	reloader  policychange.RuntimePolicyReloader
}

func NewCommandService(validator assignmentDomain.Validator, roles roleDomain.Repository, uow authzuow.UnitOfWork, reloader policychange.RuntimePolicyReloader, guard management.Guard) *CommandService {
	return &CommandService{validator: validator, roles: roles, uow: uow, reloader: reloader, guard: guard}
}

func (s *CommandService) Grant(ctx context.Context, cmd GrantCommand) (*assignmentDomain.Assignment, error) {
	result, err := s.executeGrant(ctx, cmd)
	return result.Assignment, err
}
func (s *CommandService) Revoke(ctx context.Context, cmd RevokeCommand) error {
	_, err := s.revokeWithVersion(ctx, cmd)
	return err
}

func (s *CommandService) RevokeByID(ctx context.Context, cmd RevokeByIDCommand) error {
	_, err := s.commit(ctx, cmd.ChangedBy, revokeReason(cmd.Reason), s.previewRevokeID(cmd), func(txCtx context.Context, tx authzuow.TxRepositories) error {
		assignment, err := tx.Assignments.FindByID(txCtx, cmd.AssignmentID)
		if err != nil {
			return errors.Wrap(err, "获取赋权记录失败")
		}
		target, err := tx.Roles.FindByIDForUpdate(txCtx, assignment.RoleID)
		if err != nil {
			return err
		}
		if err := s.guard.RequireAssignment(txCtx, subject.Ref{Type: assignment.SubjectType, ID: assignment.SubjectID}, target, admission.OperationRevoke, cmd.ChangedBy); err != nil {
			return err
		}
		return tx.Assignments.Delete(txCtx, assignment.ID)
	})
	return err
}

func (s *CommandService) GrantByRoleName(ctx context.Context, cmd GrantByRoleNameCommand) (int64, error) {
	if s == nil || s.roles == nil {
		return 0, errors.New("role binding command service unavailable")
	}
	role, err := s.roles.FindByName(ctx, cmd.RoleName)
	if err != nil {
		return 0, err
	}
	grant, err := NewGrantCommand(assignmentDomain.SubjectType(cmd.Subject.Type), cmd.Subject.ID, role.ID, cmd.GrantedBy)
	if err != nil {
		return 0, err
	}
	result, err := s.executeGrant(ctx, grant, authzuow.WriteRequest{Operation: "assignment.grant_named", Input: map[string]any{"subject": cmd.Subject.String(), "role_name": cmd.RoleName, "granted_by": cmd.GrantedBy}})
	return result.Version, err
}

func (s *CommandService) RevokeByRoleName(ctx context.Context, cmd RevokeByRoleNameCommand) (int64, error) {
	if s == nil || s.roles == nil {
		return 0, errors.New("role binding command service unavailable")
	}
	role, err := s.roles.FindByName(ctx, cmd.RoleName)
	if err != nil {
		return 0, err
	}
	revoke, err := NewRevokeCommand(assignmentDomain.SubjectType(cmd.Subject.Type), cmd.Subject.ID, role.ID, cmd.ChangedBy, cmd.Reason)
	if err != nil {
		return 0, err
	}
	return s.revokeWithVersion(ctx, revoke, authzuow.WriteRequest{Operation: "assignment.revoke_named", Input: map[string]any{"subject": cmd.Subject.String(), "role_name": cmd.RoleName, "changed_by": cmd.ChangedBy, "reason": revokeReason(cmd.Reason)}})
}

type grantResult struct {
	Assignment *assignmentDomain.Assignment
	Version    int64
}

func (s *CommandService) executeGrant(ctx context.Context, cmd GrantCommand, identities ...authzuow.WriteRequest) (grantResult, error) {
	if err := s.validator.ValidateGrantParameters(cmd.SubjectType, cmd.SubjectID, cmd.RoleID, cmd.GrantedBy); err != nil {
		return grantResult{}, err
	}
	request := s.previewGrant(cmd)
	if len(identities) > 0 {
		request.Operation = identities[0].Operation
		request.Input = identities[0].Input
	}
	var result grantResult
	version, err := s.commit(ctx, cmd.GrantedBy, "binding grant", request, func(txCtx context.Context, tx authzuow.TxRepositories) error {
		txValidator := assignmentDomain.NewValidatorWithSubjectResolver(tx.Roles, tx.SubjectResolver)
		if err := txValidator.CheckRoleExists(txCtx, cmd.RoleID); err != nil {
			return err
		}
		if err := txValidator.CheckSubjectExists(txCtx, cmd.SubjectType, cmd.SubjectID); err != nil {
			return err
		}
		role, err := tx.Roles.FindByIDForUpdate(txCtx, cmd.RoleID)
		if err != nil {
			return errors.Wrap(err, "获取角色失败")
		}
		if err := s.guard.RequireAssignment(txCtx, subject.Ref{Type: cmd.SubjectType, ID: cmd.SubjectID}, role, admission.OperationGrant, cmd.GrantedBy); err != nil {
			return err
		}
		assignment, err := assignmentDomain.NewAssignment(cmd.SubjectType, cmd.SubjectID, cmd.RoleID, assignmentDomain.WithGrantedBy(cmd.GrantedBy))
		if err != nil {
			return err
		}
		if err := tx.Assignments.Create(txCtx, &assignment); err != nil {
			return errors.Wrap(err, "创建赋权失败")
		}
		result.Assignment = &assignment
		return nil
	})
	if err != nil {
		return grantResult{}, err
	}
	result.Version = version
	return result, nil
}

func (s *CommandService) revokeWithVersion(ctx context.Context, cmd RevokeCommand, identities ...authzuow.WriteRequest) (int64, error) {
	if err := s.validator.ValidateRevokeParameters(cmd.SubjectType, cmd.SubjectID, cmd.RoleID); err != nil {
		return 0, err
	}
	request := s.previewRevoke(cmd)
	if len(identities) > 0 {
		request.Operation = identities[0].Operation
		request.Input = identities[0].Input
	}
	return s.commit(ctx, cmd.ChangedBy, revokeReason(cmd.Reason), request, func(txCtx context.Context, tx authzuow.TxRepositories) error {
		role, err := tx.Roles.FindByIDForUpdate(txCtx, cmd.RoleID)
		if err != nil {
			return errors.Wrap(err, "获取角色失败")
		}
		if err := s.guard.RequireAssignment(txCtx, subject.Ref{Type: cmd.SubjectType, ID: cmd.SubjectID}, role, admission.OperationRevoke, cmd.ChangedBy); err != nil {
			return err
		}
		return tx.Assignments.DeleteBySubjectAndRole(txCtx, cmd.SubjectType, cmd.SubjectID, cmd.RoleID)
	})
}

func (s *CommandService) ReplaceManagedAssignments(ctx context.Context, cmd ReplaceManagedAssignmentsCommand) (ReplaceManagedAssignmentsResult, error) {
	if s == nil || s.uow == nil {
		return ReplaceManagedAssignmentsResult{}, errors.New("role binding command service unavailable")
	}
	cmd.RoleNames = append([]string(nil), cmd.RoleNames...)
	cmd.ManagedRoleNames = append([]string(nil), cmd.ManagedRoleNames...)
	cmd.ScopedRoles = append([]assignmentDomain.ScopedRoleGrant(nil), cmd.ScopedRoles...)
	if cmd.OrgID < 0 || (cmd.OrgID == 0 && len(cmd.ScopedRoles) > 0) {
		return ReplaceManagedAssignmentsResult{}, errors.WithCode(code.ErrInvalidArgument, "scoped replacement company is required")
	}
	if cmd.OrgID > 0 {
		if len(cmd.RoleNames) > 0 {
			return ReplaceManagedAssignmentsResult{}, errors.WithCode(code.ErrInvalidArgument, "scoped roles replace legacy role names")
		}
		cmd.RoleNames = make([]string, 0, len(cmd.ScopedRoles))
		for _, target := range cmd.ScopedRoles {
			cmd.RoleNames = append(cmd.RoleNames, target.RoleName.String())
		}
	}
	validated, err := NewReplaceManagedAssignmentsCommand(
		cmd.Subject, cmd.RoleNames, cmd.ManagedRoleNames, cmd.ChangedBy, cmd.Reason,
	)
	if err != nil {
		return ReplaceManagedAssignmentsResult{}, err
	}
	validated.OrgID, validated.ScopedRoles = cmd.OrgID, cmd.ScopedRoles
	validated.ExpectedPolicyVersion = cmd.ExpectedPolicyVersion
	cmd = validated
	if err := s.guard.RequireReplacement(ctx, cmd.Subject, cmd.RoleNames, cmd.ManagedRoleNames, cmd.ChangedBy); err != nil {
		return ReplaceManagedAssignmentsResult{}, err
	}
	result := ReplaceManagedAssignmentsResult{}
	err = authzuow.WithinWrite(ctx, s.uow, s.previewReplacement(cmd, &result), func(txCtx context.Context, tx authzuow.TxRepositories) error {
		if _, planned := policychange.ExecutionPlan(txCtx); planned {
			if err := s.guard.RequireReplacement(txCtx, cmd.Subject, cmd.RoleNames, cmd.ManagedRoleNames, cmd.ChangedBy); err != nil {
				return err
			}
		}
		managedRoles, plan, err := s.loadReplacement(txCtx, cmd, tx, true)
		if err != nil {
			return err
		}
		result.DirectRoles = make([]string, 0, len(plan.DirectRoles))
		for _, roleName := range plan.DirectRoles {
			result.DirectRoles = append(result.DirectRoles, roleName.String())
		}
		sort.Strings(result.DirectRoles)
		for _, assignmentID := range plan.Revokes {
			if err := tx.Assignments.Delete(txCtx, assignmentID); err != nil {
				return errors.Wrap(err, "revoke managed assignment")
			}
		}
		for _, roleName := range plan.Grants {
			role := managedRoles[roleName.String()]
			options := []assignmentDomain.Option{assignmentDomain.WithGrantedBy(cmd.ChangedBy)}
			if cmd.OrgID > 0 {
				for _, target := range cmd.ScopedRoles {
					if target.RoleName == roleName {
						options = append(options, assignmentDomain.WithScope(target.Scope))
						break
					}
				}
			}
			assignment, err := assignmentDomain.NewAssignment(
				assignmentDomain.SubjectType(cmd.Subject.Type), cmd.Subject.ID, role.ID, options...,
			)
			if err != nil {
				return err
			}
			if err := tx.Assignments.Create(txCtx, &assignment); err != nil {
				return errors.Wrap(err, "grant managed assignment")
			}
		}
		result.Changed = plan.Changed

		if !result.Changed {
			version, err := tx.PolicyVersions.GetCurrent(txCtx)
			if err != nil {
				return err
			}
			if version != nil {
				result.PolicyVersion = version.Version
			}
			return nil
		}
		reason := strings.TrimSpace(cmd.Reason)
		if reason == "" {
			reason = "managed assignments replace"
		}
		version, err := tx.PolicyVersions.Increment(txCtx, cmd.ChangedBy, reason)
		if err != nil {
			return err
		}
		if err := policychange.StagePolicyVersionChanged(txCtx, tx.Events, version); err != nil {
			return err
		}
		result.PolicyVersion = version.Version
		return nil
	})
	if err != nil {
		return ReplaceManagedAssignmentsResult{}, err
	}
	if result.Changed {
		policychange.ReloadRuntimePolicy(ctx, s.reloader, "managed assignments replace")
	}
	return result, nil
}

func (s *CommandService) commit(ctx context.Context, changedBy, reason string, request authzuow.WriteRequest, mutation func(context.Context, authzuow.TxRepositories) error) (int64, error) {
	if s == nil || s.uow == nil {
		return 0, errors.New("role binding command service unavailable")
	}
	if strings.TrimSpace(changedBy) == "" {
		return 0, errors.New("authorization change actor is required")
	}
	var committedVersion int64
	changed := true
	preview := request.Preview
	request.Preview = func(readCtx context.Context, repos authzuow.TxRepositories) (bool, error) {
		var err error
		changed, err = preview(readCtx, repos)
		if err == nil && !changed {
			committedVersion, _ = policychange.ExpectedVersion(readCtx)
		}
		return changed, err
	}
	err := authzuow.WithinWrite(ctx, s.uow, request, func(txCtx context.Context, tx authzuow.TxRepositories) error {
		if err := mutation(txCtx, tx); err != nil {
			return err
		}
		version, err := tx.PolicyVersions.Increment(txCtx, changedBy, reason)
		if err != nil {
			return err
		}
		committedVersion = version.Version
		return policychange.StagePolicyVersionChanged(txCtx, tx.Events, version)
	})
	if err != nil {
		return 0, err
	}
	if changed {
		policychange.ReloadRuntimePolicy(ctx, s.reloader, reason)
	}
	return committedVersion, nil
}

func revokeReason(reason string) string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return "binding revoke"
	}
	return reason
}

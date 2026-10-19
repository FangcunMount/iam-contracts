package assignment

import (
	"context"
	"sort"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	change "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/policychange"
	uow "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/uow"
	domain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/assignment"
	policy "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/policy"
	role "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/role"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
)

// loadReplacement shares deterministic planning between a read-only snapshot
// and the one original write transaction. Only the latter acquires write locks.
func (s *CommandService) loadReplacement(ctx context.Context, cmd ReplaceManagedAssignmentsCommand, repos uow.TxRepositories, locking bool) (map[string]*role.Role, domain.ReplacementPlan, error) {
	fail := func(err error) (map[string]*role.Role, domain.ReplacementPlan, error) {
		return nil, domain.ReplacementPlan{}, err
	}
	validator := domain.NewValidatorWithSubjectResolver(repos.Roles, repos.SubjectResolver)
	if err := validator.CheckSubjectExists(ctx, domain.SubjectType(cmd.Subject.Type), cmd.Subject.ID); err != nil {
		return fail(err)
	}
	managed := make(map[string]*role.Role, len(cmd.ManagedRoleNames))
	ordered := make([]*role.Role, 0, len(cmd.ManagedRoleNames))
	for _, name := range cmd.ManagedRoleNames {
		value, err := repos.Roles.FindByName(ctx, name)
		if err != nil {
			return fail(perrors.Wrap(err, "find managed role"))
		}
		managed[name] = value
		ordered = append(ordered, value)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].ID.Uint64() < ordered[j].ID.Uint64() })
	for _, value := range ordered {
		checked := value
		if locking {
			var err error
			checked, err = repos.Roles.FindByIDForUpdate(ctx, value.ID)
			if err != nil {
				return fail(perrors.Wrap(err, "lock managed role"))
			}
		}
		if err := s.guard.Require(ctx, checked); err != nil {
			return fail(err)
		}
	}
	var current []*domain.Assignment
	var err error
	if locking {
		current, err = repos.Assignments.ListBySubjectForUpdate(ctx, domain.SubjectType(cmd.Subject.Type), cmd.Subject.ID)
	} else {
		current, err = repos.Assignments.ListBySubject(ctx, domain.SubjectType(cmd.Subject.Type), cmd.Subject.ID)
	}
	if err != nil {
		return fail(perrors.Wrap(err, "list subject assignments"))
	}
	if cmd.ExpectedPolicyVersion != 0 {
		if cmd.ExpectedPolicyVersion < 0 {
			return fail(perrors.WithCode(code.ErrInvalidArgument, "invalid expected policy version"))
		}
		var version int64
		if locking {
			locker, ok := repos.PolicyVersions.(policy.LockingRepository)
			if !ok {
				return fail(perrors.New("policy version lock unavailable"))
			}
			row, err := locker.GetCurrentForUpdate(ctx)
			if err != nil {
				return fail(err)
			}
			if row == nil {
				return fail(policy.ErrStaleVersion)
			}
			version = row.Version
		} else {
			var ok bool
			version, ok = change.ExpectedVersion(ctx)
			if !ok {
				return fail(change.ErrInvalidChangePlan)
			}
		}
		if version != cmd.ExpectedPolicyVersion {
			return fail(policy.ErrStaleVersion)
		}
	}
	bindings := make([]domain.ManagedRoleBinding, 0, len(ordered))
	for _, value := range ordered {
		bindings = append(bindings, domain.ManagedRoleBinding{Name: value.Name, ID: value.ID})
	}
	var plan domain.ReplacementPlan
	if cmd.OrgID == 0 {
		plan, err = (domain.ReplacementPolicy{}).Plan(domain.ReplacementRequest{TargetRoleNames: cmd.RoleNames, ManagedRoleNames: cmd.ManagedRoleNames}, bindings, current)
	} else {
		var scoped domain.ScopedReplacementPlan
		scoped, err = (domain.ReplacementPolicy{}).PlanScoped(domain.ScopedReplacementRequest{OrgID: cmd.OrgID, ManagedRoleNames: cmd.ManagedRoleNames, Targets: cmd.ScopedRoles}, bindings, current)
		plan.Revokes, plan.Changed = scoped.Revokes, scoped.Changed
		for _, grant := range scoped.Grants {
			plan.Grants = append(plan.Grants, grant.RoleName)
		}
		for _, target := range cmd.ScopedRoles {
			plan.DirectRoles = append(plan.DirectRoles, target.RoleName)
		}
	}
	if err != nil {
		return fail(err)
	}
	return managed, plan, nil
}

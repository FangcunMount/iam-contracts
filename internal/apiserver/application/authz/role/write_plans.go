package role

import (
	"context"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	change "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/policychange"
	uow "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/uow"
	domain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/role"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
)

const roleOperations = "iam:authz:collection:roles"

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
func (s *RoleCatalog) previewCreate(cmd CreateRoleCommand, created domain.Role) uow.WriteRequest {
	return uow.WriteRequest{Operation: "role.create", Freeze: change.AllocateKeys(change.ArtifactKey("role", created.NameString())), Scope: "global", ChangedBy: cmd.ChangedBy, PermissionResource: roleOperations, PermissionActions: []string{"create"}, Input: map[string]any{"name": created.NameString(), "display_name": created.DisplayName, "description": created.Description, "management_protection": created.ManagementProtection, "changed_by": cmd.ChangedBy}, Preview: func(ctx context.Context, repos uow.TxRepositories) (bool, error) {
		if err := s.guard.RequireOperation(ctx, roleOperations, "create"); err != nil {
			return false, err
		}
		if err := s.guard.Require(ctx, &created); err != nil {
			return false, err
		}
		inspector, ok := repos.Roles.(domain.NameInspector)
		if !ok {
			return false, change.ErrInvalidChangePlan
		}
		exists, err := inspector.ExistsByName(ctx, created.NameString())
		if err != nil {
			return false, err
		}
		if exists {
			return false, perrors.WithCode(code.ErrRoleAlreadyExists, "role already exists")
		}
		return true, nil
	}}
}
func (s *RoleCatalog) previewUpdate(cmd UpdateRoleCommand, result **domain.Role) uow.WriteRequest {
	return uow.WriteRequest{Operation: "role.update", Scope: "global", ChangedBy: cmd.ChangedBy, PermissionResource: roleOperations, PermissionActions: []string{"update"}, Input: map[string]any{"role_id": cmd.ID.String(), "display_name": cmd.DisplayName, "description": cmd.Description, "changed_by": cmd.ChangedBy}, Preview: func(ctx context.Context, repos uow.TxRepositories) (bool, error) {
		if err := s.guard.RequireOperation(ctx, roleOperations, "update"); err != nil {
			return false, err
		}
		before, err := repos.Roles.FindByID(ctx, cmd.ID)
		if err != nil {
			return false, err
		}
		if err = s.guard.Require(ctx, before); err != nil {
			return false, err
		}
		planned := *before
		if cmd.DisplayName != nil {
			if err = planned.Rename(*cmd.DisplayName); err != nil {
				return false, err
			}
		}
		if cmd.Description != nil {
			planned.Description = *cmd.Description
		}
		*result = &planned
		return planned != *before, nil
	}}
}
func (s *RoleCatalog) previewDelete(cmd DeleteRoleCommand) uow.WriteRequest {
	return uow.WriteRequest{Operation: "role.delete", Scope: "global", ChangedBy: cmd.ChangedBy, PermissionResource: roleOperations, PermissionActions: []string{"delete"}, Input: map[string]any{"role_id": cmd.ID.String(), "changed_by": cmd.ChangedBy}, Preview: func(ctx context.Context, repos uow.TxRepositories) (bool, error) {
		if err := s.guard.RequireOperation(ctx, roleOperations, "delete"); err != nil {
			return false, err
		}
		target, err := repos.Roles.FindByID(ctx, cmd.ID)
		if err != nil {
			return false, err
		}
		if err = s.guard.Require(ctx, target); err != nil {
			return false, err
		}
		assignments, err := repos.Assignments.ListByRole(ctx, cmd.ID)
		if err != nil {
			return false, err
		}
		grants, err := repos.PermissionGrants.ListByRole(ctx, cmd.ID)
		if err != nil {
			return false, err
		}
		if err = s.roleRemovalPolicy.EnsureUnused(assignments, grants); err != nil {
			return false, err
		}
		return true, nil
	}}
}

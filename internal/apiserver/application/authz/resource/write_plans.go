package resource

import (
	"context"
	"reflect"
	"slices"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	change "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/policychange"
	uow "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/uow"
	domain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/resource"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
)

const resourceOperations = "iam:authz:collection:resources"

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
func (s *ResourceCatalog) previewCreate(cmd CreateResourceCommand, created domain.Resource) uow.WriteRequest {
	return uow.WriteRequest{Operation: "resource.create", Freeze: change.AllocateKeys(change.ArtifactKey("resource", created.KeyString())), Scope: "global", ChangedBy: cmd.ChangedBy, PermissionResource: resourceOperations, PermissionActions: []string{"create"}, Input: map[string]any{"actor": cmd.Actor.String(), "changed_by": cmd.ChangedBy, "key": created.KeyString(), "display_name": created.DisplayName, "description": created.Description, "actions": created.ActionStrings(), "app": created.AppName, "domain": created.Domain, "type": created.Type}, Preview: func(ctx context.Context, repos uow.TxRepositories) (bool, error) {
		if err := s.requireWrite(ctx, cmd.Actor, "create"); err != nil {
			return false, err
		}
		inspector, ok := repos.Resources.(domain.KeyInspector)
		if !ok {
			return false, change.ErrInvalidChangePlan
		}
		exists, err := inspector.ExistsByKey(ctx, created.KeyString())
		if err != nil {
			return false, err
		}
		if exists {
			return false, perrors.WithCode(code.ErrResourceAlreadyExists, "resource already exists")
		}
		return true, nil
	}}
}
func (s *ResourceCatalog) previewUpdate(cmd UpdateResourceCommand, result **domain.Resource) uow.WriteRequest {
	return uow.WriteRequest{Operation: "resource.update", Scope: "global", ChangedBy: cmd.ChangedBy, PermissionResource: resourceOperations, PermissionActions: []string{"update"}, Input: map[string]any{"actor": cmd.Actor.String(), "resource_id": cmd.ID.String(), "display_name": cmd.DisplayName, "description": cmd.Description, "actions": cmd.Actions, "changed_by": cmd.ChangedBy}, Preview: func(ctx context.Context, repos uow.TxRepositories) (bool, error) {
		if err := s.requireWrite(ctx, cmd.Actor, "update"); err != nil {
			return false, err
		}
		before, err := repos.Resources.FindByID(ctx, cmd.ID)
		if err != nil {
			return false, err
		}
		planned := *before
		planned.Actions = append([]domain.Action(nil), before.Actions...)
		if cmd.DisplayName != nil {
			if err = planned.Rename(*cmd.DisplayName); err != nil {
				return false, err
			}
		}
		if len(cmd.Actions) > 0 {
			if err = planned.ChangeCatalog(cmd.Actions); err != nil {
				return false, err
			}
		}
		if cmd.Description != nil {
			planned.ChangeDescription(*cmd.Description)
		}
		grants, err := repos.PermissionGrants.ListActiveByResource(ctx, cmd.ID)
		if err != nil {
			return false, err
		}
		if err = s.resourceChangePolicy.ValidateDependencies(planned, grants); err != nil {
			return false, err
		}
		*result = &planned
		left, right := planned, *before
		left.Actions = nil
		right.Actions = nil
		same := reflect.DeepEqual(left, right) && slices.Equal(planned.Actions, before.Actions)
		return !same, nil
	}}
}
func (s *ResourceCatalog) previewDelete(cmd DeleteResourceCommand) uow.WriteRequest {
	return uow.WriteRequest{Operation: "resource.delete", Scope: "global", ChangedBy: cmd.ChangedBy, PermissionResource: resourceOperations, PermissionActions: []string{"delete"}, Input: map[string]any{"actor": cmd.Actor.String(), "resource_id": cmd.ID.String(), "changed_by": cmd.ChangedBy}, Preview: func(ctx context.Context, repos uow.TxRepositories) (bool, error) {
		if err := s.requireWrite(ctx, cmd.Actor, "delete"); err != nil {
			return false, err
		}
		if _, err := repos.Resources.FindByID(ctx, cmd.ID); err != nil {
			return false, err
		}
		grants, err := repos.PermissionGrants.ListActiveByResource(ctx, cmd.ID)
		if err != nil {
			return false, err
		}
		if err = s.resourceChangePolicy.EnsureUnused(grants); err != nil {
			return false, err
		}
		return true, nil
	}}
}

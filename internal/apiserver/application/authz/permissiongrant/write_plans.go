package permissiongrant

import (
	"context"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	change "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/policychange"
	uow "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/uow"
	domain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/permissiongrant"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
)

const grantOperations = "iam:authz:collection:permission_grants"

func (s *Service) previewCreate(cmd CreateCommand) uow.WriteRequest {
	return uow.WriteRequest{Operation: "grant.create", Scope: "global", ChangedBy: cmd.GrantedBy, PermissionResource: grantOperations, PermissionActions: []string{"create"}, Input: map[string]any{"role_id": cmd.RoleID.String(), "resource_id": cmd.ResourceID.String(), "action": cmd.Action, "granted_by": cmd.GrantedBy}, Preview: func(ctx context.Context, repos uow.TxRepositories) (bool, error) {
		if err := s.guard.RequireOperation(ctx, grantOperations, "create"); err != nil {
			return false, err
		}
		role, err := repos.Roles.FindByID(ctx, cmd.RoleID)
		if err != nil {
			return false, err
		}
		resource, err := repos.Resources.FindByID(ctx, cmd.ResourceID)
		if err != nil {
			return false, err
		}
		if err = s.guard.Require(ctx, role); err != nil {
			return false, err
		}
		grant, err := domain.New(cmd.RoleID, cmd.ResourceID, resource.KeyString(), cmd.Action, cmd.GrantedBy)
		if err != nil {
			return false, err
		}
		if err = role.ValidateGrant(grant.ResourceKey, grant.Action); err != nil {
			return false, err
		}
		if err = grant.ValidateAgainst(*resource); err != nil {
			return false, err
		}
		reader, ok := repos.PermissionGrants.(domain.PlanningReader)
		if !ok {
			return false, change.ErrInvalidChangePlan
		}
		exists, err := reader.ActiveKeyExists(ctx, grant.GrantKey)
		if err != nil {
			return false, err
		}
		if exists {
			return false, perrors.WithCode(code.ErrPermissionGrantAlreadyExists, "permission grant already exists")
		}
		return true, nil
	}}
}
func (s *Service) previewRevoke(cmd RevokeCommand) uow.WriteRequest {
	return uow.WriteRequest{Operation: "grant.revoke", Scope: "global", ChangedBy: cmd.RevokedBy, PermissionResource: grantOperations, PermissionActions: []string{"revoke"}, Input: map[string]any{"grant_id": cmd.GrantID.String(), "revoked_by": cmd.RevokedBy, "reason": cmd.Reason}, Preview: func(ctx context.Context, repos uow.TxRepositories) (bool, error) {
		if err := s.guard.RequireOperation(ctx, grantOperations, "revoke"); err != nil {
			return false, err
		}
		reader, ok := repos.PermissionGrants.(domain.PlanningReader)
		if !ok {
			return false, change.ErrInvalidChangePlan
		}
		roleID, revoked, err := reader.RevocationStatus(ctx, cmd.GrantID)
		if err != nil {
			return false, err
		}
		target, err := repos.Roles.FindByID(ctx, roleID)
		if err != nil {
			return false, err
		}
		if err = s.guard.Require(ctx, target); err != nil {
			return false, err
		}
		return !revoked, nil
	}}
}

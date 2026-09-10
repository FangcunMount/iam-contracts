package policy

import (
	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/assignment"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/permissiongrant"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
)

// RoleRemovalPolicy validates whether a role can be removed safely.
type RoleRemovalPolicy struct{}

// EnsureUnused rejects removal while active assignments or grants remain.
func (RoleRemovalPolicy) EnsureUnused(
	assignments []*assignment.Assignment,
	grants []*permissiongrant.Grant,
) error {
	if len(assignments) > 0 {
		return perrors.WithCode(code.ErrRoleInUse, "role has %d active assignments", len(assignments))
	}
	for _, grant := range grants {
		if grant != nil && grant.IsActive() {
			return perrors.WithCode(code.ErrRoleInUse, "role has active permission grants")
		}
	}
	return nil
}

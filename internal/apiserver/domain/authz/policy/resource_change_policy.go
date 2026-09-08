package policy

import (
	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/permissiongrant"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/resource"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
)

// ResourceChangePolicy validates catalog changes against active permission grants.
type ResourceChangePolicy struct{}

// ValidateDependencies proves that every active grant depending on a catalog
// resource remains valid against the candidate catalog definition.
func (ResourceChangePolicy) ValidateDependencies(candidate resource.Resource, grants []*permissiongrant.Grant) error {
	for _, grant := range grants {
		if grant == nil || !grant.IsActive() {
			continue
		}
		if err := grant.ValidateAgainst(candidate); err != nil {
			return perrors.WithCode(code.ErrResourceInUse, "resource change would invalidate permission grant %s: %v", grant.ID.String(), err)
		}
	}
	return nil
}

// EnsureUnused rejects deletion when active grants still reference the resource.
func (ResourceChangePolicy) EnsureUnused(grants []*permissiongrant.Grant) error {
	active := 0
	for _, grant := range grants {
		if grant != nil && grant.IsActive() {
			active++
		}
	}
	if active > 0 {
		return perrors.WithCode(code.ErrResourceInUse, "resource has %d active permission grants", active)
	}
	return nil
}

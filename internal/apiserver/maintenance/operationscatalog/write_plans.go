package operationscatalog

import (
	"context"
	"fmt"
	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/operation"
	change "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/policychange"
	grant "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/permissiongrant"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/resource"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"

	"github.com/FangcunMount/iam/v5/internal/apiserver/maintenance/policywrite"
	"github.com/FangcunMount/iam/v5/internal/apiserver/maintenance/rolemodel"
	"gorm.io/gorm"
)

func requireActor(tx *gorm.DB, actor string) error {
	var protected int64
	err := tx.Table("authz_assignments a").Joins("JOIN authz_roles r ON r.id=a.role_id").Joins("JOIN users u ON u.id=a.subject_id").Where("a.subject_type='user' AND a.subject_id=? AND a.deleted_at IS NULL AND r.deleted_at IS NULL AND r.name='platform_admin' AND r.management_protection='protected' AND u.deleted_at IS NULL AND u.status=1", actor).Count(&protected).Error
	if err != nil {
		return err
	}
	if protected != 1 {
		return fmt.Errorf("active protected platform administrator required")
	}
	return nil
}
func applyRequest(fp, actor string, result *Report) policywrite.Request {
	var planned Report
	var state rolemodel.State
	return policywrite.Request{Result: operation.ProjectResult("iam.maintenance.operations-catalog/v1", func() (Report, error) { return *result, nil }, func(saved Report) error { *result = saved; return nil }), Operation: "maintenance.operations_catalog.apply", Freeze: func(b *change.ArtifactBuilder) error {
		id, err := meta.ParseID(planned.ResourceID)
		if planned.CreateResource {
			id, err = b.ID(change.ArtifactKey("resource", ResourceKey))
		}
		if err != nil {
			return err
		}
		for _, name := range planned.MissingRoles {
			for _, r := range state.Roles {
				if r.Name != name || r.DeletedAt != nil {
					continue
				}
				g, e := grant.New(r.ID, resource.NewResourceID(id.Uint64()), ResourceKey, "read", "user:"+actor)
				if e != nil {
					return e
				}
				if _, e = b.ID(change.ArtifactKey("grant", g.GrantKey)); e != nil {
					return e
				}
			}
		}
		return nil
	}, ChangedBy: "user:" + actor, Input: map[string]any{"resource_key": ResourceKey, "actor_id": actor, "fingerprint": fp}, Preview: func(ctx context.Context, tx *gorm.DB) (bool, error) {
		before, err := rolemodel.LoadState(ctx, tx)
		if err != nil {
			return false, err
		}
		plan, err := inspect(before)
		if err != nil {
			return false, err
		}
		if plan.Fingerprint != fp {
			return false, fmt.Errorf("authorization facts changed; repeat preflight")
		}
		if err = requireActor(tx, actor); err != nil {
			return false, err
		}
		if plan.State == "configured" {
			*result = plan
			return false, nil
		}
		planned, state = plan, before
		return true, nil
	}}
}

package conditionretire

import (
	"context"
	"fmt"
	change "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/policychange"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/resource"
	"github.com/FangcunMount/iam/v5/internal/apiserver/maintenance/legacycondition/constraint"
	legacygrant "github.com/FangcunMount/iam/v5/internal/apiserver/maintenance/legacycondition/grant"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"

	"github.com/FangcunMount/iam/v5/internal/apiserver/maintenance/policywrite"
	"gorm.io/gorm"
)

func mutationRequest(fp string, rollback bool, result **Receipt) policywrite.Request {
	operation := "maintenance.condition_retire.apply"
	if rollback {
		operation = "maintenance.condition_retire.rollback"
	}
	var createKeys []string
	return policywrite.Request{Result: receiptResultCodec(result), Operation: operation, Freeze: func(b *change.ArtifactBuilder) error { return change.AllocateKeys(createKeys...)(b) }, ChangedBy: MigrationID, Input: map[string]any{"migration_id": MigrationID, "fingerprint": fp}, Preview: func(ctx context.Context, tx *gorm.DB) (bool, error) {
		report, saved, before, err := inspect(ctx, tx)
		if err != nil {
			return false, err
		}
		if saved != nil {
			if saved.Fingerprint != fp {
				return false, fmt.Errorf("fingerprint mismatch")
			}
			if report.State != "applied_unchanged" {
				return false, fmt.Errorf("migration conflicts with current facts: %s", report.State)
			}
			if rollback {
				return true, nil
			}
			*result = saved
			return false, nil
		}
		if rollback {
			return false, fmt.Errorf("no retirement receipt to roll back")
		}
		plan := Build(before)
		if err = plan.Validate(); err != nil {
			return false, err
		}
		if plan.Fingerprint != fp {
			return false, fmt.Errorf("preflight facts changed")
		}
		createKeys = nil
		for _, c := range plan.Grants {
			if c.ExistingID != "" {
				continue
			}
			for _, old := range before.Grants {
				if old.ID.String() != c.OldID {
					continue
				}
				if old.ResourceID == nil {
					return false, change.ErrInvalidChangePlan
				}
				grant, e := legacygrant.New(meta.FromUint64(old.RoleID), resource.NewResourceID(*old.ResourceID), old.ResourcePattern, old.Action, constraint.Empty(), MigrationID)
				if e != nil {
					return false, e
				}
				createKeys = append(createKeys, change.ArtifactKey("grant", grant.GrantKey))
			}
		}
		return true, nil
	}}
}

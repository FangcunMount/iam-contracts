package conditionretire

import (
	"context"
	"fmt"

	"github.com/FangcunMount/iam/v5/internal/apiserver/maintenance/policywrite"
	"gorm.io/gorm"
)

func mutationRequest(fp string, rollback bool, result **Receipt) policywrite.Request {
	operation := "maintenance.condition_retire.apply"
	if rollback {
		operation = "maintenance.condition_retire.rollback"
	}
	return policywrite.Request{Operation: operation, ChangedBy: MigrationID, Input: map[string]any{"migration_id": MigrationID, "fingerprint": fp}, Preview: func(ctx context.Context, tx *gorm.DB) (bool, error) {
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
		return true, nil
	}}
}

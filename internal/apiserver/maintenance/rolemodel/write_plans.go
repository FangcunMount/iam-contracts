package rolemodel

import (
	"context"
	"errors"
	"fmt"

	"github.com/FangcunMount/iam/v5/internal/apiserver/maintenance/policywrite"
	"gorm.io/gorm"
)

func applyRequest(qs *gorm.DB, fp string, result *Receipt) policywrite.Request {
	return policywrite.Request{Operation: "maintenance.role_model.apply", ChangedBy: MigrationID, Input: map[string]any{"migration_id": MigrationID, "fingerprint": fp}, Preview: func(ctx context.Context, tx *gorm.DB) (bool, error) {
		var saved Receipt
		err := tx.WithContext(ctx).First(&saved, "migration_id = ?", MigrationID).Error
		if err == nil {
			if saved.Fingerprint != fp || saved.Status != "applied" {
				return false, fmt.Errorf("migration receipt conflicts")
			}
			status, e := inspectStatus(ctx, tx)
			if e != nil {
				return false, e
			}
			if status.State != "applied_unchanged" {
				return false, fmt.Errorf("applied migration facts changed")
			}
			*result = saved
			return false, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return false, err
		}
		before, err := LoadState(ctx, tx)
		if err != nil {
			return false, err
		}
		people, err := ReadPeople(ctx, tx, qs, before)
		if err != nil {
			return false, err
		}
		input, err := before.Input(people)
		if err != nil {
			return false, err
		}
		plan := Build(input)
		if err = plan.Validate(); err != nil {
			return false, err
		}
		if plan.Fingerprint != fp {
			return false, fmt.Errorf("preflight facts changed")
		}
		return true, nil
	}}
}
func rollbackRequest(fp string, result *Receipt) policywrite.Request {
	return policywrite.Request{Operation: "maintenance.role_model.rollback", ChangedBy: MigrationID, Input: map[string]any{"migration_id": MigrationID, "fingerprint": fp}, Preview: func(ctx context.Context, tx *gorm.DB) (bool, error) {
		var saved Receipt
		if err := tx.WithContext(ctx).First(&saved, "migration_id = ?", MigrationID).Error; err != nil {
			return false, err
		}
		if saved.Fingerprint != fp {
			return false, fmt.Errorf("rollback fingerprint mismatch")
		}
		if saved.Status == "rolled_back" {
			*result = saved
			return false, nil
		}
		if saved.Status != "applied" {
			return false, fmt.Errorf("unexpected migration status")
		}
		if !tx.Migrator().HasTable(&LegacyEdge{}) {
			return false, fmt.Errorf("restore archived inheritance table schema and rows before rollback")
		}
		current, err := LoadState(ctx, tx)
		if err != nil {
			return false, err
		}
		if current.Hash() != saved.AfterHash {
			return false, fmt.Errorf("rollback conflicts with subsequent authorization writes")
		}
		return true, nil
	}}
}

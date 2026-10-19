package scopemigrate

import (
	"context"
	"errors"
	"fmt"
	change "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/policychange"

	"github.com/FangcunMount/iam/v5/internal/apiserver/maintenance/policywrite"
	"github.com/FangcunMount/iam/v5/internal/apiserver/maintenance/rolemodel"
	"gorm.io/gorm"
)

func cloneInput(input Input) Input {
	input.Mappings = append([]Mapping(nil), input.Mappings...)
	for i := range input.Mappings {
		m := &input.Mappings[i]
		m.Targets = append([]Target(nil), m.Targets...)
		for j := range m.Targets {
			m.Targets[j].StoreIDs = append([]string(nil), m.Targets[j].StoreIDs...)
		}
	}
	return input
}
func readSnapshot(ctx context.Context, tx, qs *gorm.DB) (Snapshot, error) {
	state, err := rolemodel.LoadState(ctx, tx)
	if err != nil {
		return Snapshot{}, err
	}
	facts, err := LoadBusinessFacts(ctx, qs)
	return Snapshot{IAM: state, QS: facts}, err
}
func applyRequest(qs *gorm.DB, id, actor, fp string, input Input, result **Receipt) policywrite.Request {
	var createKeys []string
	return policywrite.Request{Operation: "maintenance.scope.apply", Freeze: func(b *change.ArtifactBuilder) error { return change.AllocateKeys(createKeys...)(b) }, ChangedBy: "user:" + actor, Input: map[string]any{"migration_id": id, "actor": actor, "fingerprint": fp, "mapping": input}, Preview: func(ctx context.Context, tx *gorm.DB) (bool, error) {
		before, err := readSnapshot(ctx, tx, qs)
		if err != nil {
			return false, err
		}
		var saved Receipt
		err = tx.WithContext(ctx).Where("migration_id = ?", id).Take(&saved).Error
		if err == nil {
			report, e := inspect(before, id, &saved)
			if e != nil {
				return false, e
			}
			if report.State != "applied_unchanged" || saved.Fingerprint != fp {
				return false, fmt.Errorf("scope migration conflicts: %s", report.State)
			}
			*result = &saved
			return false, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return false, err
		}
		plan := Build(before.IAM, before.QS, input)
		if err = plan.Validate(); err != nil {
			return false, err
		}
		if plan.Fingerprint != fp {
			return false, fmt.Errorf("reviewed facts changed")
		}
		createKeys = nil
		for _, c := range plan.Changes {
			for i, target := range c.Targets {
				if i > 0 {
					createKeys = append(createKeys, change.ArtifactKey("assignment", "user", c.SubjectID, c.RoleID, target.OrgID))
				}
			}
		}
		return true, nil
	}}
}
func rollbackRequest(qs *gorm.DB, id, actor, fp string, result **Receipt) policywrite.Request {
	return policywrite.Request{Operation: "maintenance.scope.rollback", ChangedBy: "user:" + actor, Input: map[string]any{"migration_id": id, "actor": actor, "fingerprint": fp}, Preview: func(ctx context.Context, tx *gorm.DB) (bool, error) {
		current, err := readSnapshot(ctx, tx, qs)
		if err != nil {
			return false, err
		}
		var saved Receipt
		if err = tx.WithContext(ctx).Where("migration_id = ?", id).Take(&saved).Error; err != nil {
			return false, err
		}
		report, err := inspect(current, id, &saved)
		if err != nil {
			return false, err
		}
		if saved.Fingerprint != fp {
			return false, fmt.Errorf("rollback fingerprint mismatch")
		}
		if report.State == "rolled_back" && current.Hash() == saved.RollbackHash {
			*result = &saved
			return false, nil
		}
		if report.State != "applied_unchanged" {
			return false, fmt.Errorf("rollback conflicts with current facts: %s", report.State)
		}
		return true, nil
	}}
}

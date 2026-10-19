package rolemodel

import (
	"context"
	"errors"
	"fmt"
	change "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/policychange"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/resource"
	resourcepo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/resource"
	rolepo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/role"
	"github.com/FangcunMount/iam/v5/internal/apiserver/maintenance/legacycondition/constraint"
	legacygrant "github.com/FangcunMount/iam/v5/internal/apiserver/maintenance/legacycondition/grant"

	"github.com/FangcunMount/iam/v5/internal/apiserver/maintenance/policywrite"
	"gorm.io/gorm"
)

func applyRequest(qs *gorm.DB, fp string, result *Receipt) policywrite.Request {
	var frozenState State
	var frozenPlan Plan
	return policywrite.Request{Result: receiptResultCodec(result), Operation: "maintenance.role_model.apply", Freeze: func(b *change.ArtifactBuilder) error { return allocateRows(frozenState, frozenPlan, b) }, ChangedBy: MigrationID, Input: map[string]any{"migration_id": MigrationID, "fingerprint": fp}, Preview: func(ctx context.Context, tx *gorm.DB) (bool, error) {
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
		frozenState, frozenPlan = before, plan
		return true, nil
	}}
}
func rollbackRequest(fp string, result *Receipt) policywrite.Request {
	return policywrite.Request{Result: receiptResultCodec(result), Operation: "maintenance.role_model.rollback", ChangedBy: MigrationID, Input: map[string]any{"migration_id": MigrationID, "fingerprint": fp}, Preview: func(ctx context.Context, tx *gorm.DB) (bool, error) {
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

func allocateRows(before State, p Plan, builder *change.ArtifactBuilder) error {
	roles := map[string]rolepo.RolePO{}
	resources := map[string]resourcepo.ResourcePO{}
	for _, r := range before.Roles {
		if r.DeletedAt == nil {
			roles[r.Name] = r
		}
	}
	for _, r := range before.Resources {
		if r.DeletedAt == nil {
			resources[r.Key] = r
		}
	}
	for _, name := range []string{Operator, Reviewer} {
		id, err := builder.ID(change.ArtifactKey("role", name))
		if err != nil {
			return err
		}
		r := rolepo.RolePO{Name: name, ManagementProtection: "standard"}
		r.ID = id
		roles[name] = r
	}
	for _, name := range sortedGrantRoles(p.Grants) {
		r := roles[name]
		for _, permission := range p.Grants[name] {
			conditions := constraint.Empty()
			if permission.Origin != "" {
				var err error
				conditions, err = constraint.New(constraint.Equal("object.origin_type", constraint.StringValue(permission.Origin)))
				if err != nil {
					return err
				}
			}
			resourceID := resource.NewResourceID(0)
			if catalog, ok := resources[permission.Resource]; ok {
				resourceID = resource.NewResourceID(catalog.ID.Uint64())
			}
			var grant legacygrant.Grant
			var err error
			if resourceID.Uint64() == 0 || permission.Action == "*" {
				grant, err = legacygrant.NewSystem(r.ID, resourceID, permission.Resource, permission.Action, conditions, MigrationID)
			} else {
				grant, err = legacygrant.New(r.ID, resourceID, permission.Resource, permission.Action, conditions, MigrationID)
			}
			if err != nil {
				return err
			}
			found := false
			for _, old := range before.Grants {
				if old.DeletedAt == nil && old.RevokedAt == nil && old.GrantKey == grant.GrantKey {
					found = true
					break
				}
			}
			if !found {
				if _, err = builder.ID(change.ArtifactKey("grant", grant.GrantKey)); err != nil {
					return err
				}
			}
		}
	}
	for _, person := range p.People {
		for _, name := range person.After {
			r := roles[name]
			found := false
			for _, a := range before.Assignments {
				if a.DeletedAt == nil && a.SubjectType+":"+a.SubjectID == person.Subject && a.RoleID == r.ID.Uint64() {
					found = true
					break
				}
			}
			if !found {
				if _, err := builder.ID(change.ArtifactKey("assignment", "user", person.Subject[len("user:"):], r.ID.String(), "0")); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

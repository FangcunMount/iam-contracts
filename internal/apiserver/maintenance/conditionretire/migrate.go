package conditionretire

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	legacygrant "github.com/FangcunMount/iam/v5/internal/apiserver/maintenance/legacycondition/grant"

	policydomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/policy"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/resource"
	grantpo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/permissiongrant"
	policypo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/policy"
	"github.com/FangcunMount/iam/v5/internal/apiserver/maintenance/legacycondition/constraint"
	grantdomain "github.com/FangcunMount/iam/v5/internal/apiserver/maintenance/legacycondition/grant"
	"github.com/FangcunMount/iam/v5/internal/apiserver/maintenance/rolemodel"
	dbmysql "github.com/FangcunMount/iam/v5/internal/pkg/database/mysql"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"github.com/FangcunMount/iam/v5/pkg/event"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Receipt struct {
	MigrationID string    `json:"migration_id" gorm:"primaryKey;size:64"`
	Fingerprint string    `json:"fingerprint"`
	AfterHash   string    `json:"after_hash"`
	Status      string    `json:"status"`
	BeforeJSON  string    `json:"-" gorm:"type:longtext"`
	AfterJSON   string    `json:"-" gorm:"type:longtext"`
	PlanJSON    string    `json:"-" gorm:"type:longtext"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Receipt) TableName() string { return "iam_condition_retirement_migrations" }

type Report struct {
	State              string   `json:"state"`
	NextAction         string   `json:"next_action"`
	MigrationID        string   `json:"migration_id"`
	Fingerprint        string   `json:"fingerprint"`
	AfterHash          string   `json:"after_hash"`
	CurrentFingerprint string   `json:"current_fingerprint"`
	Plan               *Plan    `json:"plan,omitempty"`
	Differences        []string `json:"differences,omitempty"`
}

func encode(v any) string { b, _ := json.Marshal(v); return string(b) }
func inspect(ctx context.Context, db *gorm.DB) (Report, *Receipt, rolemodel.State, error) {
	r := Report{MigrationID: MigrationID, State: "invalid", NextAction: "stop"}
	s, err := rolemodel.LoadState(ctx, db)
	if err != nil {
		return r, nil, s, err
	}
	r.CurrentFingerprint = s.Hash()
	if !db.Migrator().HasTable(&Receipt{}) {
		r.State = "pending"
		r.NextAction = "preflight"
		return r, nil, s, nil
	}
	var receipt Receipt
	err = db.WithContext(ctx).Where("migration_id = ?", MigrationID).Take(&receipt).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		r.State = "pending"
		r.NextAction = "preflight"
		return r, nil, s, nil
	}
	if err != nil {
		return r, nil, s, err
	}
	r.Fingerprint = receipt.Fingerprint
	r.AfterHash = receipt.AfterHash
	var before, after rolemodel.State
	var p Plan
	if json.Unmarshal([]byte(receipt.BeforeJSON), &before) != nil || json.Unmarshal([]byte(receipt.AfterJSON), &after) != nil || json.Unmarshal([]byte(receipt.PlanJSON), &p) != nil || before.Hash() != receipt.Fingerprint || after.Hash() != receipt.AfterHash || encode(Build(before)) != encode(p) || p.Validate() != nil {
		return r, &receipt, s, fmt.Errorf("invalid retirement receipt")
	}
	r.Plan = &p
	switch receipt.Status {
	case "rolled_back":
		r.State = "rolled_back"
		r.NextAction = "review"
	case "applied":
		if s.Hash() == receipt.AfterHash {
			r.State = "applied_unchanged"
			r.NextAction = "verify"
		} else {
			r.State = "applied_drifted"
			r.NextAction = "review"
			r.Differences = diff(after, s)
		}
	default:
		return r, &receipt, s, fmt.Errorf("invalid retirement receipt status")
	}
	return r, &receipt, s, nil
}
func diff(a, b rolemodel.State) []string {
	out := []string{}
	for _, x := range []struct {
		name string
		a, b any
	}{{"roles", a.Roles, b.Roles}, {"assignments", a.Assignments, b.Assignments}, {"grants", a.Grants, b.Grants}, {"resources", a.Resources, b.Resources}, {"inheritance_history", a.Edges, b.Edges}, {"policy_version", a.PolicyVersion, b.PolicyVersion}} {
		if encode(x.a) != encode(x.b) {
			out = append(out, x.name)
		}
	}
	return out
}
func Status(ctx context.Context, db *gorm.DB) (Report, error) {
	var r Report
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { var err error; r, _, _, err = inspect(ctx, tx); return err }, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	return r, err
}
func Preflight(ctx context.Context, db *gorm.DB) (Report, error) {
	var r Report
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var s rolemodel.State
		var err error
		r, _, s, err = inspect(ctx, tx)
		if err != nil {
			return err
		}
		if r.State == "applied_unchanged" {
			return nil
		}
		if r.State != "pending" {
			return fmt.Errorf("migration is not executable: %s", r.State)
		}
		p := Build(s)
		r.Plan = &p
		r.Fingerprint = p.Fingerprint
		if err = p.Validate(); err != nil {
			return err
		}
		r.NextAction = "apply"
		return nil
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	return r, err
}
func Verify(ctx context.Context, db *gorm.DB) (Report, error) {
	r, err := Status(ctx, db)
	if err != nil {
		return r, err
	}
	if r.State != "applied_unchanged" {
		return r, fmt.Errorf("verification requires unchanged applied facts")
	}
	return r, nil
}
func Apply(ctx context.Context, db *gorm.DB, stager event.Stager, fp string, stopped bool) (*Receipt, error) {
	return mutate(ctx, db, stager, fp, stopped, false)
}
func Rollback(ctx context.Context, db *gorm.DB, stager event.Stager, fp string, stopped bool) (*Receipt, error) {
	return mutate(ctx, db, stager, fp, stopped, true)
}
func mutate(ctx context.Context, db *gorm.DB, stager event.Stager, fp string, stopped, rollback bool) (*Receipt, error) {
	if !stopped || len(fp) != 64 {
		return nil, fmt.Errorf("requires stopped authorization writers and reviewed fingerprint")
	}
	if !db.Migrator().HasTable(&Receipt{}) {
		return nil, fmt.Errorf("install condition retirement receipt schema first")
	}
	var result *Receipt
	err := dbmysql.NewUnitOfWork(db).WithinTransaction(ctx, func(txCtx context.Context) error {
		tx, err := dbmysql.RequireTx(txCtx)
		if err != nil {
			return err
		}
		q := tx.Table("authz_policy_versions")
		if tx.Dialector.Name() != "sqlite" {
			q = q.Clauses(clause.Locking{Strength: "UPDATE"})
		}
		var row struct{ ID uint64 }
		if err = q.Where("id = ?", 1).Take(&row).Error; err != nil {
			return err
		}
		report, receipt, before, err := inspect(txCtx, tx)
		if err != nil {
			return err
		}
		if receipt != nil {
			if receipt.Fingerprint != fp {
				return fmt.Errorf("fingerprint mismatch")
			}
			if rollback {
				if report.State != "applied_unchanged" {
					return fmt.Errorf("rollback conflicts with current facts: %s", report.State)
				}
				if err = restore(tx, receipt); err != nil {
					return err
				}
				if err = advance(txCtx, tx, stager, "condition retirement rolled back"); err != nil {
					return err
				}
				if err = tx.Model(&Receipt{}).Where("migration_id = ?", MigrationID).Updates(map[string]any{"status": "rolled_back", "updated_at": time.Now().UTC()}).Error; err != nil {
					return err
				}
				receipt.Status = "rolled_back"
				result = receipt
				return nil
			}
			if report.State != "applied_unchanged" {
				return fmt.Errorf("migration conflicts with current facts: %s", report.State)
			}
			result = receipt
			return nil
		}
		if rollback {
			return fmt.Errorf("no retirement receipt to roll back")
		}
		p := Build(before)
		if err = p.Validate(); err != nil {
			return err
		}
		if p.Fingerprint != fp {
			return fmt.Errorf("preflight facts changed")
		}
		if err = apply(tx, before, p); err != nil {
			return err
		}
		if err = advance(txCtx, tx, stager, "condition authorization retired"); err != nil {
			return err
		}
		after, err := rolemodel.LoadState(txCtx, tx)
		if err != nil {
			return err
		}
		check := Build(after)
		if check.Validate() != nil || len(check.Grants) != 0 || len(check.Resources) != 0 {
			return fmt.Errorf("conditional facts remain after retirement")
		}
		now := time.Now().UTC()
		result = &Receipt{MigrationID: MigrationID, Fingerprint: fp, AfterHash: after.Hash(), Status: "applied", BeforeJSON: encode(before), AfterJSON: encode(after), PlanJSON: encode(p), CreatedAt: now, UpdatedAt: now}
		return tx.Create(result).Error
	})
	return result, err
}
func advance(ctx context.Context, tx *gorm.DB, stager event.Stager, reason string) error {
	if stager == nil {
		return fmt.Errorf("durable policy event stager required")
	}
	v, err := policypo.NewPolicyVersionRepository(tx).Increment(ctx, MigrationID, reason)
	if err != nil {
		return err
	}
	return stager.Stage(ctx, policydomain.NewVersionChangedEvent(v.Version))
}
func apply(tx *gorm.DB, s rolemodel.State, p Plan) error {
	now := time.Now().UTC()
	grants := map[string]grantpo.GrantPO{}
	for _, g := range s.Grants {
		grants[g.ID.String()] = g
	}
	for _, c := range p.Grants {
		old := grants[c.OldID]
		if err := tx.Table("authz_permission_grants").Where("id = ?", c.OldID).Updates(map[string]any{"revoked_at": now, "updated_at": now, "version": gorm.Expr("version + 1")}).Error; err != nil {
			return err
		}
		if c.ExistingID != "" {
			continue
		}
		b, err := grantdomain.New(meta.FromUint64(old.RoleID), resource.NewResourceID(*old.ResourceID), old.ResourcePattern, old.Action, constraint.Empty(), MigrationID)
		if err != nil {
			return err
		}
		po, err := (legacygrant.Mapper{}).ToPO(&b)
		if err != nil {
			return err
		}
		if err = tx.Create(po).Error; err != nil {
			return err
		}
	}
	for _, id := range p.Resources {
		if err := tx.Table("authz_resources").Where("id = ?", id).Updates(map[string]any{"attribute_schema": emptySchema, "updated_at": now, "version": gorm.Expr("version + 1")}).Error; err != nil {
			return err
		}
	}
	return nil
}
func restore(tx *gorm.DB, r *Receipt) error {
	var before, after rolemodel.State
	var p Plan
	if err := json.Unmarshal([]byte(r.BeforeJSON), &before); err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(r.AfterJSON), &after); err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(r.PlanJSON), &p); err != nil {
		return err
	}
	old := map[string]grantpo.GrantPO{}
	for _, g := range before.Grants {
		old[g.ID.String()] = g
	}
	now := time.Now().UTC()
	// Revoke new rows first to release the active unique key. Never delete audit rows.
	for _, g := range after.Grants {
		if _, exists := old[g.ID.String()]; !exists {
			if err := tx.Table("authz_permission_grants").Where("id = ?", g.ID).Updates(map[string]any{"revoked_at": now, "updated_at": now, "version": gorm.Expr("version + 1")}).Error; err != nil {
				return err
			}
		}
	}
	for _, c := range p.Grants {
		g := old[c.OldID]
		if err := tx.Table("authz_permission_grants").Where("id = ?", c.OldID).Updates(map[string]any{"revoked_at": g.RevokedAt, "updated_at": now, "version": gorm.Expr("version + 1")}).Error; err != nil {
			return err
		}
	}
	for _, id := range p.Resources {
		for _, res := range before.Resources {
			if res.ID.String() == id {
				if err := tx.Table("authz_resources").Where("id = ?", id).Updates(map[string]any{"attribute_schema": res.AttributeSchema, "updated_at": now, "version": gorm.Expr("version + 1")}).Error; err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// Bootstrap is only called by the fresh-database migration stages, after the
// migrator has established that the database was empty before initialization.
func Bootstrap(ctx context.Context, db *gorm.DB, s event.Stager) error {
	p, err := Preflight(ctx, db)
	if err != nil {
		return err
	}
	_, err = Apply(ctx, db, s, p.Fingerprint, true)
	return err
}

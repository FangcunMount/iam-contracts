package rolemodel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/constraint"
	grantdomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/permissiongrant"
	policydomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/policy"
	resourcedomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/resource"
	roledomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/role"
	assignmentpo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/assignment"
	grantpo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/permissiongrant"
	policypo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/policy"
	resourcepo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/resource"
	rolepo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/role"
	dbmysql "github.com/FangcunMount/iam/v5/internal/pkg/database/mysql"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"github.com/FangcunMount/iam/v5/pkg/event"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Receipt struct {
	MigrationID string    `json:"migration_id" gorm:"primaryKey;size:64"`
	Fingerprint string    `json:"fingerprint" gorm:"size:64"`
	AfterHash   string    `json:"after_hash" gorm:"size:64"`
	Status      string    `json:"status"`
	BeforeJSON  string    `json:"-" gorm:"column:before_json;type:longtext"`
	AfterJSON   string    `json:"-" gorm:"column:after_json;type:longtext"`
	PlanJSON    string    `json:"-" gorm:"column:plan_json;type:longtext"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Receipt) TableName() string { return "iam_role_model_migrations" }

// Apply is an offline, all-or-nothing data cutover. The caller must stop all
// authorization and relevant identity writers; a fingerprint is not a lock on
// the separate QS database. Both sources are re-read inside this operation.
func Apply(ctx context.Context, iam, qs *gorm.DB, stager event.Stager, fingerprint string, writesStopped bool) (*Receipt, error) {
	if !writesStopped || len(fingerprint) != 64 {
		return nil, fmt.Errorf("apply requires stopped writers and the reviewed preflight fingerprint")
	}
	if !iam.Migrator().HasTable(&Receipt{}) {
		return nil, fmt.Errorf("role migration receipt schema must be installed first")
	}
	var receipt Receipt
	err := dbmysql.NewUnitOfWork(iam).WithinTransaction(ctx, func(txCtx context.Context) error {
		tx, err := dbmysql.RequireTx(txCtx)
		if err != nil {
			return err
		}
		if err = lockPolicy(tx); err != nil {
			return err
		}
		err = tx.Where("migration_id = ?", MigrationID).First(&receipt).Error
		if err == nil {
			if receipt.Fingerprint != fingerprint || receipt.Status != "applied" {
				return fmt.Errorf("migration receipt conflicts; inspect or roll back before retrying")
			}
			status, err := Status(txCtx, tx)
			if err != nil {
				return err
			}
			if status.State != "applied_unchanged" {
				return fmt.Errorf("applied migration facts changed")
			}
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		before, err := LoadState(txCtx, tx)
		if err != nil {
			return err
		}
		people, err := ReadPeople(txCtx, tx, qs, before)
		if err != nil {
			return err
		}
		in, err := before.Input(people)
		if err != nil {
			return err
		}
		plan := Build(in)
		if err = plan.Validate(); err != nil {
			return err
		}
		if plan.Fingerprint != fingerprint {
			return fmt.Errorf("preflight facts changed; regenerate and review the plan")
		}
		if err = applyPlan(txCtx, tx, before, plan); err != nil {
			return err
		}
		if err = advance(txCtx, tx, stager, "role model migrated"); err != nil {
			return err
		}
		after, err := LoadState(txCtx, tx)
		if err != nil {
			return err
		}
		if err = verifyPlan(after, plan); err != nil {
			return err
		}
		receipt = Receipt{MigrationID: MigrationID, Fingerprint: fingerprint, AfterHash: after.Hash(), Status: "applied", BeforeJSON: encode(before), AfterJSON: encode(after), PlanJSON: encode(plan), CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
		return tx.Create(&receipt).Error
	})
	return &receipt, err
}
func lockPolicy(tx *gorm.DB) error {
	q := tx.Table("authz_policy_versions")
	if tx.Dialector.Name() != "sqlite" {
		q = q.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var row struct{ ID uint64 }
	return q.Where("id = ?", 1).Take(&row).Error
}
func encode(v any) string { raw, _ := json.Marshal(v); return string(raw) }
func advance(ctx context.Context, tx *gorm.DB, stager event.Stager, reason string) error {
	if stager == nil {
		return fmt.Errorf("durable policy event stager is required")
	}
	v, err := policypo.NewPolicyVersionRepository(tx).Increment(ctx, MigrationID, reason)
	if err != nil {
		return err
	}
	return stager.Stage(ctx, policydomain.NewVersionChangedEvent(v.Version))
}
func applyPlan(ctx context.Context, tx *gorm.DB, before State, p Plan) error {
	now := time.Now().UTC()
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
	for _, spec := range []struct{ name, label, description string }{{Operator, "测评运营员", "组织测评、跟进进度、批量执行及临时测评重试；不包含专业结果读取"}, {Reviewer, "测评结果评估员", "在业务访问范围内查看答卷、评分、报告并分析结果；不包含执行及重试"}} {
		r := rolepo.RolePO{Name: spec.name, DisplayName: spec.label, Description: spec.description, ManagementProtection: "standard", IsSystem: 1}
		if err := tx.Create(&r).Error; err != nil {
			return err
		}
		roles[r.Name] = r
	}
	for name, label := range map[string]string{"platform_admin": "平台根管理员", "iam_admin": "身份与授权管理员", "qs:admin": "QS 应用管理员", "qs:content_manager": "测评内容管理员", "qs:evaluation_plan_manager": "测评计划管理员"} {
		r := roles[name]
		description := map[string]string{"platform_admin": "全局根管理与受保护能力管理", "iam_admin": "身份与普通授权管理，不包含平台受保护能力", "qs:admin": "QS 当前及后续资源的完整管理", "qs:content_manager": "问卷、量表和常模维护及发布", "qs:evaluation_plan_manager": "测评计划、任务与进度管理及计划测评重试"}[name]
		if err := tx.Table("authz_roles").Where("id = ?", r.ID).Updates(map[string]any{"display_name": label, "description": description, "updated_at": now, "version": gorm.Expr("version + 1")}).Error; err != nil {
			return err
		}
	}
	r := resources[Assessment]
	var actions []string
	if err := json.Unmarshal([]byte(r.Actions), &actions); err != nil {
		return err
	}
	actions = uniqueStrings(append(actions, "read_progress", "list_progress"))
	r.Actions = encode(actions)
	if err := tx.Table("authz_resources").Where("id = ?", r.ID).Updates(map[string]any{"actions": r.Actions, "updated_at": now, "version": gorm.Expr("version + 1")}).Error; err != nil {
		return err
	}
	resources[Assessment] = r
	desired := map[string]bool{}
	for _, name := range sortedGrantRoles(p.Grants) {
		role := roles[name]
		for _, permission := range p.Grants[name] {
			conditions := constraint.Empty()
			if permission.Origin != "" {
				var err error
				conditions, err = constraint.New(constraint.Equal("object.origin_type", constraint.StringValue(permission.Origin)))
				if err != nil {
					return err
				}
			}
			rid := resourcedomain.NewResourceID(0)
			if catalog, ok := resources[permission.Resource]; ok {
				rid = resourcedomain.NewResourceID(catalog.ID.Uint64())
			}
			var g grantdomain.Grant
			var err error
			if rid.Uint64() == 0 || permission.Action == "*" {
				g, err = grantdomain.NewSystem(role.ID, rid, permission.Resource, permission.Action, conditions, MigrationID)
			} else {
				g, err = grantdomain.New(role.ID, rid, permission.Resource, permission.Action, conditions, MigrationID)
			}
			if err != nil {
				return err
			}
			if err = (roledomain.Role{ManagementProtection: roledomain.ManagementProtection(role.ManagementProtection)}).ValidateGrant(g.ResourceKey, g.Action); err != nil {
				return err
			}
			if rid.Uint64() != 0 {
				catalog := resources[permission.Resource]
				bo, err := resourcepo.NewMapper().ToBO(&catalog)
				if err != nil {
					return err
				}
				if err = g.ValidateAgainst(*bo); err != nil {
					return err
				}
			}
			desired[g.GrantKey] = true
			exists := false
			for _, old := range before.Grants {
				if old.DeletedAt == nil && old.RevokedAt == nil && old.GrantKey == g.GrantKey {
					exists = true
					break
				}
			}
			if !exists {
				po, err := (grantpo.Mapper{}).ToPO(&g)
				if err != nil {
					return err
				}
				if err = tx.Create(po).Error; err != nil {
					return err
				}
			}
		}
	}
	for _, g := range before.Grants {
		if g.DeletedAt == nil && g.RevokedAt == nil && !desired[g.GrantKey] {
			if err := tx.Table("authz_permission_grants").Where("id = ?", g.ID).Updates(map[string]any{"revoked_at": now, "updated_at": now, "version": gorm.Expr("version + 1")}).Error; err != nil {
				return err
			}
		}
	}
	desiredAssignments := map[string]bool{}
	for _, person := range p.People {
		for _, name := range person.After {
			role := roles[name]
			key := person.Subject + "/" + role.ID.String()
			desiredAssignments[key] = true
			found := false
			for _, a := range before.Assignments {
				if a.DeletedAt == nil && a.SubjectType+":"+a.SubjectID == person.Subject && a.RoleID == role.ID.Uint64() {
					found = true
					break
				}
			}
			if !found {
				a := assignmentpo.AssignmentPO{SubjectType: "user", SubjectID: person.Subject[len("user:"):], RoleID: role.ID.Uint64(), GrantedBy: MigrationID}
				if err := tx.Create(&a).Error; err != nil {
					return err
				}
			}
		}
	}
	for _, a := range before.Assignments {
		if a.DeletedAt == nil && !desiredAssignments[a.SubjectType+":"+a.SubjectID+"/"+meta.FromUint64(a.RoleID).String()] {
			if err := tx.Table("authz_assignments").Where("id = ?", a.ID).Updates(map[string]any{"deleted_at": now, "updated_at": now, "version": gorm.Expr("version + 1")}).Error; err != nil {
				return err
			}
		}
	}
	if tx.Migrator().HasTable(&LegacyEdge{}) {
		if err := tx.Table("authz_role_inheritances").Where("deleted_at IS NULL AND revoked_at IS NULL").Updates(map[string]any{"revoked_at": now, "updated_at": now, "version": gorm.Expr("version + 1")}).Error; err != nil {
			return err
		}
	}
	for _, name := range p.ArchiveRoles {
		if err := tx.Table("authz_roles").Where("id = ?", roles[name].ID).Updates(map[string]any{"deleted_at": now, "updated_at": now, "version": gorm.Expr("version + 1")}).Error; err != nil {
			return err
		}
	}
	return tx.Table("authz_resources").Where("`key` = ? AND deleted_at IS NULL", "iam:authz:collection:role_inheritances").Updates(map[string]any{"deleted_at": now, "updated_at": now, "version": gorm.Expr("version + 1")}).Error
}
func sortedGrantRoles(m map[string][]Permission) []string {
	out := []string{}
	for name := range m {
		out = append(out, name)
	}
	return uniqueStrings(out)
}

func Verify(ctx context.Context, db *gorm.DB) (*Receipt, error) {
	status, err := Status(ctx, db)
	if err != nil {
		return nil, err
	}
	if status.State != "applied_unchanged" {
		return nil, fmt.Errorf("migration %s: %s", status.State, status.NextAction)
	}
	var r Receipt
	if err := db.WithContext(ctx).First(&r, "migration_id = ?", MigrationID).Error; err != nil {
		return nil, err
	}
	if r.Status != "applied" {
		return &r, fmt.Errorf("migration is not applied")
	}
	s, err := loadStateForVerification(ctx, db)
	if err != nil {
		return &r, err
	}
	if s.Hash() != r.AfterHash {
		return &r, fmt.Errorf("authorization facts changed since apply; reconcile before final cleanup")
	}
	for _, e := range s.Edges {
		if !e.DeletedAt.Valid && !e.RevokedAt.Valid {
			return &r, fmt.Errorf("active inheritance remains")
		}
	}
	var plan Plan
	if err = json.Unmarshal([]byte(r.PlanJSON), &plan); err != nil {
		return &r, err
	}
	return &r, verifyPlan(s, plan)
}

// Rollback restores only when the complete after-image still matches. It never
// overwrites later authorization changes and always emits a higher version.
func Rollback(ctx context.Context, db *gorm.DB, stager event.Stager, fingerprint string, writesStopped bool) (*Receipt, error) {
	if !writesStopped || len(fingerprint) != 64 {
		return nil, fmt.Errorf("rollback requires stopped writers and original fingerprint")
	}
	var r Receipt
	err := dbmysql.NewUnitOfWork(db).WithinTransaction(ctx, func(txCtx context.Context) error {
		tx, err := dbmysql.RequireTx(txCtx)
		if err != nil {
			return err
		}
		if err = lockPolicy(tx); err != nil {
			return err
		}
		if err = tx.First(&r, "migration_id = ?", MigrationID).Error; err != nil {
			return err
		}
		if r.Fingerprint != fingerprint {
			return fmt.Errorf("rollback fingerprint mismatch")
		}
		if r.Status == "rolled_back" {
			return nil
		}
		if r.Status != "applied" {
			return fmt.Errorf("unexpected migration status")
		}
		if !tx.Migrator().HasTable(&LegacyEdge{}) {
			return fmt.Errorf("restore archived inheritance table schema and rows before rollback")
		}
		current, err := LoadState(txCtx, tx)
		if err != nil {
			return err
		}
		if current.Hash() != r.AfterHash {
			return fmt.Errorf("rollback conflicts with subsequent authorization writes")
		}
		var before State
		if err = json.Unmarshal([]byte(r.BeforeJSON), &before); err != nil {
			return err
		}
		if len(before.Edges) > 0 && !tx.Migrator().HasTable(&LegacyEdge{}) {
			return fmt.Errorf("restore archived inheritance table schema before rollback")
		}
		// Remove only migration-created rows; the full after-image match proves no
		// concurrent writer can have added references to them during this window.
		if err = restoreRows(tx, "authz_assignments", before.Assignments, current.Assignments); err != nil {
			return err
		}
		if err = restoreRows(tx, "authz_permission_grants", before.Grants, current.Grants); err != nil {
			return err
		}
		if err = restoreRows(tx, "authz_resources", before.Resources, current.Resources); err != nil {
			return err
		}
		if err = restoreRows(tx, "authz_roles", before.Roles, current.Roles); err != nil {
			return err
		}
		if len(before.Edges) > 0 {
			if err = restoreRows(tx, "authz_role_inheritances", before.Edges, current.Edges); err != nil {
				return err
			}
		}
		if err = advance(txCtx, tx, stager, "role model rollback"); err != nil {
			return err
		}
		r.Status = "rolled_back"
		r.UpdatedAt = time.Now().UTC()
		return tx.Save(&r).Error
	})
	return &r, err
}
func restoreRows[T any](tx *gorm.DB, table string, before, current []T) error {
	// Use GORM's schema field mapping rather than JSON names, preserving all
	// audit values while omitting generated columns and hooks on restoration.
	ids := map[any]bool{}
	for i := range before {
		stmt := &gorm.Statement{DB: tx}
		if err := stmt.Parse(&before[i]); err != nil {
			return err
		}
		values := map[string]any{}
		for _, f := range stmt.Schema.Fields {
			if f.DBName == "" || f.DBName == "active_guard" {
				continue
			}
			v, _ := f.ValueOf(context.Background(), reflectValue(&before[i]))
			// Historical system audit actors are numeric zero. meta.ID's
			// normal Valuer maps zero to NULL, which violates legacy NOT NULL
			// audit columns when restoring through a map.
			if id, ok := v.(meta.ID); ok && id.IsZero() {
				v = int64(0)
			}
			// A nullable JSON column is read into the PO string as empty.
			// MySQL rejects an empty document; restore its SQL NULL instead.
			if f.DataType == "json" && v == "" {
				v = nil
			}
			values[f.DBName] = v
		}
		ids[values["id"]] = true
		if err := tx.Table(table).Where("id = ?", values["id"]).Updates(values).Error; err != nil {
			return err
		}
	}
	for i := range current {
		stmt := &gorm.Statement{DB: tx}
		if err := stmt.Parse(&current[i]); err != nil {
			return err
		}
		f := stmt.Schema.LookUpField("id")
		id, _ := f.ValueOf(context.Background(), reflectValue(&current[i]))
		if !ids[id] {
			if err := tx.Exec("DELETE FROM "+table+" WHERE id = ?", id).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func reflectValue(v any) reflect.Value { return reflect.ValueOf(v).Elem() }

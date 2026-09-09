package maintenance

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/constraint"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/permissiongrant"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/resource"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/role"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/roleinheritance"
	assignmentrepo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/assignment"
	grantrepo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/permissiongrant"
	resourcerepo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/resource"
	rolerepo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/role"
	inheritancerepo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/roleinheritance"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// 以下 LegacyDomain 仅用于读取迁移前的历史列，不进入新领域模型。
type retirementRole struct {
	rolerepo.RolePO
	LegacyDomain string `gorm:"column:tenant_id"`
}
type retirementAssignment struct {
	assignmentrepo.AssignmentPO
	LegacyDomain string `gorm:"column:tenant_id"`
}
type retirementInheritance struct {
	inheritancerepo.InheritancePO
	LegacyDomain string `gorm:"column:tenant_id"`
}
type retirementGrant struct {
	grantrepo.GrantPO
	LegacyDomain string `gorm:"column:tenant_id"`
}
type retirementVersion struct {
	ID            uint64
	LegacyDomain  string `gorm:"column:tenant_id"`
	PolicyVersion int64
}
type retirementState struct {
	Roles        []retirementRole
	Assignments  []retirementAssignment
	Inheritances []retirementInheritance
	Grants       []retirementGrant
	Resources    []resourcerepo.ResourcePO
	Versions     []retirementVersion
}

type RetirementIssue struct {
	Kind   string `json:"kind"`
	ID     string `json:"id"`
	Detail string `json:"detail"`
}
type RetirementRoleChange struct {
	ID         meta.ID                   `json:"id"`
	Before     string                    `json:"before"`
	After      string                    `json:"after"`
	Protection role.ManagementProtection `json:"management_protection"`
}
type RetirementPermission struct {
	Subject                   string   `json:"subject"`
	RoleID                    meta.ID  `json:"role_id"`
	Resource                  string   `json:"resource"`
	Action                    string   `json:"action"`
	LegacyDomain              string   `json:"legacy_domain"`
	NewRoleName               string   `json:"new_role_name"`
	ConstraintSet             string   `json:"constraint_set"`
	AddedProfileActions       []string `json:"added_profile_actions"`
	PartitionSelectionRemoved bool     `json:"partition_selection_removed"`
}
type TenantRetirementReport struct {
	Fingerprint          string                 `json:"fingerprint"`
	Ready                bool                   `json:"ready"`
	Issues               []RetirementIssue      `json:"issues"`
	Roles                []RetirementRoleChange `json:"roles"`
	Permissions          []RetirementPermission `json:"permissions"`
	InitialPolicyVersion int64                  `json:"initial_policy_version"`
	grantKeys            map[meta.ID]string
}

func loadRetirementState(ctx context.Context, db *gorm.DB, lock bool) (retirementState, error) {
	s := retirementState{}
	for _, query := range []struct {
		table string
		rows  any
	}{
		{"authz_roles", &s.Roles}, {"authz_assignments", &s.Assignments}, {"authz_role_inheritances", &s.Inheritances}, {"authz_permission_grants", &s.Grants}, {"authz_resources", &s.Resources}, {"authz_policy_versions", &s.Versions},
	} {
		q := db.WithContext(ctx).Unscoped().Table(query.table).Order("id ASC")
		if lock && db.Dialector.Name() == "mysql" {
			q = q.Clauses(clause.Locking{Strength: "UPDATE"})
		}
		if err := q.Find(query.rows).Error; err != nil {
			return s, err
		}
	}
	return s, nil
}

func AnalyzeTenantRetirement(ctx context.Context, db *gorm.DB) (*TenantRetirementReport, error) {
	var report *TenantRetirementReport
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		s, err := loadRetirementState(ctx, tx, false)
		if err != nil {
			return err
		}
		report = analyzeRetirement(s)
		return checkRetirementCollation(tx, report)
	})
	return report, err
}

func checkRetirementCollation(db *gorm.DB, report *TenantRetirementReport) error {
	// 使用数据库自身的排序规则，覆盖大小写、重音和尾空格等名称碰撞。
	var collisions []struct {
		Name  string
		Count int64
	}
	expression := "CASE WHEN tenant_id = 'platform' AND name = 'super_admin' THEN 'platform_admin' WHEN tenant_id = 'fangcun' AND name = 'tenant_admin' THEN 'iam_admin' ELSE name END"
	if err := db.Table("authz_roles").Select(expression + " AS name, COUNT(*) AS count").Group(expression).Having("COUNT(*) > 1").Scan(&collisions).Error; err != nil {
		return err
	}
	for _, collision := range collisions {
		report.Issues = append(report.Issues, RetirementIssue{"role_name_collision", "", collision.Name})
	}
	report.Ready = len(report.Issues) == 0
	return nil
}

func analyzeRetirement(s retirementState) *TenantRetirementReport {
	encoded, _ := json.Marshal(s)
	r := &TenantRetirementReport{Fingerprint: fmt.Sprintf("%x", sha256.Sum256(encoded)), Issues: []RetirementIssue{}, Roles: []RetirementRoleChange{}, Permissions: []RetirementPermission{}, grantKeys: map[meta.ID]string{}}
	issue := func(kind string, id meta.ID, detail string) {
		r.Issues = append(r.Issues, RetirementIssue{kind, id.String(), detail})
	}
	known := func(domain string, id meta.ID) {
		if domain != "platform" && domain != "fangcun" {
			issue("unknown_legacy_domain", id, domain)
		}
	}
	roles := map[meta.ID]retirementRole{}
	names := map[string]meta.ID{}
	protection := map[meta.ID]role.ManagementProtection{}
	nodes := []roleinheritance.RoleNode{}
	for _, row := range s.Roles {
		known(row.LegacyDomain, row.ID)
		name := row.Name
		p := role.ManagementStandard
		if row.LegacyDomain == "platform" {
			p = role.ManagementProtected
			if name == "super_admin" {
				name = "platform_admin"
			}
		}
		if row.LegacyDomain == "fangcun" && name == "tenant_admin" {
			name = "iam_admin"
		}
		if row.ID.IsZero() || strings.TrimSpace(name) == "" || name != strings.TrimSpace(name) || len(name) > 64 {
			issue("invalid_role", row.ID, name)
		}
		if previous, ok := names[name]; ok {
			issue("role_name_collision", row.ID, previous.String())
		}
		names[name] = row.ID
		roles[row.ID] = row
		protection[row.ID] = p
		r.Roles = append(r.Roles, RetirementRoleChange{row.ID, row.Name, name, p})
		if row.DeletedAt == nil {
			nodes = append(nodes, roleinheritance.RoleNode{ID: row.ID, ManagementProtection: p})
		}
	}
	reference := func(id meta.ID, roleID uint64, domain string, active bool) bool {
		known(domain, id)
		target, ok := roles[meta.ID(roleID)]
		if !ok || target.LegacyDomain != domain || (active && target.DeletedAt != nil) {
			issue("invalid_role_reference", id, fmt.Sprintf("role_id=%d record_domain=%s active=%t target_exists=%t target_name=%s target_domain=%s target_deleted=%t", roleID, domain, active, ok, target.Name, target.LegacyDomain, target.DeletedAt != nil))
			return false
		}
		return true
	}
	for _, a := range s.Assignments {
		reference(a.ID, a.RoleID, a.LegacyDomain, a.DeletedAt == nil)
	}
	edges := []*roleinheritance.Inheritance{}
	for _, e := range s.Inheritances {
		active := e.DeletedAt == nil && e.RevokedAt == nil
		reference(e.ID, e.RoleID, e.LegacyDomain, active)
		reference(e.ID, e.InheritedRoleID, e.LegacyDomain, active)
		if active {
			edges = append(edges, &roleinheritance.Inheritance{RoleID: meta.ID(e.RoleID), InheritedRoleID: meta.ID(e.InheritedRoleID)})
		}
	}
	if err := roleinheritance.ValidateGraph(nodes, edges); err != nil {
		issue("invalid_role_graph", 0, err.Error())
	}
	resources := map[uint64]resourcerepo.ResourcePO{}
	for _, x := range s.Resources {
		resources[x.ID.Uint64()] = x
	}
	activeKeys := map[string]meta.ID{}
	for _, g := range s.Grants {
		active := g.DeletedAt == nil && g.RevokedAt == nil
		reference(g.ID, g.RoleID, g.LegacyDomain, active)
		if g.ResourceID != nil {
			target, ok := resources[*g.ResourceID]
			if !ok || (active && target.DeletedAt != nil) {
				issue("invalid_resource_reference", g.ID, fmt.Sprintf("resource_id=%d resource=%s action=%s active=%t target_exists=%t target_deleted=%t", *g.ResourceID, g.ResourcePattern, g.Action, active, ok, target.DeletedAt != nil))
			}
		}
		c, err := constraint.ParseJSON([]byte(g.ConstraintSet))
		if err != nil {
			issue("invalid_constraint", g.ID, err.Error())
			continue
		}
		resourceID := resource.ResourceID{}
		if g.ResourceID != nil {
			resourceID = resource.NewResourceID(*g.ResourceID)
		}
		grant, err := permissiongrant.Restore(meta.ID(g.RoleID), resourceID, g.ResourcePattern, g.Action, c, g.GrantedBy, permissiongrant.RestoreOptions{})
		if err != nil {
			issue("invalid_grant", g.ID, err.Error())
			continue
		}
		canonical, _ := c.CanonicalJSON()
		legacyPayload := fmt.Sprintf("v1\x00%s\x00%d\x00%d\x00%s\x00%s\x00%s", g.LegacyDomain, g.RoleID, resourceID.Uint64(), g.ResourcePattern, g.Action, canonical)
		legacyKey := fmt.Sprintf("%x", sha256.Sum256([]byte(legacyPayload)))
		if g.GrantKey != legacyKey && g.GrantKey != grant.GrantKey {
			issue("invalid_grant_key", g.ID, "canonical key mismatch")
		}
		if err := (role.Role{ManagementProtection: protection[meta.ID(g.RoleID)]}).ValidateGrant(grant.ResourceKey, grant.Action); err != nil {
			issue("sensitive_standard_grant", g.ID, fmt.Sprintf("role_id=%d role=%s resource=%s action=%s active=%t deleted=%t revoked=%t", g.RoleID, roles[meta.ID(g.RoleID)].Name, g.ResourcePattern, g.Action, active, g.DeletedAt != nil, g.RevokedAt != nil))
		}
		r.grantKeys[g.ID] = grant.GrantKey
		if active {
			if old, ok := activeKeys[grant.GrantKey]; ok {
				issue("duplicate_active_grant", g.ID, old.String())
			}
			activeKeys[grant.GrantKey] = g.ID
		}
	}
	for _, v := range s.Versions {
		known(v.LegacyDomain, meta.ID(v.ID))
		if v.PolicyVersion < 0 {
			issue("invalid_policy_version", meta.ID(v.ID), "negative version")
		}
		r.InitialPolicyVersion = max(r.InitialPolicyVersion, v.PolicyVersion)
	}
	if r.InitialPolicyVersion == math.MaxInt64 {
		issue("policy_version_overflow", 0, "cannot increment")
	} else {
		r.InitialPolicyVersion++
	}
	for _, a := range s.Assignments {
		if a.DeletedAt != nil {
			continue
		}
		closure := map[uint64]bool{a.RoleID: true}
		for changed := true; changed; {
			changed = false
			for _, e := range edges {
				if closure[e.RoleID.Uint64()] && !closure[e.InheritedRoleID.Uint64()] {
					closure[e.InheritedRoleID.Uint64()] = true
					changed = true
				}
			}
		}
		for _, g := range s.Grants {
			if g.DeletedAt == nil && g.RevokedAt == nil && closure[g.RoleID] {
				name := roles[meta.ID(g.RoleID)].Name
				for _, change := range r.Roles {
					if change.ID == meta.ID(g.RoleID) {
						name = change.After
						break
					}
				}
				added := []string{}
				if g.LegacyDomain == "platform" && resource.Key(g.ResourcePattern).Covers(resource.Key("iam:identity:collection:profiles")) {
					switch g.Action {
					case "list":
						added = append(added, "list_all")
					case "search_by_mobile":
						added = append(added, "search_by_mobile_all")
					}
				}
				r.Permissions = append(r.Permissions, RetirementPermission{Subject: a.SubjectType + ":" + a.SubjectID, RoleID: meta.ID(g.RoleID), Resource: g.ResourcePattern, Action: g.Action, LegacyDomain: g.LegacyDomain, NewRoleName: name, ConstraintSet: g.ConstraintSet, AddedProfileActions: added, PartitionSelectionRemoved: true})
			}
		}
	}
	sort.Slice(r.Issues, func(i, j int) bool {
		a, b := r.Issues[i], r.Issues[j]
		return a.Kind+a.ID+a.Detail < b.Kind+b.ID+b.Detail
	})
	r.Ready = len(r.Issues) == 0
	return r
}

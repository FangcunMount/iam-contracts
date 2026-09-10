package rolemodel

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/constraint"
	assignmentpo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/assignment"
	grantpo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/permissiongrant"
	resourcepo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/resource"
	rolepo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/role"
	"gorm.io/gorm"
)

// LegacyEdge is an archival row. The migration must remain able to restore it
// after the runtime/domain inheritance implementation is retired.
type LegacyEdge struct {
	ID              uint64       `json:"id" gorm:"primaryKey"`
	RoleID          uint64       `json:"role_id"`
	InheritedRoleID uint64       `json:"inherited_role_id"`
	GrantedBy       string       `json:"granted_by"`
	GrantedAt       sql.NullTime `json:"granted_at"`
	RevokedAt       sql.NullTime `json:"revoked_at"`
	CreatedAt       sql.NullTime `json:"created_at"`
	UpdatedAt       sql.NullTime `json:"updated_at"`
	DeletedAt       sql.NullTime `json:"deleted_at"`
	CreatedBy       uint64       `json:"created_by"`
	UpdatedBy       uint64       `json:"updated_by"`
	DeletedBy       uint64       `json:"deleted_by"`
	Version         uint32       `json:"version"`
}

func (LegacyEdge) TableName() string { return "authz_role_inheritances" }

type State struct {
	Roles         []rolepo.RolePO             `json:"roles"`
	Assignments   []assignmentpo.AssignmentPO `json:"assignments"`
	Grants        []grantpo.GrantPO           `json:"grants"`
	Resources     []resourcepo.ResourcePO     `json:"resources"`
	Edges         []LegacyEdge                `json:"edges"`
	PolicyVersion int64                       `json:"policy_version"`
}

func LoadState(ctx context.Context, db *gorm.DB) (State, error) {
	var s State
	for _, q := range []struct {
		table string
		rows  any
	}{{"authz_roles", &s.Roles}, {"authz_assignments", &s.Assignments}, {"authz_permission_grants", &s.Grants}, {"authz_resources", &s.Resources}} {
		if err := db.WithContext(ctx).Unscoped().Table(q.table).Order("id ASC").Find(q.rows).Error; err != nil {
			return s, err
		}
	}
	if db.Migrator().HasTable(&LegacyEdge{}) {
		if err := db.WithContext(ctx).Unscoped().Order("id ASC").Find(&s.Edges).Error; err != nil {
			return s, err
		}
	}
	if err := db.WithContext(ctx).Table("authz_policy_versions").Where("id = ?", 1).Pluck("policy_version", &s.PolicyVersion).Error; err != nil {
		return s, err
	}
	return s, nil
}
func (s State) Hash() string {
	raw, _ := json.Marshal(s)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
func (s State) Input(people []Person) (Input, error) {
	in := Input{Version: s.PolicyVersion, FactsHash: s.Hash(), People: people}
	names := map[uint64]string{}
	for _, r := range s.Roles {
		if r.DeletedAt != nil {
			continue
		}
		names[r.ID.Uint64()] = r.Name
		in.Roles = append(in.Roles, Role{ID: r.ID.String(), Name: r.Name, Protection: r.ManagementProtection})
	}
	for _, a := range s.Assignments {
		if a.DeletedAt != nil {
			continue
		}
		in.Assignments = append(in.Assignments, Assignment{ID: a.ID.String(), Subject: a.SubjectType + ":" + a.SubjectID, Role: names[a.RoleID]})
	}
	for _, e := range s.Edges {
		if e.DeletedAt.Valid || e.RevokedAt.Valid {
			continue
		}
		in.Edges = append(in.Edges, Edge{ID: strconv.FormatUint(e.ID, 10), Role: names[e.RoleID], Parent: names[e.InheritedRoleID]})
	}
	for _, r := range s.Resources {
		if r.DeletedAt != nil {
			continue
		}
		var actions []string
		if err := json.Unmarshal([]byte(r.Actions), &actions); err != nil {
			return in, err
		}
		in.Resources = append(in.Resources, Resource{Key: r.Key, Actions: actions})
	}
	for _, g := range s.Grants {
		if g.DeletedAt != nil || g.RevokedAt != nil {
			continue
		}
		bo, err := (grantpo.Mapper{}).ToBO(&g)
		if err != nil {
			return in, fmt.Errorf("grant %s: %w", g.ID, err)
		}
		p := Permission{Resource: bo.ResourceKeyString(), Action: bo.ActionString()}
		if len(bo.Constraints.AllOf) > 0 {
			if len(bo.Constraints.AllOf) != 1 {
				return in, fmt.Errorf("unsupported source conditions on grant %s", g.ID)
			}
			c := bo.Constraints.AllOf[0]
			if c.Key != "object.origin_type" || c.Operator != constraint.OperatorEQ || c.Value.String == nil {
				return in, fmt.Errorf("unsupported source condition on grant %s", g.ID)
			}
			p.Origin = *c.Value.String
		}
		in.Grants = append(in.Grants, Grant{ID: g.ID.String(), Role: names[g.RoleID], Permission: p})
	}
	return in, nil
}

// ReadPeople never reads contact details. Active legacy staff require a unique
// active QS staff mapping; clinician evidence is tied to that staff and org.
func ReadPeople(ctx context.Context, iam, qs *gorm.DB, s State) ([]Person, error) {
	roles := map[uint64]string{}
	for _, r := range s.Roles {
		roles[r.ID.Uint64()] = r.Name
	}
	subjects := map[string]bool{}
	legacyStaff := map[string]bool{}
	for _, a := range s.Assignments {
		if a.DeletedAt != nil {
			continue
		}
		key := a.SubjectType + ":" + a.SubjectID
		subjects[key] = true
		if roles[a.RoleID] == "qs:staff" {
			legacyStaff[key] = true
		}
	}
	keys := []string{}
	for key := range subjects {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := []Person{}
	for _, key := range keys {
		p := Person{Subject: key, ClinicianIDs: []string{}}
		id, err := strconv.ParseUint(strings.TrimPrefix(key, "user:"), 10, 64)
		if err != nil || !strings.HasPrefix(key, "user:") {
			p.IdentityIssue = "invalid user reference"
			out = append(out, p)
			continue
		}
		var users []struct {
			ID        uint64
			Status    int
			DeletedAt sql.NullTime
		}
		if err := iam.WithContext(ctx).Table("users").Select("id,status,deleted_at").Where("id = ?", id).Find(&users).Error; err != nil {
			return nil, err
		}
		evidence := []any{users}
		p.Active = len(users) == 1 && users[0].Status == 1 && !users[0].DeletedAt.Valid
		if legacyStaff[key] {
			if qs == nil {
				return nil, fmt.Errorf("QS database is required for clinician evidence")
			}
			var staff []struct {
				ID    uint64
				OrgID int64
			}
			if err := qs.WithContext(ctx).Table("staff").Select("id,org_id").Where("user_id = ? AND deleted_at IS NULL AND is_active = ?", id, true).Order("id ASC").Find(&staff).Error; err != nil {
				return nil, err
			}
			evidence = append(evidence, staff)
			if len(staff) != 1 {
				p.IdentityIssue = "expected one active QS staff identity"
			} else {
				var clinicians []struct {
					ID    uint64
					OrgID int64
				}
				if err := qs.WithContext(ctx).Table("clinician").Select("id,org_id").Where("operator_id = ? AND deleted_at IS NULL AND is_active = ?", staff[0].ID, true).Order("id ASC").Find(&clinicians).Error; err != nil {
					return nil, err
				}
				evidence = append(evidence, clinicians)
				p.ClinicianChecked = true
				for _, c := range clinicians {
					if c.OrgID != staff[0].OrgID {
						p.IdentityIssue = "clinician organization mismatch"
					}
					p.ClinicianIDs = append(p.ClinicianIDs, strconv.FormatUint(c.ID, 10))
				}
				if len(clinicians) > 1 {
					p.IdentityIssue = "multiple active clinician identities"
				}
			}
		}
		raw, _ := json.Marshal(evidence)
		sum := sha256.Sum256(raw)
		p.EvidenceHash = hex.EncodeToString(sum[:])
		out = append(out, p)
	}
	return out, nil
}

func Preflight(ctx context.Context, iam, qs *gorm.DB) (Plan, error) {
	var p Plan
	err := iam.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		s, err := LoadState(ctx, tx)
		if err != nil {
			return err
		}
		people, err := ReadPeople(ctx, tx, qs, s)
		if err != nil {
			return err
		}
		in, err := s.Input(people)
		if err != nil {
			return err
		}
		p = Build(in)
		return nil
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	return p, err
}

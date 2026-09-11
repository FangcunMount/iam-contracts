// Package conditionretire owns the one-time retirement of object conditions.
// It reads archived persistence facts; it must never participate in authorization.
package conditionretire

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/FangcunMount/iam/v5/internal/pkg/meta"

	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/resource"
	grantpo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/permissiongrant"
	"github.com/FangcunMount/iam/v5/internal/apiserver/maintenance/legacycondition/constraint"
	grantdomain "github.com/FangcunMount/iam/v5/internal/apiserver/maintenance/legacycondition/grant"
	"github.com/FangcunMount/iam/v5/internal/apiserver/maintenance/rolemodel"
)

const MigrationID = "condition-authz-retire-v1"
const assessment = "qs:evaluation:collection:assessments"
const emptySchema = `{"version":1,"attributes":[]}`

type GrantChange struct {
	OldID      string `json:"old_id"`
	Role       string `json:"role"`
	Origin     string `json:"origin"`
	TargetKey  string `json:"target_key"`
	ExistingID string `json:"existing_id,omitempty"`
}
type UserChange struct {
	Subject       string   `json:"subject"`
	Resource      string   `json:"resource"`
	Action        string   `json:"action"`
	BeforeOrigins []string `json:"before_origins"`
	AfterOrigins  []string `json:"after_origins"`
	AddedOrigins  []string `json:"added_origins"`
}
type Plan struct {
	Fingerprint string        `json:"fingerprint"`
	Grants      []GrantChange `json:"grants"`
	Resources   []string      `json:"resources"`
	Users       []UserChange  `json:"users"`
	Issues      []string      `json:"issues"`
}

func (p Plan) Validate() error {
	if len(p.Issues) > 0 {
		return fmt.Errorf("retirement preflight blocked: %v", p.Issues)
	}
	return nil
}

func decode(raw string, v any) error {
	d := json.NewDecoder(bytes.NewBufferString(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return fmt.Errorf("trailing JSON data")
	}
	return nil
}
func conditions(raw string) (constraint.Set, error) {
	var c constraint.Set
	if err := decode(raw, &c); err != nil {
		return c, err
	}
	return c.Normalize()
}

// Build permits exactly the two reviewed business rules. Unexpected conditions
// are reported, never erased or promoted to unconditional access.
func Build(s rolemodel.State) Plan {
	p := Plan{Fingerprint: s.Hash(), Grants: []GrantChange{}, Resources: []string{}, Users: []UserChange{}, Issues: []string{}}
	roles := map[uint64]string{}
	resources := map[uint64]string{}
	for _, r := range s.Roles {
		if r.DeletedAt == nil {
			roles[r.ID.Uint64()] = r.Name
		}
	}
	for _, r := range s.Resources {
		if r.DeletedAt != nil {
			continue
		}
		resources[r.ID.Uint64()] = r.Key
		var schema struct {
			Version    uint32 `json:"version"`
			Attributes []struct {
				Key    string   `json:"key"`
				Type   string   `json:"type"`
				Values []string `json:"allowed_string_values"`
			} `json:"attributes"`
		}
		raw := r.AttributeSchema
		if raw == "" {
			raw = "null"
		}
		if err := decode(raw, &schema); err != nil || (schema.Version != 0 && schema.Version != 1) {
			p.Issues = append(p.Issues, "invalid resource schema: "+r.ID.String())
			continue
		}
		if len(schema.Attributes) == 0 {
			continue
		}
		valid := r.Key == assessment && len(schema.Attributes) == 1
		if valid {
			a := schema.Attributes[0]
			sort.Strings(a.Values)
			valid = a.Key == "object.origin_type" && a.Type == "string" && len(a.Values) == 2 && a.Values[0] == "adhoc" && a.Values[1] == "plan"
		}
		if !valid {
			p.Issues = append(p.Issues, "unexpected resource schema: "+r.Key)
			continue
		}
		p.Resources = append(p.Resources, r.ID.String())
	}
	active := []grantpo.GrantPO{}
	parsed := map[string]constraint.Set{}
	existing := map[string]string{}
	for _, g := range s.Grants {
		if g.DeletedAt != nil || g.RevokedAt != nil {
			continue
		}
		c, err := conditions(g.ConstraintSet)
		if err != nil {
			p.Issues = append(p.Issues, "invalid grant conditions: "+g.ID.String())
			continue
		}
		active = append(active, g)
		parsed[g.ID.String()] = c
		rid := resource.NewResourceID(0)
		if g.ResourceID != nil {
			rid = resource.NewResourceID(*g.ResourceID)
		}
		var b grantdomain.Grant
		if rid.Uint64() == 0 || g.Action == "*" {
			b, err = grantdomain.NewSystem(meta.FromUint64(g.RoleID), rid, g.ResourcePattern, g.Action, c, g.GrantedBy)
		} else {
			b, err = grantdomain.New(meta.FromUint64(g.RoleID), rid, g.ResourcePattern, g.Action, c, g.GrantedBy)
		}
		if err != nil || b.GrantKey != g.GrantKey {
			p.Issues = append(p.Issues, "invalid grant canonical key: "+g.ID.String())
			continue
		}
		if roles[g.RoleID] == "" {
			p.Issues = append(p.Issues, "grant has no active role: "+g.ID.String())
		}
		if c.IsUnconditional() {
			existing[g.GrantKey] = g.ID.String()
			continue
		}
		expected := map[string]string{rolemodel.Operator: "adhoc", "qs:evaluation_plan_manager": "plan"}[roles[g.RoleID]]
		if expected == "" || g.ResourcePattern != assessment || g.Action != "retry" || g.ResourceID == nil || resources[*g.ResourceID] != assessment || len(c.AllOf) != 1 {
			p.Issues = append(p.Issues, "unexpected conditional grant: "+g.ID.String())
			continue
		}
		pred := c.AllOf[0]
		if pred.Key != "object.origin_type" || pred.Operator != constraint.OperatorEQ || pred.Value.String == nil || *pred.Value.String != expected {
			p.Issues = append(p.Issues, "unexpected retry condition: "+g.ID.String())
			continue
		}
		target, err := grantdomain.New(meta.FromUint64(g.RoleID), rid, g.ResourcePattern, g.Action, constraint.Empty(), MigrationID)
		if err != nil {
			p.Issues = append(p.Issues, err.Error())
			continue
		}
		p.Grants = append(p.Grants, GrantChange{OldID: g.ID.String(), Role: roles[g.RoleID], Origin: expected, TargetKey: target.GrantKey})
	}
	for i := range p.Grants {
		p.Grants[i].ExistingID = existing[p.Grants[i].TargetKey]
	}
	affected := map[string]bool{}
	changed := map[string]bool{}
	assigned := map[string]map[uint64]bool{}
	for _, g := range p.Grants {
		changed[g.OldID] = true
	}
	for _, a := range s.Assignments {
		if a.DeletedAt != nil {
			continue
		}
		sub := a.SubjectType + ":" + a.SubjectID
		if assigned[sub] == nil {
			assigned[sub] = map[uint64]bool{}
		}
		assigned[sub][a.RoleID] = true
		for _, g := range active {
			if changed[g.ID.String()] && g.RoleID == a.RoleID {
				affected[sub] = true
			}
		}
	}
	target, _ := resource.NewKey(assessment)
	for sub := range affected {
		before := map[string]bool{}
		after := map[string]bool{}
		for _, g := range active {
			if !assigned[sub][g.RoleID] {
				continue
			}
			key, err := resource.NewKey(g.ResourcePattern)
			if err != nil || !key.Covers(target) || (g.Action != "retry" && g.Action != "*") {
				continue
			}
			c := parsed[g.ID.String()]
			for _, origin := range []string{"adhoc", "plan"} {
				if c.IsUnconditional() || (len(c.AllOf) == 1 && c.AllOf[0].Value.String != nil && *c.AllOf[0].Value.String == origin) {
					before[origin] = true
					after[origin] = true
				}
				if changed[g.ID.String()] {
					after[origin] = true
				}
			}
		}
		u := UserChange{Subject: sub, Resource: assessment, Action: "retry", BeforeOrigins: []string{}, AfterOrigins: []string{}, AddedOrigins: []string{}}
		for _, o := range []string{"adhoc", "plan"} {
			if before[o] {
				u.BeforeOrigins = append(u.BeforeOrigins, o)
			}
			if after[o] {
				u.AfterOrigins = append(u.AfterOrigins, o)
			}
			if after[o] && !before[o] {
				u.AddedOrigins = append(u.AddedOrigins, o)
			}
		}
		p.Users = append(p.Users, u)
	}
	sort.Slice(p.Users, func(i, j int) bool { return p.Users[i].Subject < p.Users[j].Subject })
	return p
}

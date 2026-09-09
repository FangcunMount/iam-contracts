// Package rolemodel describes the reviewed migration from inherited roles to
// independent business responsibilities. It does not infer jobs from identities.
package rolemodel

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const (
	MigrationID = "independent-business-roles-v1"
	Operator    = "qs:assessment_operator"
	Reviewer    = "qs:result_reviewer"
	Assessment  = "qs:evaluation:collection:assessments"
	Testee      = "qs:actor:collection:testees"
)

type Role struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Protection string `json:"protection"`
}
type Assignment struct {
	ID      string `json:"id"`
	Subject string `json:"subject"`
	Role    string `json:"role"`
}
type Edge struct {
	ID     string `json:"id"`
	Role   string `json:"role"`
	Parent string `json:"parent"`
}

// Person must be obtained from authoritative IAM users and QS staff/clinician
// rows, including explicit negative evidence when no active clinician exists.
type Person struct {
	Subject          string   `json:"subject"`
	EvidenceHash     string   `json:"evidence_hash"`
	Active           bool     `json:"active"`
	ClinicianChecked bool     `json:"clinician_checked"`
	ClinicianIDs     []string `json:"clinician_ids"`
	IdentityIssue    string   `json:"identity_issue,omitempty"`
}
type Permission struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
	Origin   string `json:"origin,omitempty"`
}
type Grant struct {
	ID         string     `json:"id"`
	Role       string     `json:"role"`
	Permission Permission `json:"permission"`
}
type Resource struct {
	Key     string   `json:"key"`
	Actions []string `json:"actions"`
}
type Input struct {
	Version int64 `json:"version"`
	// FactsHash fingerprints complete persisted rows, not only this projection.
	FactsHash   string       `json:"facts_hash"`
	Roles       []Role       `json:"roles"`
	Assignments []Assignment `json:"assignments"`
	Edges       []Edge       `json:"edges"`
	People      []Person     `json:"people"`
	Grants      []Grant      `json:"grants"`
	Resources   []Resource   `json:"resources"`
}
type Mapping struct {
	Subject      string       `json:"subject"`
	Before       []string     `json:"before"`
	After        []string     `json:"after"`
	ClinicianIDs []string     `json:"clinician_ids,omitempty"`
	Reasons      []string     `json:"reasons"`
	Added        []Permission `json:"added"`
	Removed      []Permission `json:"removed"`
}
type Plan struct {
	MigrationID  string                  `json:"migration_id"`
	Fingerprint  string                  `json:"fingerprint"`
	Ready        bool                    `json:"ready"`
	Issues       []string                `json:"issues"`
	People       []Mapping               `json:"people"`
	Grants       map[string][]Permission `json:"grants"`
	ArchiveRoles []string                `json:"archive_roles"`
	RemoveEdges  []Edge                  `json:"remove_edges"`
}

var knownRoles = map[string]bool{"platform_admin": true, "iam_admin": true, "qs:admin": true, "qs:content_manager": true, "qs:evaluation_plan_manager": true, "qs:evaluator": true, "qs:staff": true, "super_admin": true, "user": true}
var knownEdges = map[string]bool{
	"super_admin>iam_admin": true, "super_admin>qs:admin": true, "iam_admin>user": true,
	"qs:admin>qs:content_manager": true, "qs:admin>qs:evaluator": true, "qs:admin>qs:evaluation_plan_manager": true,
	"qs:evaluator>qs:staff": true, "qs:evaluation_plan_manager>qs:staff": true,
}

// Build is deterministic, side-effect free and refuses unknown source facts.
// New target roles are created by apply; encountering them without a migration
// receipt is a conflict, never permission to overwrite an existing role.
func Build(in Input) Plan {
	encoded, _ := json.Marshal(in)
	var owned Input
	_ = json.Unmarshal(encoded, &owned)
	in = owned
	p := Plan{MigrationID: MigrationID, Issues: []string{}, People: []Mapping{}, Grants: map[string][]Permission{}, ArchiveRoles: []string{"qs:evaluator", "qs:staff", "super_admin"}, RemoveEdges: append([]Edge{}, in.Edges...)}
	canonicalize(&in)
	raw, _ := json.Marshal(in)
	sum := sha256.Sum256(raw)
	p.Fingerprint = hex.EncodeToString(sum[:])
	issue := func(s string) { p.Issues = append(p.Issues, s) }
	roles := map[string]Role{}
	for _, r := range in.Roles {
		if !knownRoles[r.Name] {
			issue("unexpected role: " + r.Name)
		}
		if _, ok := roles[r.Name]; ok {
			issue("duplicate role: " + r.Name)
		}
		roles[r.Name] = r
		if r.Protection != "standard" && r.Protection != "protected" {
			issue("invalid protection: " + r.Name)
		}
	}
	for name := range knownRoles {
		if _, ok := roles[name]; !ok {
			issue("missing source role: " + name)
		}
	}
	if r, ok := roles["platform_admin"]; ok && r.Protection != "protected" {
		issue("platform_admin must remain protected")
	}
	parents := map[string][]string{}
	seenEdge := map[string]bool{}
	for _, e := range in.Edges {
		key := e.Role + ">" + e.Parent
		if !knownEdges[key] || seenEdge[key] {
			issue("unexpected or duplicate inheritance: " + key)
		}
		seenEdge[key] = true
		if _, ok := roles[e.Role]; !ok {
			issue("inheritance missing role: " + e.Role)
		}
		if _, ok := roles[e.Parent]; !ok {
			issue("inheritance missing parent: " + e.Parent)
		}
		parents[e.Role] = append(parents[e.Role], e.Parent)
	}
	source := map[string][]Permission{}
	for _, g := range in.Grants {
		if _, ok := roles[g.Role]; !ok {
			issue("grant references unknown role: " + g.ID)
		}
		source[g.Role] = append(source[g.Role], g.Permission)
	}
	// Explicitly lock the legacy roles being split, so a newly added permission
	// cannot disappear during migration merely because a role name is familiar.
	expected := legacyPermissions()
	for _, name := range []string{"qs:evaluator", "qs:staff", "qs:evaluation_plan_manager", "user"} {
		if !samePermissions(source[name], expected[name]) {
			issue("source grant contract changed: " + name)
		}
	}
	if !samePermissions(source["qs:admin"], []Permission{{Resource: "qs:*:*:*", Action: "*"}}) {
		issue("qs:admin wildcard contract changed")
	}
	if !samePermissions(source["platform_admin"], []Permission{{Resource: "*:*:*:*", Action: "*"}}) {
		issue("platform_admin wildcard contract changed")
	}
	if len(source["super_admin"]) != 0 {
		issue("super_admin has unexpected direct permissions")
	}
	for _, name := range []string{"platform_admin", "iam_admin", "qs:admin", "qs:content_manager", "user", "qs:evaluation_plan_manager"} {
		p.Grants[name] = append([]Permission{}, source[name]...)
	}
	// The retired management resource is intentionally removed from iam_admin.
	for name, grants := range p.Grants {
		kept := []Permission{}
		for _, g := range grants {
			if g.Resource != "iam:authz:collection:role_inheritances" {
				kept = append(kept, g)
			}
		}
		p.Grants[name] = kept
	}
	p.Grants["iam_admin"] = append(p.Grants["iam_admin"], source["user"]...)
	p.Grants[Operator] = permissions(Testee, "read", "list")
	p.Grants[Operator] = append(p.Grants[Operator], permissions(Assessment, "read_progress", "list_progress", "batch_evaluate")...)
	p.Grants[Operator] = append(p.Grants[Operator], Permission{Resource: Assessment, Action: "retry", Origin: "adhoc"})
	p.Grants[Reviewer] = permissions(Testee, "read", "list", "analyze", "statistics")
	p.Grants[Reviewer] = append(p.Grants[Reviewer], permissions("qs:answersheet:collection:answersheets", "read", "list", "statistics")...)
	p.Grants[Reviewer] = append(p.Grants[Reviewer], permissions(Assessment, "read", "list", "statistics")...)
	p.Grants[Reviewer] = append(p.Grants[Reviewer], permissions("qs:evaluation:collection:reports", "read", "list")...)
	p.Grants["qs:evaluation_plan_manager"] = append(p.Grants["qs:evaluation_plan_manager"], permissions(Testee, "read", "list")...)
	p.Grants["qs:evaluation_plan_manager"] = append(p.Grants["qs:evaluation_plan_manager"], permissions(Assessment, "read_progress", "list_progress")...)
	catalog := map[string]map[string]bool{}
	for _, r := range in.Resources {
		if catalog[r.Key] != nil {
			issue("duplicate resource: " + r.Key)
		}
		catalog[r.Key] = map[string]bool{}
		for _, a := range r.Actions {
			catalog[r.Key][a] = true
		}
	}
	for name, gs := range p.Grants {
		p.Grants[name] = uniquePermissions(gs)
		for _, g := range p.Grants[name] {
			if strings.Contains(g.Resource, "*") {
				continue
			}
			if g.Resource == Assessment && (g.Action == "read_progress" || g.Action == "list_progress") && catalog[Assessment] != nil {
				continue
			}
			if !catalog[g.Resource][g.Action] {
				issue("missing catalog action: " + g.Resource + "/" + g.Action)
			}
		}
	}
	people := map[string]Person{}
	for _, person := range in.People {
		if _, ok := people[person.Subject]; ok {
			issue("duplicate identity: " + person.Subject)
		}
		people[person.Subject] = person
	}
	direct := map[string][]string{}
	seenAssignment := map[string]bool{}
	for _, a := range in.Assignments {
		if _, ok := roles[a.Role]; !ok {
			issue("assignment references unknown role: " + a.ID)
		}
		if !strings.HasPrefix(a.Subject, "user:") {
			issue("unsupported subject: " + a.Subject)
		}
		key := a.Subject + "/" + a.Role
		if seenAssignment[key] {
			issue("duplicate assignment: " + key)
		}
		seenAssignment[key] = true
		direct[a.Subject] = append(direct[a.Subject], a.Role)
	}
	subjects := []string{}
	for s := range direct {
		subjects = append(subjects, s)
	}
	sort.Strings(subjects)
	for _, s := range subjects {
		person, ok := people[s]
		if !ok || !person.Active || person.IdentityIssue != "" {
			issue("invalid authoritative identity: " + s + " " + person.IdentityIssue)
		}
		m := Mapping{Subject: s, Before: uniqueStrings(direct[s]), After: []string{}, Reasons: []string{}, Added: []Permission{}, Removed: []Permission{}}
		for _, name := range m.Before {
			switch name {
			case "qs:evaluator":
				m.After = append(m.After, Operator, Reviewer)
				m.Reasons = append(m.Reasons, "split_evaluator")
			case "qs:staff":
				m.After = append(m.After, Operator)
				m.Reasons = append(m.Reasons, "staff_to_operator")
				if !person.ClinicianChecked || len(person.ClinicianIDs) > 1 {
					issue("unresolved clinician mapping: " + s)
				}
				if len(person.ClinicianIDs) == 1 {
					m.After = append(m.After, Reviewer)
					m.ClinicianIDs = person.ClinicianIDs
					m.Reasons = append(m.Reasons, "active_clinician_results")
				}
			case "super_admin":
				m.After = append(m.After, "iam_admin", "qs:admin")
				m.Reasons = append(m.Reasons, "expand_super_admin")
			default:
				m.After = append(m.After, name)
			}
		}
		m.After = uniqueStrings(m.After)
		old := effective(m.Before, parents, source)
		next := effective(m.After, nil, p.Grants)
		m.Added = difference(next, old)
		allowedAdditions := []Permission{}
		for _, name := range m.Before {
			switch name {
			case "qs:staff":
				allowedAdditions = append(allowedAdditions, p.Grants[Operator]...)
				if len(person.ClinicianIDs) == 1 {
					allowedAdditions = append(allowedAdditions, p.Grants[Reviewer]...)
				}
			case "qs:evaluator", "qs:evaluation_plan_manager":
				allowedAdditions = append(allowedAdditions, permissions(Assessment, "read_progress", "list_progress")...)
			}
		}
		for _, q := range difference(m.Added, allowedAdditions) {
			issue("unexpected permission addition: " + s + " " + permissionKey(q))
		}
		m.Removed = difference(old, next)
		for _, q := range m.Removed {
			if q.Resource != "iam:authz:collection:role_inheritances" {
				issue("unexpected permission removal: " + s + " " + permissionKey(q))
			}
		}
		p.People = append(p.People, m)
	}
	sort.Strings(p.Issues)
	p.Issues = uniqueStrings(p.Issues)
	p.Ready = len(p.Issues) == 0
	return p
}
func permissions(key string, actions ...string) []Permission {
	out := []Permission{}
	for _, a := range actions {
		out = append(out, Permission{Resource: key, Action: a})
	}
	return out
}
func legacyPermissions() map[string][]Permission {
	m := map[string][]Permission{"qs:staff": permissions(Testee, "read", "list"), "user": permissions("iam:identity:instance:profile", "read", "update")}
	m["qs:evaluator"] = append(permissions(Testee, "analyze", "statistics"), permissions("qs:answersheet:collection:answersheets", "read", "list", "statistics")...)
	m["qs:evaluator"] = append(m["qs:evaluator"], permissions(Assessment, "batch_evaluate", "read", "list", "statistics")...)
	m["qs:evaluator"] = append(m["qs:evaluator"], Permission{Resource: Assessment, Action: "retry", Origin: "adhoc"})
	m["qs:evaluator"] = append(m["qs:evaluator"], permissions("qs:evaluation:collection:reports", "read", "list")...)
	m["qs:evaluation_plan_manager"] = permissions("qs:plan:collection:evaluation_plans", "create", "read", "list", "update", "pause", "resume", "cancel", "enroll", "terminate", "statistics")
	m["qs:evaluation_plan_manager"] = append(m["qs:evaluation_plan_manager"], permissions("qs:plan_task:collection:evaluation_plan_tasks", "schedule", "read", "list", "open", "complete", "expire", "cancel")...)
	m["qs:evaluation_plan_manager"] = append(m["qs:evaluation_plan_manager"], Permission{Resource: Assessment, Action: "retry", Origin: "plan"})
	return m
}
func permissionKey(p Permission) string { return p.Resource + "\x00" + p.Action + "\x00" + p.Origin }
func uniquePermissions(ps []Permission) []Permission {
	m := map[string]Permission{}
	for _, p := range ps {
		m[permissionKey(p)] = p
	}
	keys := []string{}
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := []Permission{}
	for _, k := range keys {
		out = append(out, m[k])
	}
	return out
}
func samePermissions(a, b []Permission) bool {
	aa, _ := json.Marshal(uniquePermissions(a))
	bb, _ := json.Marshal(uniquePermissions(b))
	return string(aa) == string(bb)
}
func uniqueStrings(xs []string) []string {
	m := map[string]bool{}
	for _, s := range xs {
		m[s] = true
	}
	out := []string{}
	for s := range m {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
func effective(names []string, parents map[string][]string, grants map[string][]Permission) []Permission {
	seen := map[string]bool{}
	queue := append([]string{}, names...)
	out := []Permission{}
	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]
		if seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, grants[n]...)
		queue = append(queue, parents[n]...)
	}
	return uniquePermissions(out)
}
func covers(have, want Permission) bool {
	hp, wp := strings.Split(have.Resource, ":"), strings.Split(want.Resource, ":")
	if len(hp) != 4 || len(wp) != 4 {
		return false
	}
	for i := range hp {
		if hp[i] != "*" && hp[i] != wp[i] {
			return false
		}
	}
	return (have.Action == "*" || have.Action == want.Action) && (have.Origin == "" || have.Origin == want.Origin)
}
func difference(a, b []Permission) []Permission {
	out := []Permission{}
	for _, p := range a {
		found := false
		for _, q := range b {
			if covers(q, p) {
				found = true
				break
			}
		}
		if !found {
			out = append(out, p)
		}
	}
	return uniquePermissions(out)
}
func canonicalize(in *Input) {
	sort.Slice(in.Roles, func(i, j int) bool { return in.Roles[i].Name < in.Roles[j].Name })
	sort.Slice(in.Assignments, func(i, j int) bool { return in.Assignments[i].ID < in.Assignments[j].ID })
	sort.Slice(in.Edges, func(i, j int) bool { return in.Edges[i].ID < in.Edges[j].ID })
	sort.Slice(in.People, func(i, j int) bool { return in.People[i].Subject < in.People[j].Subject })
	sort.Slice(in.Grants, func(i, j int) bool { return in.Grants[i].ID < in.Grants[j].ID })
	sort.Slice(in.Resources, func(i, j int) bool { return in.Resources[i].Key < in.Resources[j].Key })
	for i := range in.People {
		in.People[i].ClinicianIDs = uniqueStrings(in.People[i].ClinicianIDs)
	}
	for i := range in.Resources {
		in.Resources[i].Actions = uniqueStrings(in.Resources[i].Actions)
	}
}
func (p Plan) Validate() error {
	if !p.Ready {
		return fmt.Errorf("role migration preflight failed: %s", strings.Join(p.Issues, "; "))
	}
	return nil
}

package rolemodel

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func fixture() Input {
	in := Input{Version: 11, FactsHash: "complete-row-fingerprint"}
	for name := range knownRoles {
		protection := "standard"
		if name == "platform_admin" {
			protection = "protected"
		}
		in.Roles = append(in.Roles, Role{ID: name, Name: name, Protection: protection})
	}
	for key := range knownEdges {
		for i := range key {
			if key[i] == '>' {
				in.Edges = append(in.Edges, Edge{ID: key, Role: key[:i], Parent: key[i+1:]})
				break
			}
		}
	}
	grants := legacyPermissions()
	grants["qs:admin"] = []Permission{{Resource: "qs:*:*:*", Action: "*"}}
	grants["platform_admin"] = []Permission{{Resource: "*:*:*:*", Action: "*"}}
	catalog := map[string][]string{}
	for name, gs := range grants {
		for i, g := range gs {
			in.Grants = append(in.Grants, Grant{ID: fmt.Sprintf("%s-%d", name, i), Role: name, Permission: g})
			catalog[g.Resource] = append(catalog[g.Resource], g.Action)
		}
	}
	for k, as := range catalog {
		in.Resources = append(in.Resources, Resource{Key: k, Actions: uniqueStrings(as)})
	}
	return in
}
func assign(in *Input, subject string, roles ...string) {
	in.People = append(in.People, Person{Subject: subject, Active: true, ClinicianChecked: true})
	for _, r := range roles {
		in.Assignments = append(in.Assignments, Assignment{ID: subject + r, Subject: subject, Role: r})
	}
}
func TestMigrationUsesDirectRolesAndExplicitClinicianEvidence(t *testing.T) {
	in := fixture()
	assign(&in, "user:1", "qs:staff")
	assign(&in, "user:2", "qs:staff")
	in.People[1].ClinicianIDs = []string{"doctor-2"}
	assign(&in, "user:3", "qs:evaluator")
	assign(&in, "user:4", "qs:evaluation_plan_manager")
	assign(&in, "user:5", "qs:admin")
	assign(&in, "user:6", "qs:staff", "qs:evaluator")
	p := Build(in)
	require.NoError(t, p.Validate())
	got := map[string][]string{}
	for _, m := range p.People {
		got[m.Subject] = m.After
	}
	require.Equal(t, []string{Operator}, got["user:1"])
	require.Equal(t, []string{Operator, Reviewer}, got["user:2"])
	require.Equal(t, []string{Operator, Reviewer}, got["user:3"])
	require.Equal(t, []string{"qs:evaluation_plan_manager"}, got["user:4"])
	require.Equal(t, []string{"qs:admin"}, got["user:5"])
	require.Equal(t, []string{Operator, Reviewer}, got["user:6"])
	require.NotContains(t, p.Grants[Operator], Permission{Resource: Assessment, Action: "read"})
	require.NotContains(t, p.Grants[Reviewer], Permission{Resource: Assessment, Action: "batch_evaluate"})
	require.Contains(t, p.Grants[Operator], Permission{Resource: Assessment, Action: "retry", Origin: "adhoc"})
	require.Contains(t, p.Grants["qs:evaluation_plan_manager"], Permission{Resource: Assessment, Action: "retry", Origin: "plan"})
}
func TestPreflightRefusesAmbiguousIdentitiesAndUnexpectedFacts(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Input)
	}{
		{"unchecked clinician", func(in *Input) { in.People[0].ClinicianChecked = false }},
		{"ambiguous clinician", func(in *Input) { in.People[0].ClinicianIDs = []string{"a", "b"} }},
		{"inactive user", func(in *Input) { in.People[0].Active = false }},
		{"unknown role", func(in *Input) { in.Roles = append(in.Roles, Role{ID: "new", Name: "new", Protection: "standard"}) }},
		{"new target collision", func(in *Input) { in.Roles = append(in.Roles, Role{ID: "new", Name: Operator, Protection: "standard"}) }},
		{"unknown edge", func(in *Input) { in.Edges = append(in.Edges, Edge{ID: "new", Role: "qs:staff", Parent: "qs:admin"}) }},
		{"changed source grant", func(in *Input) {
			in.Grants = append(in.Grants, Grant{ID: "new", Role: "qs:evaluator", Permission: Permission{Resource: Assessment, Action: "force_retry"}})
		}},
		{"missing source grant", func(in *Input) { in.Grants = in.Grants[1:] }},
		{"missing catalog", func(in *Input) { in.Resources = nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := fixture()
			assign(&in, "user:1", "qs:staff")
			tc.mutate(&in)
			require.Error(t, Build(in).Validate())
		})
	}
}
func TestPreflightFingerprintIndependentOfRowOrderAndDoesNotMutateInput(t *testing.T) {
	in := fixture()
	assign(&in, "user:1", "qs:staff")
	before, _ := json.Marshal(in)
	p := Build(in)
	after, _ := json.Marshal(in)
	require.Equal(t, string(before), string(after))
	for i, j := 0, len(in.Grants)-1; i < j; i, j = i+1, j-1 {
		in.Grants[i], in.Grants[j] = in.Grants[j], in.Grants[i]
	}
	require.Equal(t, p.Fingerprint, Build(in).Fingerprint)
	in.FactsHash = "changed persisted audit field"
	require.NotEqual(t, p.Fingerprint, Build(in).Fingerprint)
}

func TestPreflightRefusesUnapprovedExpansionWhenInheritanceWasRemoved(t *testing.T) {
	in := fixture()
	assign(&in, "user:1", "iam_admin")
	kept := []Edge{}
	for _, edge := range in.Edges {
		if edge.Role != "iam_admin" {
			kept = append(kept, edge)
		}
	}
	in.Edges = kept
	require.ErrorContains(t, Build(in).Validate(), "unexpected permission addition")
}

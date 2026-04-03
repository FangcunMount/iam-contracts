package main

import (
	"testing"

	assignmentDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authz/assignment"
	policyDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authz/policy"
)

func TestNormalizePolicyActionPattern(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{name: "wildcard", input: "*", expect: ".*"},
		{name: "regex kept", input: "read|update", expect: "read|update"},
		{name: "trimmed", input: "  .*  ", expect: ".*"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := normalizePolicyActionPattern(tc.input); got != tc.expect {
				t.Fatalf("normalizePolicyActionPattern(%q) = %q, want %q", tc.input, got, tc.expect)
			}
		})
	}
}

func TestBuildDesiredPolicyState(t *testing.T) {
	t.Parallel()

	state, err := buildDesiredPolicyState(
		[]PolicyConfig{
			{Type: "p", Subject: "role:super_admin", Values: []string{"platform", "*", "*"}},
			{Type: "g", Subject: "role:qs:admin", Values: []string{"role:qs:evaluator", "1"}},
		},
		[]RoleConfig{
			{Name: "super_admin", TenantID: "platform"},
			{Name: "tenant_admin", TenantID: "fangcun"},
			{Name: "qs:admin", TenantID: "1"},
			{Name: "qs:evaluator", TenantID: "1"},
		},
	)
	if err != nil {
		t.Fatalf("buildDesiredPolicyState returned error: %v", err)
	}

	policyKey := policyRuleKey(policyDomain.PolicyRule{
		Sub: "role:super_admin",
		Dom: "platform",
		Obj: "*",
		Act: ".*",
	})
	if _, ok := state.Policies[policyKey]; !ok {
		t.Fatalf("expected normalized policy %q to exist", policyKey)
	}

	groupingKey := groupingRuleKey(policyDomain.GroupingRule{
		Sub:  "role:qs:admin",
		Role: "role:qs:evaluator",
		Dom:  "1",
	})
	if _, ok := state.Groupings[groupingKey]; !ok {
		t.Fatalf("expected grouping %q to exist", groupingKey)
	}

	if _, ok := state.ManagedRoleKey["role:qs:admin"]; !ok {
		t.Fatalf("expected managed role set to include inherited role")
	}
	if _, ok := state.ManagedRoleKey["role:qs:evaluator"]; !ok {
		t.Fatalf("expected managed role set to include descendant role")
	}
	if _, ok := state.ManagedTenant["platform"]; !ok {
		t.Fatalf("expected managed tenant set to include platform")
	}
	if _, ok := state.ManagedTenant["1"]; !ok {
		t.Fatalf("expected managed tenant set to include org-scoped qs domain")
	}
}

func TestIsManagedGroupingRuleSkipsUserAssignments(t *testing.T) {
	t.Parallel()

	managedRoles := map[string]struct{}{
		"role:qs:admin":                   {},
		"role:qs:content_manager":         {},
		"role:qs:evaluation_plan_manager": {},
	}
	managedTenants := map[string]struct{}{"1": {}}

	if isManagedGroupingRule(policyDomain.GroupingRule{
		Sub:  "user:110001",
		Role: "role:qs:admin",
		Dom:  "1",
	}, managedRoles, managedTenants) {
		t.Fatalf("user-role grouping should not be managed by policy sync")
	}

	if !isManagedGroupingRule(policyDomain.GroupingRule{
		Sub:  "role:qs:admin",
		Role: "role:qs:content_manager",
		Dom:  "1",
	}, managedRoles, managedTenants) {
		t.Fatalf("role-role grouping should be managed by policy sync")
	}
}

func TestParseAssignmentSubjectType(t *testing.T) {
	t.Parallel()

	subjectType, err := parseAssignmentSubjectType("")
	if err != nil {
		t.Fatalf("parseAssignmentSubjectType default returned error: %v", err)
	}
	if subjectType != assignmentDomain.SubjectTypeUser {
		t.Fatalf("default subject type = %q, want %q", subjectType, assignmentDomain.SubjectTypeUser)
	}

	if _, err := parseAssignmentSubjectType("service"); err == nil {
		t.Fatalf("expected unsupported subject type to fail")
	}
}

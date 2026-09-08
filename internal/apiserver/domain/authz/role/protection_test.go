package role

import (
	"testing"

	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/resource"
	"github.com/stretchr/testify/require"
)

func TestStandardRoleCannotAcquireSensitiveCapabilities(t *testing.T) {
	for _, tc := range []struct {
		pattern, action string
		sensitive       bool
	}{
		{"iam:authz:collection:roles", "manage_protected", true},
		{"iam:authz:collection:resources", "*", true},
		{"iam:*:*:*", "create", true},
		{"*:*:*:*", "*", true},
		{"iam:identity:collection:profiles", "list_all", true},
		{"iam:identity:collection:profiles", "search_by_mobile_all", true},
		{"iam:identity:collection:profiles", "list", false},
		{"iam:identity:collection:profiles", "search_by_mobile", false},
		{"qs:*:*:*", "*", false},
		{"iam:authz:collection:roles", "create", false},
	} {
		t.Run(tc.pattern+"/"+tc.action, func(t *testing.T) {
			r, err := NewRole("operator", "操作员")
			require.NoError(t, err)
			err = r.ValidateGrant(resource.Pattern(tc.pattern), resource.ActionPattern(tc.action))
			if tc.sensitive {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			r.ManagementProtection = ManagementProtected
			require.NoError(t, r.ValidateGrant(resource.Pattern(tc.pattern), resource.ActionPattern(tc.action)))
		})
	}
}

func TestInvalidRoleProtectionRejected(t *testing.T) {
	_, err := NewRole("operator", "操作员", WithManagementProtection("unknown"))
	require.Error(t, err)
}

package maintenance

import (
	"testing"
	"time"

	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/role"
	rolerepo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/role"
	base "github.com/FangcunMount/iam/v5/internal/pkg/database/mysql"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"github.com/stretchr/testify/require"
)

func legacyRole(id uint64, domain, name string) retirementRole {
	return retirementRole{RolePO: rolerepo.RolePO{AuditFields: base.AuditFields{ID: meta.ID(id)}, Name: name}, LegacyDomain: domain}
}
func TestRetirementPreservesIDsAndMapsProtection(t *testing.T) {
	r := analyzeRetirement(retirementState{Roles: []retirementRole{legacyRole(7, "platform", "super_admin"), legacyRole(8, "fangcun", "super_admin"), legacyRole(9, "fangcun", "tenant_admin")}, Versions: []retirementVersion{{ID: 1, LegacyDomain: "platform", PolicyVersion: 12}, {ID: 2, LegacyDomain: "fangcun", PolicyVersion: 20}}})
	require.True(t, r.Ready, r.Issues)
	require.EqualValues(t, 21, r.InitialPolicyVersion)
	require.Equal(t, []RetirementRoleChange{{meta.ID(7), "super_admin", "platform_admin", role.ManagementProtected}, {meta.ID(8), "super_admin", "super_admin", role.ManagementStandard}, {meta.ID(9), "tenant_admin", "iam_admin", role.ManagementStandard}}, r.Roles)
}
func TestRetirementRejectsHistoricalUnknownDomainAndNameConflict(t *testing.T) {
	deleted := time.Now()
	old := legacyRole(2, "unknown", "reader")
	old.DeletedAt = &deleted
	for _, state := range []retirementState{
		{Roles: []retirementRole{old}},
		{Roles: []retirementRole{legacyRole(1, "platform", "super_admin"), legacyRole(2, "fangcun", "platform_admin")}},
		{Roles: []retirementRole{legacyRole(1, "platform", "reader"), legacyRole(2, "fangcun", "reader")}},
	} {
		r := analyzeRetirement(state)
		require.False(t, r.Ready)
		require.NotEmpty(t, r.Issues)
	}
}

package roleinheritance_test

import (
	"testing"

	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/roleinheritance"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"github.com/stretchr/testify/require"
)

func TestGraphDepthAndReferences(t *testing.T) {
	roles := make([]roleinheritance.RoleNode, 33)
	edges := make([]*roleinheritance.Inheritance, 0, 32)
	for i := range roles {
		roles[i] = roleinheritance.RoleNode{ID: meta.ID(i + 1)}
		if i > 0 {
			edges = append(edges, &roleinheritance.Inheritance{RoleID: meta.ID(i), InheritedRoleID: meta.ID(i + 1)})
		}
	}
	require.NoError(t, roleinheritance.ValidateGraph(roles[:32], edges[:31]))
	require.Error(t, roleinheritance.ValidateGraph(roles, edges))
	require.Error(t, roleinheritance.ValidateGraph(roles[:32], edges))
	roles[31].ID = 999
	require.Error(t, roleinheritance.ValidateGraph(roles[:32], edges[:31]))
	roles[31].ID = 32
	edges[0].InheritedRoleID = edges[0].RoleID
	require.Error(t, roleinheritance.ValidateGraph(roles[:32], edges[:31]))
	edges[0].InheritedRoleID = 2
	edges[1].InheritedRoleID = 1
	require.Error(t, roleinheritance.ValidateGraph(roles[:32], edges[:31]))
}

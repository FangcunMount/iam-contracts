package runtime

import (
	"testing"

	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/subject"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"github.com/stretchr/testify/require"
)

func TestRoleGraphResolvesStableRoleIDs(t *testing.T) {
	user, err := subject.NewUserRef(meta.ID(42))
	require.NoError(t, err)
	group, err := subject.NewRef(subject.TypeGroup, meta.ID(42))
	require.NoError(t, err)
	b := newRoleGraphBuilder()
	b.addAssignment(user, 10)
	b.addAssignment(user, 20)
	b.addAssignment(user, 10)
	b.addAssignment(group, 30)
	b.addInheritance(10, 30)
	b.addInheritance(20, 30)
	b.addInheritance(30, 40)
	b.addInheritance(30, 40)
	g := b.build(maxRoleHierarchyLevel)
	direct, err := g.DirectRoles(user)
	require.NoError(t, err)
	require.Equal(t, []meta.ID{10, 20}, direct)
	effective, err := g.EffectiveRoles(user)
	require.NoError(t, err)
	require.Equal(t, []meta.ID{10, 20, 30, 40}, effective)
	direct[0] = 99
	direct, err = g.DirectRoles(user)
	require.NoError(t, err)
	require.Equal(t, []meta.ID{10, 20}, direct)
	direct, err = g.DirectRoles(group)
	require.NoError(t, err)
	require.Equal(t, []meta.ID{30}, direct)
	unknown, err := subject.NewUserRef(meta.ID(100))
	require.NoError(t, err)
	effective, err = g.EffectiveRoles(unknown)
	require.NoError(t, err)
	require.Empty(t, effective)
	require.NotNil(t, effective)
}
func TestRoleGraphPreservesMaximumHierarchyLevel(t *testing.T) {
	sub, err := subject.NewUserRef(meta.ID(42))
	require.NoError(t, err)
	b := newRoleGraphBuilder()
	b.addAssignment(sub, 1)
	for i := 1; i <= maxRoleHierarchyLevel; i++ {
		b.addInheritance(meta.ID(i), meta.ID(i+1))
	}
	got, err := b.build(maxRoleHierarchyLevel).EffectiveRoles(sub)
	require.Error(t, err)
	require.Nil(t, got)
}

package runtime

import (
	"testing"

	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/subject"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"github.com/stretchr/testify/require"
)

func TestDirectAssignmentsAreImmutableDeduplicatedAndSubjectScoped(t *testing.T) {
	user, err := subject.NewUserRef(meta.ID(42))
	require.NoError(t, err)
	group, err := subject.NewRef(subject.TypeGroup, meta.ID(42))
	require.NoError(t, err)
	b := newDirectRoleBuilder()
	b.addAssignment(user, 10)
	b.addAssignment(user, 20)
	b.addAssignment(user, 10)
	b.addAssignment(group, 30)
	g := b.build()
	direct, err := g.DirectRoles(user)
	require.NoError(t, err)
	require.Equal(t, []meta.ID{10, 20}, direct)
	effective, err := g.EffectiveRoles(user)
	require.NoError(t, err)
	require.Equal(t, []meta.ID{10, 20}, effective)
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

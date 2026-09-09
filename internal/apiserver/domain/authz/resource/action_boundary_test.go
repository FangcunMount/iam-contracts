package resource

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCatalogActionBoundaries(t *testing.T) {
	for _, value := range []string{"*", "read_*", "read|list", ".*"} {
		t.Run(value, func(t *testing.T) {
			_, err := NewResource("example:catalog:collection:documents", []string{value}, WithDisplayName("Documents"))
			require.Error(t, err)
			_, err = RestoreResource("example:catalog:collection:documents", []string{value})
			require.Error(t, err)
			r, err := NewResource("example:catalog:collection:documents", []string{"read"}, WithDisplayName("Documents"))
			require.NoError(t, err)
			require.Error(t, r.ChangeCatalog([]string{value}))
			require.Equal(t, []string{"read"}, r.ActionStrings())
			require.False(t, r.HasAction(value))
		})
	}
}

func TestUnifiedActionConstructionAndMatching(t *testing.T) {
	for _, value := range []string{"review", "batch_update", "*"} {
		a, err := NewAction(" " + value + " ")
		require.NoError(t, err)
		require.Equal(t, value, a.String())
		require.Equal(t, value == "*", a.IsWildcard())
	}
	for _, value := range []string{"", "read_*", "read|list", ".*"} {
		_, err := NewAction(value)
		require.Error(t, err)
	}
	for _, value := range []string{"*", "", "read_*", "read|list", ".*", " read "} {
		require.Error(t, Action(value).ValidateConcrete())
	}
	require.NoError(t, Action("review").ValidateConcrete())
	require.True(t, Action("review").Matches(Action("review")))
	require.False(t, Action("review").Matches(Action("read")))
	require.True(t, WildcardAction.Matches(Action("review")))
	require.False(t, Action("review").Matches(WildcardAction))
	require.False(t, WildcardAction.Matches(Action("")))
}

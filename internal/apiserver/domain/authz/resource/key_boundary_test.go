package resource

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestResourceTargetKeyBoundaries(t *testing.T) {
	for _, key := range []string{"qs:*:*:*", "qs:assessment:*:*", "*:*:*:*"} {
		_, err := NewResource(key, []string{"read"}, WithDisplayName("Target"))
		require.Error(t, err)
		_, err = RestoreResource(key, []string{"read"})
		require.Error(t, err)
	}
	_, err := NewResource("qs:assessment:report:*", []string{"read"}, WithDisplayName("Reports"))
	require.NoError(t, err)
}

func TestUnifiedResourceKeyRanges(t *testing.T) {
	for _, value := range []string{"qs:assessment:collection:reports", "qs:assessment:*:*", "qs:*:*:*", "*:*:*:*"} {
		k, err := NewKey(" " + value + " ")
		require.NoError(t, err)
		require.Equal(t, value, k.String())
		require.True(t, k.Covers(Key("qs:assessment:collection:reports")))
	}
	require.False(t, Key("qs:*:*:*").Covers(Key("iam:authz:collection:roles")))
	require.False(t, Key("qs:assessment:collection:reports").Covers(Key("qs:*:*:*")))
	for _, value := range []string{"", "qs:*", "qs:assessment:collection:report_*", "qs: assessment:collection:reports"} {
		_, err := NewKey(value)
		require.Error(t, err)
		require.Error(t, Key(value).ValidateTarget())
		require.False(t, Key(value).Covers(Key("qs:assessment:collection:reports")))
	}
	for _, value := range []string{"qs:*:*:*", "qs:assessment:*:*", "*:*:*:*"} {
		require.Error(t, Key(value).ValidateTarget())
	}
	require.NoError(t, Key("qs:assessment:report:*").ValidateTarget())
	app, ok := AppNameFromKey("qs:*:*:*")
	require.True(t, ok)
	require.Equal(t, "qs", app)
	_, ok = AppNameFromKey("*:*:*:*")
	require.False(t, ok)
}

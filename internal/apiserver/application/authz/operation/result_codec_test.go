package operation

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResultRestoresOriginalProjectionWithoutBusinessOrNewIdentities(t *testing.T) {
	type projection struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	original := projection{ID: "42", Name: "first"}
	captures := 0
	codec := ProjectResult("iam.role-result/v1", func() (projection, error) { captures++; return original, nil }, func(value projection) error { original = value; return nil })
	raw, err := codec.Capture()
	require.NoError(t, err)
	require.Equal(t, 1, captures)
	original = projection{ID: "999", Name: "changed"}
	require.NoError(t, codec.Restore(raw))
	require.Equal(t, projection{ID: "42", Name: "first"}, original)
	require.Equal(t, 1, captures, "restore repeated business capture")
	for _, bad := range [][]byte{append(bytes.Clone(raw), ' '), bytes.Replace(raw, []byte(`"id":"42"`), []byte(`"id":"42","id":"42"`), 1), bytes.Replace(raw, []byte(`"name":"first"`), []byte(`"name":"first","secret":"unexpected"`), 1), bytes.Replace(raw, []byte("iam.role-result/v1"), []byte("iam.role-result/v2"), 1)} {
		require.ErrorIs(t, codec.Restore(bad), ErrInvalid)
	}
}

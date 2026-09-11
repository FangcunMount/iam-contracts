package authorization

import (
	"testing"

	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/subject"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"github.com/stretchr/testify/require"
)

func TestRequestRequiresConcreteAction(t *testing.T) {
	sub, err := subject.NewUserRef(meta.FromUint64(1))
	require.NoError(t, err)
	for _, value := range []string{"*", "read_*", "read|list", ".*", ""} {
		_, err := NewRequest(sub, "example:catalog:collection:documents", value)
		require.Error(t, err, value)
	}
	req, err := NewRequest(sub, "example:catalog:collection:documents", " read ")
	require.NoError(t, err)
	require.Equal(t, "read", req.Action.String())
}

func TestRequestResourceTargetBoundaries(t *testing.T) {
	sub, err := subject.NewUserRef(meta.FromUint64(1))
	require.NoError(t, err)
	for _, key := range []string{"qs:*:*:*", "qs:assessment:*:*", "*:*:*:*"} {
		_, err := NewRequest(sub, key, "read")
		require.Error(t, err)
	}
	_, err = NewRequest(sub, "qs:assessment:report:*", "read")
	require.NoError(t, err)
}

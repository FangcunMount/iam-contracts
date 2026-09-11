package authorization_test

import (
	"testing"
	"time"

	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/authorization"
	"github.com/stretchr/testify/require"
)

func TestDenyReportsPolicyNotMatched(t *testing.T) {
	d := authorization.Deny(7, time.Time{})
	require.False(t, d.Allowed)
	require.Equal(t, authorization.ReasonNotMatched, d.Reason)
	require.EqualValues(t, 7, d.PolicyVersion)
}

package errors

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestWrapPreservesGRPCSemantics(t *testing.T) {
	t.Parallel()

	err := Wrap(status.Error(codes.NotFound, "user not found"))
	require.Error(t, err)
	require.True(t, IsNotFound(err))
	require.False(t, IsUnauthorized(err))
	require.Equal(t, codes.NotFound, GRPCCode(err))
	require.Equal(t, "user not found", Message(err))
	require.Equal(t, 404, ToHTTPStatus(err))

	iamErr, ok := AsIAMError(err)
	require.True(t, ok)
	require.Equal(t, "NotFound", iamErr.Code)
	require.Equal(t, codes.NotFound, iamErr.GRPCCode)
}

func TestWrapWithCodeKeepsCustomCodeAndRetryableStatus(t *testing.T) {
	t.Parallel()

	err := WrapWithCode(status.Error(codes.Unavailable, "downstream unavailable"), "UPSTREAM_DOWN", "iam upstream unavailable")
	require.Error(t, err)
	require.True(t, IsServiceUnavailable(err))
	require.True(t, IsRetryable(err))
	require.Equal(t, codes.Unavailable, GRPCCode(err))
	require.Equal(t, "iam upstream unavailable", Message(err))
	require.Equal(t, 503, ToHTTPStatus(err))

	iamErr, ok := AsIAMError(err)
	require.True(t, ok)
	require.Equal(t, "UPSTREAM_DOWN", iamErr.Code)
	require.Equal(t, "iam upstream unavailable", iamErr.Message)
}

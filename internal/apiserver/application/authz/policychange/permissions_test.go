package policychange

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPermissionCaptureIsIsolatedDeduplicatedAndBounded(t *testing.T) {
	ctx, read, err := CapturePermissions(t.Context())
	require.NoError(t, err)
	other, otherRead, err := CapturePermissions(t.Context())
	require.NoError(t, err)
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(1)
		go func() { defer wg.Done(); RecordPermission(ctx, "roles", fmt.Sprintf("action%d", i%2)) }()
	}
	wg.Wait()
	actual, err := read()
	require.NoError(t, err)
	require.Equal(t, []Permission{{Resource: "roles", Action: "action0"}, {Resource: "roles", Action: "action1"}}, actual)
	RecordPermission(other, "resources", "read")
	isolated, err := otherRead()
	require.NoError(t, err)
	require.Equal(t, []Permission{{Resource: "resources", Action: "read"}}, isolated)
	actual[0].Action = "modified"
	fresh, err := read()
	require.NoError(t, err)
	require.Equal(t, "action0", fresh[0].Action)
	for i := range 65 {
		RecordPermission(ctx, "roles", fmt.Sprintf("unique%d", i))
	}
	_, err = read()
	require.ErrorIs(t, err, ErrInvalidChangePlan, "overflow silently dropped original authorization requirements")
}

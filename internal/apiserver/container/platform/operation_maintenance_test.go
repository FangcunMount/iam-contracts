package platform

import (
	"context"
	"testing"
	"time"

	"github.com/FangcunMount/reliable-messaging/transaction"
	"github.com/stretchr/testify/require"
)

func TestOperationMaintenanceRetainsBlockedDBOwnershipAfterCancellation(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	m := &operationMaintenance{interval: time.Millisecond, budget: time.Millisecond, batch: 50, abort: func(ctx context.Context, n int) (int, error) {
		require.Equal(t, 50, n)
		close(entered)
		<-release
		return 0, ctx.Err()
	}}
	require.NoError(t, m.start())
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("maintenance did not enter")
	}
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	require.ErrorIs(t, m.close(canceled), transaction.ErrShutdownIncomplete)
	require.True(t, m.snapshot().Running, "cancellation falsely released live DB work")
	require.Error(t, m.start(), "stopped maintenance restarted")
	close(release)
	finish, done := context.WithTimeout(t.Context(), time.Second)
	defer done()
	require.NoError(t, m.close(finish))
	require.NoError(t, m.close(finish))
	require.False(t, m.snapshot().Running)
	require.EqualValues(t, 1, m.snapshot().Errors)
}

func TestOperationMaintenanceConstructorAndUnstartedCloseDoNotRunDB(t *testing.T) {
	m := &operationMaintenance{abort: func(context.Context, int) (int, error) { t.Fatal("DB call before explicit start"); return 0, nil }}
	require.NoError(t, m.close(t.Context()))
	require.Error(t, m.start())
}

package authz

import (
	"context"
	"github.com/FangcunMount/reliable-messaging/transaction"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPlanningAdmissionRetainsRealOperationsAcrossIncompleteShutdown(t *testing.T) {
	admission := newPlanAdmission(1)
	require.NoError(t, admission.enter(context.Background()))
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	require.ErrorIs(t, admission.enter(canceled), context.Canceled)
	admission.stopAdmission()
	admission.stopAdmission()
	err := admission.wait(canceled)
	var incomplete *transaction.ShutdownIncomplete
	require.ErrorAs(t, err, &incomplete)
	require.Equal(t, 1, incomplete.ActiveSends)
	require.ErrorIs(t, admission.enter(context.Background()), ErrWriteAdmissionClosed)
	admission.leave()
	require.NoError(t, admission.wait(context.Background()))
}

package authz

import (
	"context"
	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/operation"
	"github.com/FangcunMount/reliable-messaging/transaction"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
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

type heldOperationAccess struct{ entered, release chan struct{} }

func (a heldOperationAccess) Principal(context.Context) (string, bool) {
	close(a.entered)
	<-a.release
	return "", false
}
func (a heldOperationAccess) Require(context.Context, string, []operation.Permission) error {
	return operation.ErrAccessDenied
}

type unusedOperationReader struct{}

func (unusedOperationReader) Read(context.Context, string) (operation.Record, error) {
	return operation.Record{}, operation.ErrNotFound
}
func (unusedOperationReader) ReadProtocol(context.Context, operation.Candidate) (operation.ProtocolState, error) {
	return operation.ProtocolState{}, operation.ErrNotFound
}

func TestWriterShutdownAlsoWaitsForActuallyRunningOperationQueries(t *testing.T) {
	access := heldOperationAccess{entered: make(chan struct{}), release: make(chan struct{})}
	query, err := operation.NewQueryService(unusedOperationReader{}, unusedOperationReader{}, access)
	require.NoError(t, err)
	writer := &PlannedUnitOfWork{admission: newPlanAdmission(1)}
	writer.admission.queries = []*operation.QueryService{query}
	finished := make(chan error, 1)
	go func() { _, err := query.Query(t.Context(), "f46ebdfd-c2a4-43b9-aafb-b98b9675505d"); finished <- err }()
	select {
	case <-access.entered:
	case <-time.After(time.Second):
		t.Fatal("query did not start")
	}
	writer.StopAdmission()
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	require.ErrorIs(t, writer.Wait(canceled), transaction.ErrShutdownIncomplete)
	_, err = query.Query(t.Context(), "f46ebdfd-c2a4-43b9-aafb-b98b9675505d")
	require.ErrorIs(t, err, operation.ErrQueryAdmissionClosed)
	close(access.release)
	require.ErrorIs(t, <-finished, operation.ErrAccessDenied)
	require.NoError(t, writer.Wait(t.Context()))
}

package operation

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/FangcunMount/reliable-messaging/transaction"
	"github.com/stretchr/testify/require"
)

type queryReader struct {
	records []Record
	reads   int
}

func (r *queryReader) Read(context.Context, string) (Record, error) {
	index := min(r.reads, len(r.records)-1)
	r.reads++
	return r.records[index], nil
}

type protocolReader struct {
	state ProtocolState
	err   error
	calls int
}

func (r *protocolReader) ReadProtocol(context.Context, Candidate) (ProtocolState, error) {
	r.calls++
	return r.state, r.err
}

type queryAccess struct {
	principal, scope string
	denied           bool
	checks           int
	permissions      []Permission
}

func (a *queryAccess) Principal(context.Context) (string, bool) {
	return a.principal, a.principal != ""
}
func (a *queryAccess) Require(_ context.Context, scope string, permissions []Permission) error {
	a.checks++
	a.permissions = permissions
	if scope != a.scope || a.denied {
		return ErrAccessDenied
	}
	return nil
}
func queryRecord(t *testing.T, changed bool) Record {
	candidate := candidateFixture(t, "private-idempotency-key-never-expose", changed)
	version := candidate.ExpectedVersion()
	if changed {
		version++
	}
	return Record{Candidate: candidate, State: Committed, PolicyVersion: version, Result: []byte(`{"role_id":"42"}`)}
}
func TestQueryHidesOtherPrincipalsAndRechecksOriginalScopeAndRights(t *testing.T) {
	record := queryRecord(t, true)
	for _, access := range []*queryAccess{{principal: "user:8", scope: "global"}, {principal: "user:7", scope: "org:1"}, {principal: "user:7", scope: "global", denied: true}} {
		protocol := &protocolReader{}
		service, err := NewQueryService(&queryReader{records: []Record{record}}, protocol, access)
		require.NoError(t, err)
		view, err := service.Query(t.Context(), record.Candidate.ID())
		require.Error(t, err)
		require.Empty(t, view.ID)
		require.Zero(t, protocol.calls)
		if access.principal == "user:8" {
			require.ErrorIs(t, err, ErrNotFound)
			require.Zero(t, access.checks)
		} else {
			require.ErrorIs(t, err, ErrAccessDenied)
			require.Equal(t, record.Candidate.Permissions(), access.permissions)
		}
	}
}
func TestQueryPreservesCommittedBusinessWhenFinalConfirmationUnknown(t *testing.T) {
	record := queryRecord(t, true)
	protocol := &protocolReader{state: ProtocolState{Exists: true, Half: transaction.HalfPrepared, Local: transaction.LocalCommitted, Final: transaction.FinalUnknown}}
	service, err := NewQueryService(&queryReader{records: []Record{record}}, protocol, &queryAccess{principal: "user:7", scope: "global"})
	require.NoError(t, err)
	view, err := service.Query(t.Context(), record.Candidate.ID())
	require.NoError(t, err)
	require.Equal(t, Committed, view.State)
	require.Equal(t, transaction.LocalCommitted, view.Phases.Local)
	require.Equal(t, transaction.FinalUnknown, view.Phases.Final)
	require.JSONEq(t, string(record.Result), string(view.Result))
	raw, err := json.Marshal(view)
	require.NoError(t, err)
	for _, private := range []string{"private-idempotency-key-never-expose", "sdk_plan", "execution_token", "candidate_hash", "primary", "writer_epoch"} {
		require.NotContains(t, string(raw), private)
	}
	view.Result[0] = 'x'
	require.JSONEq(t, `{"role_id":"42"}`, string(record.Result))
}
func TestQueryUsesFreshJournalBoundaryForConcurrentCommit(t *testing.T) {
	committed := queryRecord(t, true)
	prepared := committed
	prepared.State = Prepared
	prepared.Result = nil
	prepared.PolicyVersion = 0
	reader := &queryReader{records: []Record{prepared, committed}}
	protocol := &protocolReader{state: ProtocolState{Exists: true, Half: transaction.HalfPrepared, Local: transaction.LocalCommitted, Final: transaction.FinalUnknown}}
	service, err := NewQueryService(reader, protocol, &queryAccess{principal: "user:7", scope: "global"})
	require.NoError(t, err)
	view, err := service.Query(t.Context(), committed.Candidate.ID())
	require.NoError(t, err)
	require.Equal(t, 2, reader.reads)
	require.Equal(t, Committed, view.State)
	require.Empty(t, view.Issue)
	require.NotEmpty(t, view.Result)
}
func TestQueryNoopNeedsNoBrokerAndProtocolFailureCannotEraseCommit(t *testing.T) {
	for _, changed := range []bool{false, true} {
		record := queryRecord(t, changed)
		protocol := &protocolReader{err: errors.New("unavailable private dependency")}
		service, err := NewQueryService(&queryReader{records: []Record{record}}, protocol, &queryAccess{principal: "user:7", scope: "global"})
		require.NoError(t, err)
		view, err := service.Query(t.Context(), record.Candidate.ID())
		require.NoError(t, err)
		require.Equal(t, transaction.LocalCommitted, view.Phases.Local)
		require.NotEmpty(t, view.Result)
		if changed {
			require.Equal(t, "protocol_status_unavailable", view.Issue)
			require.Equal(t, transaction.FinalUnknown, view.Phases.Final)
		} else {
			require.Zero(t, protocol.calls)
			require.Empty(t, view.Issue)
			require.Equal(t, transaction.HalfNotAttempted, view.Phases.Half)
			require.Equal(t, transaction.FinalNotAttempted, view.Phases.Final)
		}
	}
}

type heldAccess struct{ entered, release chan struct{} }

func (*heldAccess) Principal(context.Context) (string, bool) { return "user:7", true }
func (a *heldAccess) Require(context.Context, string, []Permission) error {
	close(a.entered)
	<-a.release
	return nil
}
func TestQueryCloseRetainsRunningCapacityAfterCancellation(t *testing.T) {
	record := queryRecord(t, false)
	access := &heldAccess{entered: make(chan struct{}), release: make(chan struct{})}
	service, err := NewQueryService(&queryReader{records: []Record{record}}, &protocolReader{}, access, QueryConfig{Concurrency: 1, Timeout: time.Second})
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := service.Query(ctx, record.Candidate.ID()); done <- err }()
	<-access.entered
	cancel()
	_, err = service.Query(t.Context(), record.Candidate.ID())
	require.ErrorIs(t, err, ErrQueryCapacity, "cancellation freed a still-running query slot")
	service.StopAdmission()
	expired, stop := context.WithCancel(t.Context())
	stop()
	var incomplete *QueryShutdownIncomplete
	require.ErrorAs(t, service.Wait(expired), &incomplete)
	require.Equal(t, 1, incomplete.Pending)
	close(access.release)
	require.ErrorIs(t, <-done, context.Canceled)
	require.NoError(t, service.Wait(t.Context()))
	_, err = service.Query(t.Context(), record.Candidate.ID())
	require.ErrorIs(t, err, ErrQueryAdmissionClosed)
}

//go:build integration

package migration

import (
	"fmt"
	"sync"
	"testing"
	"time"

	operation "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/operation"
	change "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/policychange"
	journal "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/authzoperation"
	txmysql "github.com/FangcunMount/reliable-messaging/storage/transaction/mysql"
	"github.com/FangcunMount/reliable-messaging/transaction"
	"github.com/FangcunMount/reliable-messaging/transport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// Storage-only journal proof. The later host-coordinator acceptance must also
// prove the SDK guard/witness in the same root; this fixture does not replace it.
func testM3BusinessOperationJournal(t *testing.T, host *gorm.DB, binding transaction.Binding) {
	pool, err := host.DB()
	require.NoError(t, err)
	store, err := journal.New(pool, binding)
	require.NoError(t, err)
	require.NoError(t, host.Exec("CREATE TABLE iam_m3_journal_facts(operation_id VARBINARY(36) PRIMARY KEY)").Error)
	candidate := func(t *testing.T, key, value string, changed bool) operation.Candidate {
		t.Helper()
		var version int64
		require.NoError(t, pool.QueryRowContext(t.Context(), "SELECT policy_version FROM authz_policy_versions WHERE id=1").Scan(&version))
		command, err := change.NewCommandIntent("role.update", "user:7", "global", key, map[string]string{"role_id": "42", "display_name": value})
		require.NoError(t, err)
		id := uuid.NewString()
		var plan *transaction.FrozenPlan
		if changed {
			physical, err := transport.PhysicalTopicV2("iam.authz.version.v2", "TRANSACTION", "iam.mysql", "m3-1")
			require.NoError(t, err)
			frozen, err := change.FreezeVersionPlan(command, version, id, uuid.NewString(), time.Now().UTC(), "iam.authz.version.v2", physical)
			require.NoError(t, err)
			plan = &frozen
		}
		var noop [][]byte
		if !changed {
			noop = [][]byte{[]byte(`{"changed":false}`)}
		}
		result, err := operation.NewCandidate(command, id, binding, version, []operation.Permission{{Resource: "iam:authz:collection:roles", Action: "update"}}, plan, noop...)
		require.NoError(t, err)
		return result
	}
	t.Run("one-complete-winner-no-generated-artifact-conflict", func(t *testing.T) {
		const count = 8
		candidates := make([]operation.Candidate, count)
		for i := range count {
			candidates[i] = candidate(t, "journal-concurrent", "same", true)
		}
		reservations := make([]journal.Reservation, count)
		failures := make([]error, count)
		var wg sync.WaitGroup
		for i := range count {
			wg.Add(1)
			go func() {
				defer wg.Done()
				reservations[i], failures[i] = store.Reserve(t.Context(), candidates[i], time.Minute)
			}()
		}
		wg.Wait()
		winner := ""
		claims := 0
		for i := range count {
			require.NoError(t, failures[i])
			if reservations[i].Claim != nil {
				claims++
				winner = candidates[i].ID()
			}
		}
		require.Equal(t, 1, claims)
		for _, reserved := range reservations {
			require.Equal(t, winner, reserved.Snapshot.Candidate.ID())
			require.Equal(t, journal.Prepared, reserved.Snapshot.State)
		}
		_, err := store.Reserve(t.Context(), candidate(t, "journal-concurrent", "different", true), time.Minute)
		require.ErrorIs(t, err, operation.ErrConflict)
	})
	for _, commit := range []bool{false, true} {
		t.Run(fmt.Sprintf("business-and-result-root-commit-%v", commit), func(t *testing.T) {
			c := candidate(t, fmt.Sprintf("root-%v", commit), "root", true)
			reserved, err := store.Reserve(t.Context(), c, time.Minute)
			require.NoError(t, err)
			require.NotNil(t, reserved.Claim)
			root, err := pool.BeginTx(t.Context(), nil)
			require.NoError(t, err)
			defer func() { _ = root.Rollback() }()
			guard, err := store.Bind(t.Context(), root, reserved.Claim)
			require.NoError(t, err)
			_, err = root.ExecContext(t.Context(), "INSERT INTO iam_m3_journal_facts VALUES(?)", c.ID())
			require.NoError(t, err)
			changed, err := root.ExecContext(t.Context(), "UPDATE authz_policy_versions SET policy_version=policy_version+1 WHERE id=1 AND policy_version=?", c.ExpectedVersion())
			require.NoError(t, err)
			affected, err := changed.RowsAffected()
			require.NoError(t, err)
			require.EqualValues(t, 1, affected)
			require.NoError(t, guard.Commit(t.Context(), []byte(`{"accepted":true}`), c.ExpectedVersion()+1))
			pending, err := store.Read(t.Context(), c.ID())
			require.NoError(t, err)
			require.Equal(t, journal.Prepared, pending.State, "uncommitted result escaped original root")
			if commit {
				require.NoError(t, root.Commit())
			} else {
				require.NoError(t, root.Rollback())
			}
			reused, err := pool.BeginTx(t.Context(), nil)
			require.NoError(t, err)
			_, err = store.Bind(t.Context(), reused, reserved.Claim)
			require.ErrorIs(t, err, operation.ErrStale, "a root rollback must not grant another callback execution")
			require.NoError(t, reused.Rollback())
			current, err := store.Read(t.Context(), c.ID())
			require.NoError(t, err)
			var rows int
			require.NoError(t, pool.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM iam_m3_journal_facts WHERE operation_id=?", c.ID()).Scan(&rows))
			outcome, err := store.Abort(t.Context(), reserved.Claim)
			require.NoError(t, err)
			if commit {
				require.Equal(t, journal.Committed, outcome.State)
				require.Equal(t, current.Result, outcome.Result)
				require.Equal(t, current.FinalizedAt, outcome.FinalizedAt)
			} else {
				require.Equal(t, journal.Aborted, outcome.State)
				require.Empty(t, outcome.Result)
			}
			if commit {
				require.Equal(t, journal.Committed, current.State)
				require.Equal(t, 1, rows)
				require.JSONEq(t, `{"accepted":true}`, string(current.Result))
			} else {
				require.Equal(t, journal.Prepared, current.State)
				require.Zero(t, rows)
				require.Empty(t, current.Result)
			}
		})
	}
	t.Run("expired-noop-must-win-an-abort-before-late-bind-is-rejected", func(t *testing.T) {
		c := candidate(t, "expired-noop", "noop", false)
		reserved, err := store.Reserve(t.Context(), c, time.Minute)
		require.NoError(t, err)
		_, err = pool.ExecContext(t.Context(), "UPDATE iam_authz_operations SET lease_until=UTC_TIMESTAMP(6)-INTERVAL 1 SECOND WHERE operation_id=?", c.ID())
		require.NoError(t, err)
		before, err := store.Read(t.Context(), c.ID())
		require.NoError(t, err)
		require.Equal(t, journal.Prepared, before.State, "lease expiry fabricated a rollback proof")
		count, err := store.AbortExpired(t.Context(), 100)
		require.NoError(t, err)
		require.Equal(t, 1, count)
		after, err := store.Read(t.Context(), c.ID())
		require.NoError(t, err)
		require.Equal(t, journal.Aborted, after.State)
		root, err := pool.BeginTx(t.Context(), nil)
		require.NoError(t, err)
		defer func() { _ = root.Rollback() }()
		_, err = store.Bind(t.Context(), root, reserved.Claim)
		require.ErrorIs(t, err, operation.ErrStale)
		require.NoError(t, root.Rollback())
		duplicate, err := store.Reserve(t.Context(), candidate(t, "expired-noop", "noop", false), time.Minute)
		require.NoError(t, err)
		require.Nil(t, duplicate.Claim)
		require.Equal(t, c.ID(), duplicate.Snapshot.Candidate.ID())
	})
	t.Run("live-original-root-fences-the-expiry-aborter", func(t *testing.T) {
		c := candidate(t, "live-noop", "noop", false)
		reserved, err := store.Reserve(t.Context(), c, 500*time.Millisecond)
		require.NoError(t, err)
		root, err := pool.BeginTx(t.Context(), nil)
		require.NoError(t, err)
		defer func() { _ = root.Rollback() }()
		guard, err := store.Bind(t.Context(), root, reserved.Claim)
		require.NoError(t, err)
		time.Sleep(600 * time.Millisecond)
		count, err := store.AbortExpired(t.Context(), 100)
		require.NoError(t, err)
		require.Zero(t, count, "abort crossed a live original root")
		require.NoError(t, guard.Commit(t.Context(), []byte(`{"changed":false}`), c.ExpectedVersion()))
		require.NoError(t, root.Commit())
		current, err := store.Read(t.Context(), c.ID())
		require.NoError(t, err)
		require.Equal(t, journal.Committed, current.State)
		duplicate, err := store.Reserve(t.Context(), candidate(t, "live-noop", "noop", false), time.Minute)
		require.NoError(t, err)
		require.Nil(t, duplicate.Claim)
		require.Equal(t, c.ID(), duplicate.Snapshot.Candidate.ID())
	})

	t.Run("sdk-and-business-guards-share-original-root", func(t *testing.T) {
		sdk, err := txmysql.New(pool, binding)
		require.NoError(t, err)
		for _, commit := range []bool{false, true} {
			c := candidate(t, fmt.Sprintf("joint-guards-%v", commit), "joint", true)
			reserved, err := store.Reserve(t.Context(), c, time.Minute)
			require.NoError(t, err)
			plan, err := c.Plan()
			require.NoError(t, err)
			protocol, err := sdk.Reserve(t.Context(), transaction.RequestFor(plan), plan, transaction.ReserveOptions{Lease: time.Minute})
			require.NoError(t, err)
			require.True(t, protocol.Acquired)
			initial, err := store.ReadProtocol(t.Context(), c)
			require.NoError(t, err)
			require.Equal(t, transaction.HalfNotAttempted, initial.Half)
			require.Equal(t, transaction.LocalNotStarted, initial.Local)
			require.NoError(t, sdk.BeginPrepare(t.Context(), protocol.Execution))
			interrupted, err := store.ReadProtocol(t.Context(), c)
			require.NoError(t, err)
			require.Equal(t, transaction.HalfUnknown, interrupted.Half)
			ref := protocol.Execution.References()[0]
			require.NoError(t, sdk.RecordHalf(t.Context(), protocol.Execution, ref, transaction.HalfReceipt{Outcome: transaction.HalfPrepared, BrokerMessageID: "fixture-only", BrokerTransactionID: "fixture-only"}))
			root, err := pool.BeginTx(t.Context(), nil)
			require.NoError(t, err)
			sdkGuard, err := sdk.Bind(t.Context(), root, protocol.Execution)
			require.NoError(t, err)
			businessGuard, err := store.Bind(t.Context(), root, reserved.Claim)
			require.NoError(t, err)
			_, err = root.ExecContext(t.Context(), "INSERT INTO iam_m3_journal_facts VALUES(?)", c.ID())
			require.NoError(t, err)
			_, err = root.ExecContext(t.Context(), "UPDATE authz_policy_versions SET policy_version=policy_version+1 WHERE id=1 AND policy_version=?", c.ExpectedVersion())
			require.NoError(t, err)
			require.NoError(t, businessGuard.Commit(t.Context(), []byte(`{"role_id":"42"}`), c.ExpectedVersion()+1))
			require.NoError(t, sdkGuard.CommitWitness(t.Context()))
			before, err := store.Read(t.Context(), c.ID())
			require.NoError(t, err)
			require.Equal(t, journal.Prepared, before.State)
			if commit {
				require.NoError(t, root.Commit())
			} else {
				require.NoError(t, root.Rollback())
				_, err = store.Abort(t.Context(), reserved.Claim)
				require.NoError(t, err)
				_, err = sdk.Abort(t.Context(), protocol.Execution)
				require.NoError(t, err)
			}
			var receipts int
			require.NoError(t, pool.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM iam_m3_journal_facts WHERE operation_id=?", c.ID()).Scan(&receipts))
			after, err := store.Read(t.Context(), c.ID())
			require.NoError(t, err)
			observed, err := store.ReadProtocol(t.Context(), c)
			require.NoError(t, err)
			if commit {
				require.Equal(t, 1, receipts)
				require.Equal(t, journal.Committed, after.State)
				require.Equal(t, transaction.LocalCommitted, observed.Local)
				require.Equal(t, transaction.FinalUnknown, observed.Final)
			} else {
				require.Zero(t, receipts)
				require.Equal(t, journal.Aborted, after.State)
				require.Equal(t, transaction.LocalAborted, observed.Local)
			}
		}
	})
	t.Run("wrong-store-and-corrupt-evidence-refused", func(t *testing.T) {
		c := candidate(t, "corrupt", "evidence", false)
		reserved, err := store.Reserve(t.Context(), c, time.Minute)
		require.NoError(t, err)
		wrong, err := journal.New(pool, transaction.Binding{Owner: binding.Owner, StoreID: "wrong-store", WriterEpoch: binding.WriterEpoch})
		require.NoError(t, err)
		_, err = wrong.Read(t.Context(), c.ID())
		require.ErrorIs(t, err, operation.ErrInvalid)
		_, err = pool.ExecContext(t.Context(), "UPDATE iam_authz_operations SET candidate=? WHERE operation_id=?", []byte("{}"), c.ID())
		require.NoError(t, err)
		_, err = store.Read(t.Context(), c.ID())
		require.ErrorIs(t, err, operation.ErrInvalid)
		root, err := pool.BeginTx(t.Context(), nil)
		require.NoError(t, err)
		defer func() { _ = root.Rollback() }()
		_, err = store.Bind(t.Context(), root, reserved.Claim)
		require.ErrorIs(t, err, operation.ErrInvalid)
		require.NoError(t, root.Rollback())
	})
}

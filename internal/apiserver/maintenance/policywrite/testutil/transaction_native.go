//go:build integration

package testutil

import (
	"context"
	"errors"
	journal "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/authzoperation"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	host "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/uow/authz"
	txmysql "github.com/FangcunMount/reliable-messaging/storage/transaction/mysql"
	"github.com/FangcunMount/reliable-messaging/transaction"
	"github.com/FangcunMount/reliable-messaging/transport"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// StoredHalfSender replaces Broker I/O only. All reservations, guards and
// witness/fence writes use the released SDK and the fixture's real MySQL pool.
// This is host atomicity evidence, never Broker or production evidence.
type StoredHalfSender struct {
	Store  *txmysql.Store
	Calls  int
	Reject bool
}

func (s *StoredHalfSender) Reserve(ctx context.Context, r transaction.Request, p transaction.FrozenPlan) (transaction.Reservation, error) {
	return s.Store.Reserve(ctx, r, p, transaction.ReserveOptions{Lease: time.Minute})
}
func (s *StoredHalfSender) Execute(ctx context.Context, e transaction.Execution, local transaction.LocalExecutor) (transaction.Result, error) {
	s.Calls++
	result := transaction.InitialResult(e)
	if err := s.Store.BeginPrepare(ctx, e); err != nil {
		return result, err
	}
	if s.Reject {
		_, err := s.Store.Abort(ctx, e)
		result.Local = transaction.LocalAborted
		return result, errors.Join(err, errors.New("explicit stored-half fixture rejection"))
	}
	for i, ref := range e.References() {
		if err := s.Store.RecordHalf(ctx, e, ref, transaction.HalfReceipt{Outcome: transaction.HalfPrepared, BrokerMessageID: "storage-fixture-only", BrokerTransactionID: "storage-fixture-only"}); err != nil {
			return result, err
		}
		result.Branches[i].Half = transaction.HalfPrepared
		result.Branches[i].Final = transaction.FinalUnknown
	}
	active, cancel := context.WithDeadline(ctx, e.LeaseUntil)
	defer cancel()
	outcome, err := local(active, e)
	if outcome == transaction.LocalAborted {
		_, abortErr := s.Store.Abort(ctx, e)
		err = errors.Join(err, abortErr)
	}
	evidence := make([]*transaction.Evidence, 0, len(e.References()))
	for _, ref := range e.References() {
		value, readErr := s.Store.Evidence(ctx, ref)
		if readErr != nil {
			return result, errors.Join(err, readErr)
		}
		evidence = append(evidence, value)
	}
	result.Local = transaction.DecideLocal(e.References(), evidence)
	return result, err
}

// NativeWriter explicitly installs host migration templates only in the caller's
// fresh disposable fixture, which already owns its legacy maintenance schema.
// No host journal is fabricated and no running constructor performs DDL.
func NativeWriter(t *testing.T, db *gorm.DB) (*host.PlannedUnitOfWork, *StoredHalfSender) {
	t.Helper()
	require.Equal(t, "mysql", db.Dialector.Name(), "required native maintenance dependency absent")
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	root := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(root)
		require.NotEqual(t, root, parent)
		root = parent
	}
	for _, name := range []string{"000042_transaction_messaging.up.sql", "000043_authz_message_protocol.up.sql", "000044_authz_business_operations.up.sql"} {
		raw, err := os.ReadFile(filepath.Join(root, "internal/pkg/migration/migrations", name))
		require.NoError(t, err)
		// These fixed templates contain simple CREATE/INSERT statements only. Strip
		// whole-line comments before splitting their explicit statement terminators.
		lines := []string{}
		for _, line := range strings.Split(string(raw), "\n") {
			if !strings.HasPrefix(strings.TrimSpace(line), "--") {
				lines = append(lines, line)
			}
		}
		for _, statement := range strings.Split(strings.Join(lines, "\n"), ";") {
			if strings.TrimSpace(statement) != "" {
				require.NoError(t, db.Exec(statement).Error)
			}
		}
	}
	binding := transaction.Binding{Owner: "iam.mysql", StoreID: "maintenance-native", WriterEpoch: "m3-1"}
	require.NoError(t, db.Exec("INSERT INTO rm_tx_binding VALUES (?,?,?,1)", binding.Owner, binding.StoreID, binding.WriterEpoch).Error)
	require.NoError(t, db.Exec("UPDATE iam_authz_message_protocol SET protocol_type='transaction',store_id=?,writer_epoch=?,route_revision='m3-1',generation=2 WHERE owner='iam.mysql'", binding.StoreID, binding.WriterEpoch).Error)
	local, err := host.NewTransactionUnitOfWork(db, nil, binding)
	require.NoError(t, err)
	sender := &StoredHalfSender{Store: local.Store()}
	topic := "iam.authz.version.v2"
	physical, err := transport.PhysicalTopicV2(topic, transport.TransactionMessageType, binding.Owner, "m3-1")
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	operations, err := journal.New(pool, binding)
	require.NoError(t, err)
	writer, err := host.NewPlannedUnitOfWork(local, sender, host.PlanConfig{LogicalTopic: topic, PhysicalTopic: physical, Sends: 8, ProofBudget: 2 * time.Second, Admission: func() bool { return true }, MaintenancePrincipal: "maintenance:fixture", Journal: operations, OperationLease: time.Minute})
	require.NoError(t, err)
	t.Cleanup(func() {
		writer.StopAdmission()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		require.NoError(t, writer.Wait(ctx))
	})
	return writer, sender
}
func AssertCommittedWitnesses(t *testing.T, db *gorm.DB, want int64) {
	t.Helper()
	var count int64
	require.NoError(t, db.Table("rm_tx_witness").Where("state='committed'").Count(&count).Error)
	require.Equal(t, want, count)
	require.NoError(t, db.Table("rm_tx_execution").Where("state='committed'").Count(&count).Error)
	require.Equal(t, want, count)
}

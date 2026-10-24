//go:build integration && m3_iam_runtime

package migration

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/management"
	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/operation"
	change "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/policychange"
	appuow "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/uow"
	"github.com/FangcunMount/iam/v5/internal/apiserver/container/platform"
	roledomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/role"
	"github.com/FangcunMount/iam/v5/pkg/eventcatalog"
	txmysql "github.com/FangcunMount/reliable-messaging/storage/transaction/mysql"
	"github.com/FangcunMount/reliable-messaging/transaction"
	"github.com/stretchr/testify/require"
)

// The isolated infra caller disconnects only its Broker after the genuine half,
// then permits the original IAM root to commit. It reconnects after this owner
// has really stopped. All private identities remain in its owned0700 directory.
// This is a real host writer/network interruption, not a whole API or SIGKILL.
func TestM3ActualOwnerNativeRecovery(t *testing.T) {
	require.Equal(t, "true", os.Getenv("GITHUB_ACTIONS"))
	require.Equal(t, "Linux", os.Getenv("RUNNER_OS"))
	require.Equal(t, "github-hosted", os.Getenv("RUNNER_ENVIRONMENT"))
	directory := os.Getenv("IAM_M3_NATIVE_RECOVERY_DIRECTORY")
	require.True(t, filepath.IsAbs(directory))
	info, err := os.Lstat(directory)
	require.NoError(t, err)
	require.True(t, info.IsDir())
	require.Zero(t, info.Mode().Perm()&0077)
	write := func(name string, value any) {
		t.Helper()
		raw, e := json.Marshal(value)
		require.NoError(t, e)
		f, e := os.OpenFile(filepath.Join(directory, name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		require.NoError(t, e)
		_, e = f.Write(raw)
		require.NoError(t, e)
		require.NoError(t, f.Close())
	}
	wait := func(name string) {
		t.Helper()
		deadline := time.Now().Add(210 * time.Second)
		for time.Now().Before(deadline) {
			if f, e := os.Lstat(filepath.Join(directory, name)); e == nil {
				require.True(t, f.Mode().IsRegular())
				require.Zero(t, f.Mode().Perm()&0077)
				return
			}
			select {
			case <-t.Context().Done():
				t.Fatal("owned recovery control cancelled")
			case <-time.After(50 * time.Millisecond):
			}
		}
		t.Fatal("owned recovery control timed out")
	}
	pool, db := ownedM3BrokerDatabase(t)
	opts := readM3BrokerOptions(t, "IAM_M3_BROKER_CONFIG")
	b := opts.Transactions["iam.mysql"]
	binding := transaction.Binding{Owner: "iam.mysql", StoreID: b.StoreID, WriterEpoch: b.WriterEpoch}
	_, err = pool.ExecContext(t.Context(), "INSERT INTO rm_tx_binding VALUES(?,?,?,1)", []byte(binding.Owner), []byte(binding.StoreID), []byte(binding.WriterEpoch))
	require.NoError(t, err)
	selectOwnedTransactionProtocol(t, pool, binding)
	config, err := eventcatalog.Load("../../../configs/events-m3.yaml")
	require.NoError(t, err)
	owner, err := platform.NewTransactionEventing(platform.EventingDeps{DB: db, Messaging: opts}, eventcatalog.NewCatalog(config), "")
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		require.NoError(t, owner.Stop(ctx))
	})
	require.NoError(t, owner.Start(t.Context()))
	role, err := roledomain.NewRole("m3:native_recovery_original", "Original native recovery root")
	require.NoError(t, err)
	input := map[string]string{"name": "m3:native_recovery_original"}
	request := appuow.WriteRequest{Result: operation.EmptyResultCodec(), AuthorizeKnown: appuow.ReplayPermissions(management.NewGuard(nil).RequireOperation), Operation: "native.recovery", Scope: "global", ChangedBy: "fixture", Input: input,
		Preview: func(context.Context, appuow.TxRepositories) (bool, error) { return true, nil }, Freeze: change.AllocateKeys(change.ArtifactKey("role", "m3:native_recovery_original"))}
	ctx, cancel := context.WithTimeout(management.WithAuthenticatedService(t.Context(), "admin"), 75*time.Second)
	defer cancel()
	ctx, err = change.WithRequestKey(ctx, "native-recovery-original-root")
	require.NoError(t, err)
	var calls atomic.Int32
	entered, release, joined := make(chan struct{}), make(chan struct{}), make(chan struct{})
	finished := make(chan error, 1)
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
		cancel()
		select {
		case <-joined:
		case <-time.After(15 * time.Second):
			t.Error("owned original root not joined")
		}
	})
	go func() {
		defer close(joined)
		finished <- owner.Writer.WithinPlannedTx(ctx, request, func(ctx context.Context, repos appuow.TxRepositories) error {
			calls.Add(1)
			close(entered)
			select {
			case <-release:
			case <-ctx.Done():
				return ctx.Err()
			}
			if e := repos.Roles.Create(ctx, &role); e != nil {
				return e
			}
			version, e := repos.PolicyVersions.Increment(ctx, "fixture", "original native recovery")
			if e != nil {
				return e
			}
			return change.StagePolicyVersionChanged(ctx, repos.Events, version)
		})
	}()
	select {
	case <-entered:
	case e := <-finished:
		t.Fatalf("original host root not entered: %T", e)
	case <-time.After(35 * time.Second):
		t.Fatal("original half not ready")
	}
	var database string
	require.NoError(t, pool.QueryRowContext(t.Context(), "SELECT DATABASE()").Scan(&database))
	var raw []byte
	require.NoError(t, pool.QueryRowContext(t.Context(), "SELECT reference FROM rm_tx_witness").Scan(&raw))
	ref, err := transaction.DecodeReference(raw)
	require.NoError(t, err)
	write("half.json", map[string]any{"database": database, "owner": binding.Owner, "store_id": binding.StoreID, "writer_epoch": binding.WriterEpoch, "operation_id": ref.OperationID, "branch_id": ref.BranchID, "fence": ref.Fence, "message_id": ref.MessageID})
	wait("commit") // Broker must already be disconnected by the isolated caller.
	close(release)
	select {
	case err = <-finished:
		require.NoError(t, err)
	case <-time.After(25 * time.Second):
		t.Fatal("original local commit did not finish")
	}
	<-joined
	require.EqualValues(t, 1, calls.Load())
	require.NoError(t, owner.Stop(t.Context()))
	var state, final string
	require.NoError(t, pool.QueryRowContext(t.Context(), "SELECT state,final_outcome FROM rm_tx_witness").Scan(&state, &final))
	require.Equal(t, "committed", state)
	require.NotEqual(t, "commit_confirmed", final)
	write("closed.json", map[string]any{"original_root_committed": true, "owner_stopped": true, "business_executions": calls.Load()})
	wait("record-notice")
	nativeRaw, e := os.ReadFile(filepath.Join(directory, "native-index.json"))
	require.NoError(t, e)
	expected := sha256.Sum256(nativeRaw)
	evidence, e := platform.ParseNativeLimitEvidence(nativeRaw, expected)
	require.NoError(t, e)
	require.Equal(t, 1, evidence.IndexCount())
	notices, e := platform.NewSQLNativeLimits(pool, binding, "m3-1")
	require.NoError(t, e)
	require.NoError(t, notices.RecordNativeLimit(t.Context(), evidence))
	write("notice.json", map[string]bool{"exact_original_index_recorded": true})
	wait("verify") // Only after actual native recovery, receipt and governance.
	store, err := txmysql.New(pool, binding)
	require.NoError(t, err)
	require.Equal(t, transaction.Commit, (transaction.StoreChecker{Store: store}).Check(t.Context(), ref))
	var count int
	require.NoError(t, pool.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM authz_roles WHERE name='m3:native_recovery_original'").Scan(&count))
	require.Equal(t, 1, count)
	require.NoError(t, pool.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM rm_tx_request").Scan(&count))
	require.Equal(t, 1, count)
	require.NoError(t, pool.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM rm_tx_witness WHERE final_outcome='commit_confirmed' AND final_recorded_at IS NOT NULL").Scan(&count))
	require.Equal(t, 1, count)
	require.NoError(t, pool.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM messaging_native_limit_notices WHERE status='resolved' AND resolved_at IS NOT NULL AND resolution_receipt_sha256 IS NOT NULL").Scan(&count))
	require.Equal(t, 1, count)
	opts.PublishEnabled = false
	next, err := platform.NewTransactionEventing(platform.EventingDeps{DB: db, Messaging: opts}, eventcatalog.NewCatalog(config), "")
	require.NoError(t, err)
	require.NoError(t, next.Start(t.Context()))
	require.NoError(t, next.CheckReady(t.Context()))
	require.NoError(t, next.Stop(t.Context()))
	require.NoError(t, pool.PingContext(t.Context()))
	t.Log(`M3_IAM_NATIVE_RECOVERY {"passed":5,"failed":0,"skipped":0,"original_root_once":true,"real_broker_disconnect":true,"published_sdk_rpc_verified_by_caller":true,"resolved_primary_and_paused_replacement":true,"system_acceptance":false}`)
}

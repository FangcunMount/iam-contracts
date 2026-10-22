//go:build integration && m3_iam_runtime

package migration

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/management"
	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/operation"
	change "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/policychange"
	appuow "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/uow"
	"github.com/FangcunMount/iam/v5/internal/apiserver/container/platform"
	roledomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/role"
	authzuow "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/uow/authz"
	"github.com/FangcunMount/iam/v5/pkg/eventcatalog"
	txmysql "github.com/FangcunMount/reliable-messaging/storage/transaction/mysql"
	"github.com/FangcunMount/reliable-messaging/transaction"
	"github.com/stretchr/testify/require"
)

// Actual host owner, original root and fixed SDK against owned MySQL/Broker.
// This is an in-process runtime proof, not a whole API, SIGKILL or system login.
func TestM3ActualOwnerDrain(t *testing.T) {
	pool, db := ownedM3BrokerDatabase(t)
	pool.SetMaxOpenConns(4)
	pool.SetMaxIdleConns(2)
	opts := readM3BrokerOptions(t, "IAM_M3_BROKER_CONFIG")
	binding := opts.Transactions["iam.mysql"]
	proofBinding := transaction.Binding{Owner: "iam.mysql", StoreID: binding.StoreID, WriterEpoch: binding.WriterEpoch}
	_, err := pool.ExecContext(t.Context(), "INSERT INTO rm_tx_binding VALUES(?,?,?,1)", []byte(proofBinding.Owner), []byte(proofBinding.StoreID), []byte(proofBinding.WriterEpoch))
	require.NoError(t, err)
	selectOwnedTransactionProtocol(t, pool, proofBinding)
	config, err := eventcatalog.Load("../../../configs/events-m3.yaml")
	require.NoError(t, err)
	var owners []*platform.TransactionEventing
	t.Cleanup(func() {
		for i := len(owners) - 1; i >= 0; i-- {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			e := owners[i].Stop(ctx)
			cancel()
			require.NoError(t, e)
		}
	})
	newOwner := func() *platform.TransactionEventing {
		t.Helper()
		owner, e := platform.NewTransactionEventing(platform.EventingDeps{DB: db, Messaging: opts}, eventcatalog.NewCatalog(config), "")
		require.NoError(t, e)
		owners = append(owners, owner)
		require.NoError(t, owner.Start(t.Context()))
		require.NoError(t, owner.CheckReady(t.Context()))
		return owner
	}
	owner := newOwner()
	role, err := roledomain.NewRole("m3:runtime_original", "Original runtime root")
	require.NoError(t, err)
	input := map[string]string{"name": "m3:runtime_original"}
	request := appuow.WriteRequest{Result: operation.EmptyResultCodec(), AuthorizeKnown: appuow.ReplayPermissions(management.NewGuard(nil).RequireOperation), Operation: "native.runtime", Scope: "global", ChangedBy: "fixture", Input: input,
		Preview: func(context.Context, appuow.TxRepositories) (bool, error) { return true, nil }, Freeze: change.AllocateKeys(change.ArtifactKey("role", "m3:runtime_original"))}
	businessCtx, cancelBusiness := context.WithTimeout(management.WithAuthenticatedService(t.Context(), "admin"), 60*time.Second)
	keyCtx, err := change.WithRequestKey(businessCtx, "runtime-original-root")
	require.NoError(t, err)
	entered, release, joined := make(chan struct{}), make(chan struct{}), make(chan struct{})
	finished := make(chan error, 1)
	var releaseOnce sync.Once
	var calls atomic.Int32
	t.Cleanup(func() {
		releaseOnce.Do(func() { close(release) })
		cancelBusiness()
		select {
		case <-joined:
		case <-time.After(20 * time.Second):
			t.Error("owned original runtime root did not join")
		}
	})
	go func() {
		defer close(joined)
		finished <- owner.Writer.WithinPlannedTx(keyCtx, request, func(ctx context.Context, repos appuow.TxRepositories) error {
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
			version, e := repos.PolicyVersions.Increment(ctx, "fixture", "original runtime shutdown")
			if e != nil {
				return e
			}
			return change.StagePolicyVersionChanged(ctx, repos.Events, version)
		})
	}()
	select {
	case <-entered:
	case e := <-finished:
		t.Fatalf("owned runtime root failed before its barrier: %T", e)
	case <-time.After(45 * time.Second):
		t.Fatal("owned runtime root barrier not reached")
	}
	closeCtx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	err = owner.Stop(closeCtx)
	cancel()
	var incomplete *transaction.ShutdownIncomplete
	require.True(t, errors.As(err, &incomplete), "in-flight original business must report incomplete shutdown")
	require.Equal(t, int32(1), calls.Load())
	select {
	case <-joined:
		t.Fatal("original business completed before release")
	default:
	}
	require.NoError(t, pool.PingContext(t.Context()), "incomplete shutdown closed the host pool")
	blockedCtx, err := change.WithRequestKey(businessCtx, "runtime-new-after-seal")
	require.NoError(t, err)
	require.ErrorIs(t, owner.Writer.WithinPlannedTx(blockedCtx, request, func(context.Context, appuow.TxRepositories) error {
		return errors.New("sealed writer reached business")
	}), authzuow.ErrWriteAdmissionClosed)
	releaseOnce.Do(func() { close(release) })
	select {
	case err = <-finished:
		require.NoError(t, err)
	case <-time.After(30 * time.Second):
		t.Fatal("admitted original business did not finish")
	}
	<-joined
	require.NoError(t, owner.Stop(t.Context()))
	require.NoError(t, pool.PingContext(t.Context()), "actual owner closed the borrowed primary pool")
	next := newOwner()
	store, err := txmysql.New(pool, proofBinding)
	require.NoError(t, err)
	intent, err := change.NewCommandIntent(request.Operation, "service:admin", request.Scope, "runtime-original-root", input)
	require.NoError(t, err)
	saved, err := store.Lookup(t.Context(), transaction.Request{Owner: proofBinding.Owner, Scope: change.PolicyScope, Key: intent.Key(), InputHash: intent.InputHash()})
	require.NoError(t, err)
	require.NotEmpty(t, saved.Execution.References())
	checker := transaction.StoreChecker{Store: store}
	for _, reference := range saved.Execution.References() {
		require.Equal(t, transaction.Commit, checker.Check(t.Context(), reference))
	}
	require.NoError(t, next.Writer.WithinPlannedTx(keyCtx, request, func(context.Context, appuow.TxRepositories) error {
		calls.Add(1)
		return errors.New("committed replay executed business again")
	}))
	require.Equal(t, int32(1), calls.Load())
	require.NoError(t, next.SetAdmission(false))
	require.NoError(t, next.CheckReady(t.Context()), "write pause stopped the actual idle Checker")
	require.ErrorIs(t, next.Writer.WithinPlannedTx(blockedCtx, request, func(context.Context, appuow.TxRepositories) error {
		return errors.New("paused writer reached business")
	}), authzuow.ErrWriteAdmissionClosed)
	require.NoError(t, next.Stop(t.Context()))
	require.NoError(t, pool.PingContext(t.Context()))
	var count int64
	require.NoError(t, db.Table("authz_roles").Where("name=?", "m3:runtime_original").Count(&count).Error)
	require.Equal(t, int64(1), count)
	shaCtx, shaCancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer shaCancel()
	sha, err := exec.CommandContext(shaCtx, "git", "rev-parse", "HEAD").Output()
	require.NoError(t, err)
	source := strings.TrimSpace(string(sha))
	bytes, e := hex.DecodeString(source)
	require.NoError(t, e)
	require.Len(t, bytes, 20)
	cases := map[string]bool{"actual_owner_start": true, "inflight_root_shutdown_incomplete_retains_pool": true, "late_commit_once_with_witness": true, "new_owner_fresh_exact_checker": true, "paused_owner_keeps_checker_and_seals_writes": true, "actual_stop_retains_host_pool": true}
	receipt, err := json.Marshal(map[string]any{"source_sha": source, "scope": "actual IAM owner lifecycle and original root; in-process only", "passed": len(cases), "failed": 0, "skipped": 0, "cases": cases, "business_executions": calls.Load(), "full_api_process_fault_or_system_not_claimed": true})
	require.NoError(t, err)
	t.Log("M3_IAM_RUNTIME_RECEIPT " + string(receipt))
}

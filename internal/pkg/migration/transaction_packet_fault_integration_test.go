//go:build integration && m3_iam_faults

package migration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/management"
	change "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/policychange"
	roleapp "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/role"
	appuow "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/uow"
	"github.com/FangcunMount/iam/v5/internal/apiserver/container/platform"
	policyrepo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/policy"
	"github.com/FangcunMount/iam/v5/pkg/eventcatalog"
	txmysql "github.com/FangcunMount/reliable-messaging/storage/transaction/mysql"
	"github.com/FangcunMount/reliable-messaging/transaction"
	"github.com/FangcunMount/reliable-messaging/transport"
	"github.com/FangcunMount/reliable-messaging/transport/rocketmq"
	driver "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type packetCountingUOW struct {
	appuow.UnitOfWork
	planned appuow.PlannedUnitOfWork
	calls   atomic.Int32
}

func (u *packetCountingUOW) WithinPlannedTx(ctx context.Context, request appuow.WriteRequest, fn func(context.Context, appuow.TxRepositories) error) error {
	return u.planned.WithinPlannedTx(ctx, request, func(ctx context.Context, repos appuow.TxRepositories) error {
		u.calls.Add(1)
		return fn(ctx, repos)
	})
}

// The parent infra harness owns/arms the real wire proxy and separately checks
// its backend ACK. This test never throws a substitute commit failure.
func TestM3IAMOriginalCommitPacketFault(t *testing.T) {
	boundary := os.Getenv("IAM_M3_FAULT_BOUNDARY")
	require.Contains(t, []string{"before", "after"}, boundary)
	proxy := os.Getenv("IAM_M3_FAULT_MYSQL_PROXY")
	host, _, err := net.SplitHostPort(proxy)
	require.NoError(t, err)
	require.True(t, net.ParseIP(host).IsLoopback())
	primary, primaryDB := ownedM3BrokerDatabase(t)
	var schema string
	require.NoError(t, primary.QueryRowContext(t.Context(), "SELECT DATABASE()").Scan(&schema))
	cfg, err := driver.ParseDSN(os.Getenv("IAM_M3_TEST_MYSQL_DSN"))
	require.NoError(t, err)
	cfg.Addr, cfg.DBName, cfg.ParseTime, cfg.MultiStatements, cfg.Loc = proxy, schema, true, true, time.UTC
	originalPool, err := sql.Open("mysql", cfg.FormatDSN())
	require.NoError(t, err)
	originalPool.SetMaxOpenConns(4)
	originalPool.SetMaxIdleConns(2)
	t.Cleanup(func() { require.NoError(t, originalPool.Close()) })
	originalDB, err := gorm.Open(gormmysql.New(gormmysql.Config{Conn: originalPool}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	opts := readM3BrokerOptions(t, "IAM_M3_BROKER_CONFIG")
	observer := readM3BrokerOptions(t, "IAM_M3_OBSERVER_CONFIG")
	bindingConfig := opts.Transactions["iam.mysql"]
	binding := transaction.Binding{Owner: "iam.mysql", StoreID: bindingConfig.StoreID, WriterEpoch: bindingConfig.WriterEpoch}
	_, err = primary.ExecContext(t.Context(), "INSERT INTO rm_tx_binding VALUES(?,?,?,1)", []byte(binding.Owner), []byte(binding.StoreID), []byte(binding.WriterEpoch))
	require.NoError(t, err)
	selectOwnedTransactionProtocol(t, primary, binding)
	catalogConfig, err := eventcatalog.Load("../../../configs/events-m3.yaml")
	require.NoError(t, err)
	owner, err := platform.NewTransactionEventing(platform.EventingDeps{DB: originalDB, Messaging: opts}, eventcatalog.NewCatalog(catalogConfig), "")
	require.NoError(t, err)
	stopOwner := func(o *platform.TransactionEventing) {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		require.NoError(t, o.Stop(ctx))
	}
	t.Cleanup(func() { stopOwner(owner) })
	require.NoError(t, owner.Start(t.Context()))
	config, err := observer.RocketMQ([]string{"iam.authz.version.v2"}, 1, 5*time.Second)
	require.NoError(t, err)
	consumer, err := rocketmq.NewSubscriber(rocketmq.SubscriberConfig{Config: config, MaxAttempts: 3, InvisibleDuration: 15 * time.Second, HandlerTimeout: 5 * time.Second, RetryDelay: time.Second})
	require.NoError(t, err)
	closeConsumer := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		require.NoError(t, consumer.Close(ctx))
	}
	t.Cleanup(closeConsumer)
	received := make(chan transport.Received, 4)
	require.NoError(t, consumer.Subscribe(t.Context(), "iam.authz.version.v2", "iam-m3-native-observer", func(_ context.Context, d transport.Delivery) error {
		if e := d.Ack(); e != nil {
			return e
		}
		select {
		case received <- d.Message():
			return nil
		default:
			return errors.New("bounded original-root observer full")
		}
	}, func(context.Context, transport.Failure) error { return errors.New("original-root audit refused") }))
	require.Eventually(t, func() bool { return consumer.Ready(10 * time.Second) }, 30*time.Second, 50*time.Millisecond)
	base, err := policyrepo.NewPolicyVersionRepository(primaryDB).GetCurrent(t.Context())
	require.NoError(t, err)
	require.NotNil(t, base)
	counted := &packetCountingUOW{UnitOfWork: owner.Writer, planned: owner.Writer}
	service := roleapp.NewRoleCatalog(counted, nil, management.NewGuard(nil))
	command, err := roleapp.NewCreateRoleCommand("m3:packet_"+boundary, "Actual packet boundary", "")
	require.NoError(t, err)
	command.ChangedBy = "fixture"
	key := "packet-original-" + boundary
	ctx, err := change.WithRequestKey(management.WithAuthenticatedService(t.Context(), "admin"), key)
	require.NoError(t, err)
	result, applicationError := service.CreateRole(ctx, command)
	authority, err := txmysql.New(primary, binding)
	require.NoError(t, err)
	intent, err := change.NewCommandIntent("role.create", "service:admin", "global", key, map[string]string{"name": command.NameString(), "display_name": command.DisplayName, "description": command.Description, "management_protection": string(command.ManagementProtection), "changed_by": command.ChangedBy})
	require.NoError(t, err)
	request := transaction.Request{Owner: binding.Owner, Scope: change.PolicyScope, Key: intent.Key(), InputHash: intent.InputHash()}
	reservation, err := authority.Lookup(t.Context(), request)
	require.NoError(t, err)
	require.False(t, reservation.Acquired)
	require.Len(t, reservation.Execution.References(), 1)
	reference := reservation.Execution.References()[0]
	evidence, err := authority.Evidence(t.Context(), reference)
	require.NoError(t, err)
	decision := transaction.Resolve(reference, evidence)
	var facts int64
	require.NoError(t, primaryDB.Table("authz_roles").Where("name=?", command.NameString()).Count(&facts).Error)
	current, err := policyrepo.NewPolicyVersionRepository(primaryDB).GetCurrent(t.Context())
	require.NoError(t, err)
	// Fixed counts/enums only; raw error, identity, wire and credentials stay private.
	diagnostic, err := json.Marshal(map[string]any{"boundary": boundary, "api_error": applicationError != nil, "business_rows": facts, "version_delta": current.Version - base.Version, "decision": decision, "callback_calls": counted.calls.Load()})
	require.NoError(t, err)
	fmt.Println("M3_IAM_FAULT_DIAGNOSTIC " + string(diagnostic))
	require.Equal(t, int32(1), counted.calls.Load())
	cases := map[string]bool{"actual_role_root_once": true}
	if boundary == "before" {
		require.Error(t, applicationError)
		require.Nil(t, result)
		require.Zero(t, facts)
		require.Equal(t, base.Version, current.Version)
		require.NotEqual(t, transaction.Commit, decision)
	} else {
		require.NoError(t, applicationError)
		require.NotNil(t, result)
		require.Equal(t, int64(1), facts)
		require.Equal(t, base.Version+1, current.Version)
		require.Equal(t, transaction.Commit, decision)
	}
	cases["business_version_witness_match_boundary"] = true
	duplicate, duplicateError := service.CreateRole(ctx, command)
	require.Equal(t, int32(1), counted.calls.Load(), "original request authorized a second business callback")
	require.Equal(t, int64(1), owner.Stats().HalfPreparedResults)
	observed, err := authority.Lookup(t.Context(), request)
	require.NoError(t, err)
	require.Equal(t, reservation.Execution.Plan.Digest(), observed.Execution.Plan.Digest())
	if boundary == "after" {
		require.NoError(t, duplicateError)
		require.NotNil(t, duplicate)
		require.Equal(t, result.ID, duplicate.ID)
	} else {
		require.Error(t, duplicateError)
		require.Nil(t, duplicate)
	}
	cases["duplicate_keeps_original_plan_and_result"] = true
	newChecks := int64(0)
	if boundary == "after" {
		select {
		case got := <-received:
			branch := reservation.Execution.Plan.Input().Branches[0]
			require.Equal(t, branch.Message.Input().ID, got.ID)
			require.True(t, bytes.Equal(branch.Wire, got.Wire))
		case <-time.After(45 * time.Second):
			t.Fatal("actual original committed event not received")
		}
	} else {
		stopOwner(owner)
		paused := opts
		paused.PublishEnabled = false
		next, e := platform.NewTransactionEventing(platform.EventingDeps{DB: primaryDB, Messaging: paused}, eventcatalog.NewCatalog(catalogConfig), "")
		require.NoError(t, e)
		t.Cleanup(func() { stopOwner(next) })
		require.NoError(t, next.Start(t.Context()))
		// Expiry alone is not rollback. Require persisted exact abort AND a real
		// new paused owner Broker callback; never retry the application to get it.
		require.Eventually(t, func() bool {
			read, cancel := context.WithTimeout(t.Context(), 2*time.Second)
			defer cancel()
			ev, e := authority.Evidence(read, reference)
			return e == nil && transaction.Resolve(reference, ev) == transaction.Rollback && next.Stats().CheckRollback > 0
		}, 100*time.Second, 100*time.Millisecond)
		newChecks = next.Stats().CheckRollback
		require.Zero(t, next.Stats().HalfPreparedResults)
		select {
		case <-received:
			t.Fatal("uncommitted original role event became visible")
		case <-time.After(2 * time.Second):
		}
		stopOwner(next)
	}
	cases["broker_visibility_or_new_paused_checker"] = true
	for _, table := range []string{"domain_event_outbox", "rm_outbox"} {
		var count int64
		require.NoError(t, primary.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM "+table).Scan(&count))
		require.Zero(t, count)
	}
	cases["published_outboxes_empty"] = true
	closeConsumer()
	stopOwner(owner)
	require.NoError(t, primary.PingContext(t.Context()))
	require.NoError(t, originalPool.PingContext(t.Context()))
	cases["owned_owner_and_transport_drained_primary_retained"] = true
	source, err := exec.CommandContext(t.Context(), "git", "rev-parse", "HEAD").Output()
	require.NoError(t, err)
	raw, err := json.Marshal(map[string]any{"source_sha": strings.TrimSpace(string(source)), "boundary": boundary, "scope": "actual IAM role.create original GORM root with wire-level MySQL COMMIT faults; component only", "passed": len(cases), "failed": 0, "skipped": 0, "cases": cases, "callback_calls": counted.calls.Load(), "half_prepared": owner.Stats().HalfPreparedResults, "new_paused_rollback_checks": newChecks, "full_api_process_provider_system_not_claimed": true})
	require.NoError(t, err)
	fmt.Println("M3_IAM_FAULT_RECEIPT " + string(raw))
}

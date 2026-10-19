//go:build integration

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
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/management"
	change "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/policychange"
	roleapp "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/role"
	appuow "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/uow"
	"github.com/FangcunMount/iam/v5/internal/apiserver/container/platform"
	roledomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/role"
	"github.com/FangcunMount/iam/v5/internal/apiserver/options"
	"github.com/FangcunMount/iam/v5/pkg/eventcatalog"
	txmysql "github.com/FangcunMount/reliable-messaging/storage/transaction/mysql"
	"github.com/FangcunMount/reliable-messaging/transaction"
	"github.com/FangcunMount/reliable-messaging/transport"
	"github.com/FangcunMount/reliable-messaging/transport/rocketmq"
	driver "github.com/go-sql-driver/mysql"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func readM3BrokerOptions(t *testing.T, key string) options.MessagingOptions {
	t.Helper()
	var opts options.MessagingOptions
	raw, err := os.ReadFile(os.Getenv(key))
	if err != nil || json.Unmarshal(raw, &opts) != nil {
		t.Fatal("owned Broker options are required")
	}
	host, _, err := net.SplitHostPort(opts.Endpoint)
	if err != nil || !net.ParseIP(host).IsLoopback() {
		t.Fatal("owned loopback Broker required")
	}
	return opts
}
func ownedM3BrokerDatabase(t *testing.T) (*sql.DB, *gorm.DB) {
	t.Helper()
	if !strings.HasPrefix(os.Getenv("IAM_M3_FIXTURE_PROJECT"), "rocketmq-m2-") {
		t.Fatal("required owned infra fixture identity missing")
	}
	cfg, err := driver.ParseDSN(os.Getenv("IAM_M3_TEST_MYSQL_DSN"))
	if err != nil {
		t.Fatal("owned MySQL DSN required")
	}
	host, _, err := net.SplitHostPort(cfg.Addr)
	if cfg.Net != "tcp" || err != nil || !net.ParseIP(host).IsLoopback() {
		t.Fatal("owned loopback MySQL required")
	}
	cfg.DBName = ""
	cfg.ParseTime = true
	cfg.MultiStatements = true
	admin, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal("owned MySQL unavailable")
	}
	name := fmt.Sprintf("iam_m3_broker_test_%d", time.Now().UnixNano())
	if _, err = admin.ExecContext(t.Context(), "CREATE DATABASE `"+name+"`"); err != nil {
		t.Fatal("owned schema creation failed")
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := admin.ExecContext(ctx, "DROP DATABASE `"+name+"`"); err != nil {
			t.Error("owned IAM schema cleanup failed")
		}
		if err := admin.Close(); err != nil {
			t.Error("owned admin pool cleanup failed")
		}
	})
	cfg.DBName = name
	pool, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal("owned host pool unavailable")
	}
	t.Cleanup(func() {
		if err := pool.Close(); err != nil {
			t.Error("owned host pool close failed")
		}
	})
	db, err := gorm.Open(gormmysql.New(gormmysql.Config{Conn: pool}), &gorm.Config{})
	if err != nil {
		t.Fatal("owned ORM unavailable")
	}
	migrationPool, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal("owned migration pool unavailable")
	}
	version, _, err := NewMigrator(migrationPool, &Config{Enabled: true, Database: name, FreshStages: freshStagesForTest(t, db)}).Run()
	if err != nil || version != 42 {
		t.Fatal("explicit full IAM installation failed")
	}
	// These are generated migration seed notifications, in this newly-created
	// fixture only. They are not imported production messages or a handoff proof.
	for _, table := range []string{"domain_event_outbox", "rm_outbox"} {
		if _, err = pool.ExecContext(t.Context(), "TRUNCATE TABLE "+table); err != nil {
			t.Fatal("fixture seed isolation failed")
		}
	}
	return pool, db
}

func TestM3ActualBrokerHost(t *testing.T) {
	pool, db := ownedM3BrokerDatabase(t)
	opts := readM3BrokerOptions(t, "IAM_M3_BROKER_CONFIG")
	observer := readM3BrokerOptions(t, "IAM_M3_OBSERVER_CONFIG")
	binding, ok := opts.Transactions["iam.mysql"]
	if !ok {
		t.Fatal("explicit IAM owner missing")
	}
	if _, err := pool.ExecContext(t.Context(), "INSERT INTO rm_tx_binding VALUES(?,?,?,1)", []byte("iam.mysql"), []byte(binding.StoreID), []byte(binding.WriterEpoch)); err != nil {
		t.Fatal("explicit fixture binding installation failed")
	}
	catalogConfig, err := eventcatalog.Load("../../../configs/events-m3.yaml")
	if err != nil {
		t.Fatal("explicit m3 catalog missing")
	}
	owner, err := platform.NewTransactionEventing(platform.EventingDeps{DB: db, Messaging: opts}, eventcatalog.NewCatalog(catalogConfig), "")
	if err != nil {
		t.Fatal("host transaction constructor failed")
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if owner.Stop(ctx) != nil {
			t.Error("actual host transaction owner did not drain")
		}
	})
	if err = owner.Start(t.Context()); err != nil {
		t.Fatal("actual host transaction startup failed")
	}
	topic := "iam.authz.version.v2"
	subscription := "iam-m3-native-observer"
	config, err := observer.RocketMQ([]string{topic}, 4, 5*time.Second)
	if err != nil {
		t.Fatal("owned consumer configuration failed")
	}
	consumer, err := rocketmq.NewSubscriber(rocketmq.SubscriberConfig{Config: config, MaxAttempts: 3, InvisibleDuration: 15 * time.Second, HandlerTimeout: 5 * time.Second, RetryDelay: time.Second})
	if err != nil {
		t.Fatal("actual consumer construction failed")
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if consumer.Close(ctx) != nil {
			t.Error("actual consumer did not drain")
		}
	})
	received := make(chan transport.Received, 16)
	var seenMu sync.Mutex
	seen := map[string]bool{}
	if err = consumer.Subscribe(t.Context(), topic, subscription, func(_ context.Context, d transport.Delivery) error {
		msg := d.Message()
		if err := d.Ack(); err != nil {
			return err
		}
		seenMu.Lock()
		duplicate := seen[msg.ID]
		seen[msg.ID] = true
		seenMu.Unlock()
		if duplicate {
			return nil
		}
		select {
		case received <- msg:
			return nil
		default:
			return errors.New("bounded fixture receipt queue full")
		}
	}, func(context.Context, transport.Failure) error {
		return errors.New("fixture failed-delivery audit refused")
	}); err != nil {
		t.Fatal("actual consumer subscription failed")
	}
	store, err := txmysql.New(pool, transaction.Binding{Owner: "iam.mysql", StoreID: binding.StoreID, WriterEpoch: binding.WriterEpoch})
	if err != nil {
		t.Fatal(err)
	}
	ctx := management.WithAuthenticatedService(t.Context(), "admin")
	lookup := func(operation, key string, input map[string]string) transaction.Reservation {
		command, err := change.NewCommandIntent(operation, "service:admin", "global", key, input)
		if err != nil {
			t.Fatal("fixture intent invalid")
		}
		r, err := store.Lookup(t.Context(), transaction.Request{Owner: "iam.mysql", Scope: change.PolicyScope, Key: command.Key(), InputHash: command.InputHash()})
		if err != nil {
			t.Fatal("actual SDK reservation unavailable")
		}
		return r
	}
	waitMessage := func(r transaction.Reservation) {
		select {
		case msg := <-received:
			branch := r.Execution.Plan.Input().Branches[0]
			if msg.ID != branch.Message.Input().ID || !bytes.Equal(msg.Wire, branch.Wire) {
				t.Fatal("actual Broker changed original event identity or first wire")
			}
			evidence, err := store.Evidence(t.Context(), r.Execution.References()[0])
			if err != nil || transaction.Resolve(r.Execution.References()[0], evidence) != transaction.Commit {
				t.Fatal("consumed message lacked exact committed host witness")
			}
		case <-time.After(45 * time.Second):
			t.Fatal("committed host event was not consumed")
		}
	}
	keyCtx, err := change.WithRequestKey(ctx, "broker-original-root")
	if err != nil {
		t.Fatal(err)
	}
	entered, release, finished := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	role, err := roledomain.NewRole("m3:broker_original", "Original root")
	if err != nil {
		t.Fatal(err)
	}
	request := appuow.WriteRequest{Operation: "native.commit", Scope: "global", ChangedBy: "fixture", Input: map[string]string{"name": "m3:broker_original"}, Preview: func(context.Context, appuow.TxRepositories) (bool, error) { return true, nil }, Freeze: change.AllocateKeys(change.ArtifactKey("role", "m3:broker_original"))}
	go func() {
		finished <- owner.Writer.WithinPlannedTx(keyCtx, request, func(txCtx context.Context, repos appuow.TxRepositories) error {
			close(entered)
			select {
			case <-release:
			case <-txCtx.Done():
				return txCtx.Err()
			}
			if err := repos.Roles.Create(txCtx, &role); err != nil {
				return err
			}
			version, err := repos.PolicyVersions.Increment(txCtx, "fixture", "actual original root")
			if err != nil {
				return err
			}
			return change.StagePolicyVersionChanged(txCtx, repos.Events, version)
		})
	}()
	released, joined := false, false
	defer func() {
		if !released {
			close(release)
		}
		if !joined {
			<-finished
		}
	}()
	select {
	case <-entered:
	case err := <-finished:
		joined = true
		t.Fatal("actual half preparation failed", err)
	case <-time.After(45 * time.Second):
		t.Fatal("local root did not begin after actual half")
	}
	select {
	case <-received:
		t.Error("uncommitted actual half became visible")
	case <-time.After(2 * time.Second):
	}
	close(release)
	released = true
	err = <-finished
	joined = true
	if err != nil {
		t.Fatal("actual guarded host root failed")
	}
	first := lookup("native.commit", "broker-original-root", map[string]string{"name": "m3:broker_original"})
	waitMessage(first)
	// A real application entrypoint uses that same platform-composed UOW.
	service := roleapp.NewRoleCatalog(owner.Writer, nil, management.NewGuard(nil))
	command, err := roleapp.NewCreateRoleCommand("m3:broker_application", "Actual application", "")
	if err != nil {
		t.Fatal(err)
	}
	command.ChangedBy = "fixture"
	applicationCtx, err := change.WithRequestKey(ctx, "broker-application")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.CreateRole(applicationCtx, command); err != nil {
		t.Fatal("actual application transaction failed")
	}
	application := lookup("role.create", "broker-application", map[string]string{"name": command.NameString(), "display_name": command.DisplayName, "description": command.Description, "management_protection": string(command.ManagementProtection), "changed_by": command.ChangedBy})
	waitMessage(application)

	abortCtx, err := change.WithRequestKey(ctx, "broker-rollback")
	if err != nil {
		t.Fatal(err)
	}
	failure := errors.New("explicit fixture business rejection")
	err = owner.Writer.WithinPlannedTx(abortCtx, appuow.WriteRequest{Operation: "native.abort", Scope: "global", ChangedBy: "fixture", Input: map[string]string{"name": "m3:broker_aborted"}, Preview: func(context.Context, appuow.TxRepositories) (bool, error) { return true, nil }, Freeze: change.AllocateKeys(change.ArtifactKey("role", "m3:broker_aborted"))}, func(txCtx context.Context, repos appuow.TxRepositories) error {
		candidate, e := roledomain.NewRole("m3:broker_aborted", "Aborted")
		if e != nil {
			return e
		}
		if e = repos.Roles.Create(txCtx, &candidate); e != nil {
			return e
		}
		version, e := repos.PolicyVersions.Increment(txCtx, "fixture", "actual rollback")
		if e != nil {
			return e
		}
		if e = change.StagePolicyVersionChanged(txCtx, repos.Events, version); e != nil {
			return e
		}
		return failure
	})
	if !errors.Is(err, failure) {
		t.Fatal("actual rollback lost original business outcome")
	}
	rolled := lookup("native.abort", "broker-rollback", map[string]string{"name": "m3:broker_aborted"})
	evidence, err := store.Evidence(t.Context(), rolled.Execution.References()[0])
	if err != nil || transaction.Resolve(rolled.Execution.References()[0], evidence) != transaction.Rollback {
		t.Fatal("actual rollback fence not persisted")
	}
	select {
	case <-received:
		t.Fatal("aborted host event consumed")
	case <-time.After(7 * time.Second):
	}
	var count int64
	if err = db.Table("authz_roles").Where("name=?", "m3:broker_aborted").Count(&count).Error; err != nil || count != 0 {
		t.Fatal("aborted business fact persisted")
	}
	if err = db.Table("rm_outbox").Count(&count).Error; err != nil || count != 0 {
		t.Fatal("actual host transaction used a publishing Outbox")
	}
	t.Log(`M3_HOST_RECEIPT {"passed":4,"failed":0,"skipped":0,"half_invisible":true,"commit_wire_and_witness":true,"application_entry":true,"rollback_hidden":true}`)
}

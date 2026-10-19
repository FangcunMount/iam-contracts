//go:build integration

package migration

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	change "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/policychange"
	appuow "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/uow"
	domain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/policy"
	policyrepo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/policy"
	hostuow "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/uow/authz"
	dbmysql "github.com/FangcunMount/iam/v5/internal/pkg/database/mysql"
	txmysql "github.com/FangcunMount/reliable-messaging/storage/transaction/mysql"
	"github.com/FangcunMount/reliable-messaging/transaction"
	driver "github.com/go-sql-driver/mysql"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestM3PlannedPolicyNative(t *testing.T) {
	dsn := os.Getenv("IAM_M3_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Fatal("owned disposable IAM_M3_TEST_MYSQL_DSN is required")
	}
	config, err := driver.ParseDSN(dsn)
	if err != nil {
		t.Fatal("invalid disposable DSN")
	}
	config.DBName = ""
	config.MultiStatements = true
	config.ParseTime = true
	server, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	name := fmt.Sprintf("iam_m3_policy_test_%d", time.Now().UnixNano())
	if _, err = server.ExecContext(t.Context(), "CREATE DATABASE `"+name+"`"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := server.ExecContext(cleanup, "DROP DATABASE `"+name+"`"); err != nil {
			t.Error("owned planned-policy schema cleanup failed")
		}
	}()
	config.DBName = name
	db, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	host, err := gorm.Open(gormmysql.New(gormmysql.Config{Conn: db}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	// The migration command owns/closes its driver pool. The runtime pool and
	// preparation UOW are separate host resources, as in the full-chain fixture.
	migrationPool, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	defer migrationPool.Close()
	version, _, err := NewMigrator(migrationPool, &Config{Enabled: true, Database: name, FreshStages: freshStagesForTest(t, host)}).Run()
	if err != nil || version != 42 {
		t.Fatal("full original host schema", version, err)
	}
	if err = db.PingContext(t.Context()); err != nil {
		t.Fatal("borrowed host pool unavailable after explicit migration", err)
	}
	base := policyrepo.NewPolicyVersionRepository(host)
	current, err := base.GetCurrent(t.Context())
	if err != nil || current == nil {
		t.Fatal("original policy fact missing", err)
	}
	boundary := hostuow.NewUnitOfWork(host, nil)
	reader := boundary.(appuow.ReadOnlyPlanner)
	snapshot, err := appuow.PrepareChange(t.Context(), reader, appuow.WriteRequest{Operation: "role.update", Scope: "global", Preview: func(ctx context.Context, repos appuow.TxRepositories) (bool, error) {
		if repos.Events != nil {
			return false, errors.New("read planning exposed an event stager")
		}
		tx, err := dbmysql.RequireTx(ctx)
		if err != nil {
			return false, err
		}
		raw, ok := tx.Statement.ConnPool.(*sql.Tx)
		if !ok {
			return false, errors.New("planning root missing")
		}
		_, err = raw.ExecContext(ctx, `UPDATE authz_policy_versions SET policy_version=policy_version+1 WHERE id=1`)
		var denied *driver.MySQLError
		if !errors.As(err, &denied) || denied.Number != 1792 {
			return false, errors.New("database did not enforce read-only planning")
		}
		return false, nil
	}})
	if err != nil || snapshot.Changed || snapshot.Expected != current.Version {
		t.Fatal("read-only no-op plan", snapshot, err)
	}
	err = boundary.WithinTx(t.Context(), func(active context.Context, _ appuow.TxRepositories) error {
		return reader.ReadPlan(active, func(context.Context, appuow.TxRepositories) error { return errors.New("nested planning callback ran") })
	})
	if !errors.Is(err, dbmysql.ErrActiveTransactionRequired) {
		t.Fatal("planning borrowed an existing snapshot", err)
	}
	binding := transaction.Binding{Owner: "iam.mysql", StoreID: "owned-primary", WriterEpoch: "m3-1"}
	store, err := txmysql.New(db, binding)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.ExecContext(t.Context(), `INSERT INTO rm_tx_binding VALUES(?,?,?,1)`, []byte(binding.Owner), []byte(binding.StoreID), []byte(binding.WriterEpoch)); err != nil {
		t.Fatal(err)
	}
	planned := base.(domain.PlannedRepository)
	at := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	if _, err = planned.IncrementExpected(t.Context(), current.Version, "system", "reject pool", at); !errors.Is(err, dbmysql.ErrActiveTransactionRequired) {
		t.Fatal("ordinary DB handle accepted", err)
	}
	const concurrency = 8
	executions := make([]transaction.Execution, concurrency)
	for i := range concurrency {
		id := fmt.Sprintf("cas-%d", i)
		intent, err := change.NewCommandIntent("role.update", "user:7", "global", id, map[string]string{"id": "42", "display_name": "冻结名称"})
		if err != nil {
			t.Fatal(err)
		}
		plan, err := change.FreezeVersionPlan(intent, current.Version, id, "event-"+id, at, "iam.authz.version.v2", "m3.policy.test")
		if err != nil {
			t.Fatal(err)
		}
		reserved, err := store.Reserve(t.Context(), transaction.RequestFor(plan), plan, transaction.ReserveOptions{Lease: time.Minute})
		if err != nil || !reserved.Acquired {
			t.Fatal(err)
		}
		executions[i] = reserved.Execution
	}
	type outcome struct {
		index     int
		committed bool
		err       error
	}
	results := make(chan outcome, concurrency)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i, e := range executions {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
			defer cancel()
			raw, err := db.BeginTx(ctx, nil)
			if err != nil {
				results <- outcome{i, false, err}
				return
			}
			defer raw.Rollback()
			guard, err := store.Bind(ctx, raw, e)
			if err != nil {
				results <- outcome{i, false, err}
				return
			}
			tx, err := gorm.Open(gormmysql.New(gormmysql.Config{Conn: raw, SkipInitializeWithVersion: true}), &gorm.Config{DisableAutomaticPing: true, SkipDefaultTransaction: true})
			if err != nil {
				results <- outcome{i, false, err}
				return
			}
			intent, intentErr := change.NewCommandIntent("role.update", "user:7", "global", fmt.Sprintf("cas-%d", i), map[string]string{"id": "42", "display_name": "冻结名称"})
			if intentErr != nil {
				results <- outcome{i, false, intentErr}
				return
			}
			plannedCtx, intentErr := change.WithExecutionPlan(ctx, e, intent)
			if intentErr != nil {
				results <- outcome{i, false, intentErr}
				return
			}
			repo := policyrepo.NewPolicyVersionRepository(tx)
			// More than512 UTF-8 bytes, within the original VARCHAR(512) character limit.
			_, err = repo.Increment(plannedCtx, "system", strings.Repeat("中", 200))
			if errors.Is(err, domain.ErrStaleVersion) {
				if err = raw.Rollback(); err == nil {
					var ok bool
					ok, err = store.Abort(ctx, e)
					if !ok && err == nil {
						err = errors.New("stale execution did not persist abort")
					}
				}
				results <- outcome{i, false, err}
				return
			}
			if err == nil {
				err = guard.CommitWitness(ctx)
			}
			if err == nil {
				err = raw.Commit()
			}
			results <- outcome{i, err == nil, err}
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	winners := 0
	checker := transaction.StoreChecker{Store: store}
	for result := range results {
		if result.err != nil {
			t.Fatal(result.err)
		}
		want := transaction.Rollback
		if result.committed {
			winners++
			want = transaction.Commit
		}
		if checker.Check(t.Context(), executions[result.index].References()[0]) != want {
			t.Fatal("CAS fact and witness diverged")
		}
	}
	if winners != 1 {
		t.Fatalf("%d effective executions, want1", winners)
	}
	after, err := base.GetCurrent(t.Context())
	if err != nil || after.Version != current.Version+1 {
		t.Fatal("stale plan chose another version", after, err)
	}
	// Explicit rollback keeps both the original version and all witnesses unchanged.
	id := "rollback-cas"
	intent, err := change.NewCommandIntent("role.update", "user:7", "global", id, map[string]string{"id": "42"})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := change.FreezeVersionPlan(intent, after.Version, id, "event-"+id, at, "iam.authz.version.v2", "m3.policy.test")
	if err != nil {
		t.Fatal(err)
	}
	r, err := store.Reserve(t.Context(), transaction.RequestFor(plan), plan, transaction.ReserveOptions{Lease: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := db.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Rollback()
	guard, err := store.Bind(t.Context(), raw, r.Execution)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := gorm.Open(gormmysql.New(gormmysql.Config{Conn: raw, SkipInitializeWithVersion: true}), &gorm.Config{DisableAutomaticPing: true, SkipDefaultTransaction: true})
	if err != nil {
		t.Fatal(err)
	}
	_, err = policyrepo.NewPolicyVersionRepository(tx).(domain.PlannedRepository).IncrementExpected(t.Context(), after.Version, "system", "rollback", at)
	if err != nil {
		t.Fatal(err)
	}
	if err = guard.CommitWitness(t.Context()); err != nil {
		t.Fatal(err)
	}
	if checker.Check(t.Context(), r.Execution.References()[0]) != transaction.Unknown {
		t.Fatal("uncommitted root was visible")
	}
	if err = raw.Rollback(); err != nil {
		t.Fatal(err)
	}
	restored, err := base.GetCurrent(t.Context())
	if err != nil || restored.Version != after.Version {
		t.Fatal("root rollback lost", err)
	}
	if checker.Check(t.Context(), r.Execution.References()[0]) != transaction.Unknown {
		t.Fatal("rollback fabricated witness")
	}
}

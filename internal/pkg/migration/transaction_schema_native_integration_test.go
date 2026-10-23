//go:build integration

package migration

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/FangcunMount/reliable-messaging/message"
	txmysql "github.com/FangcunMount/reliable-messaging/storage/transaction/mysql"
	"github.com/FangcunMount/reliable-messaging/transaction"
	driver "github.com/go-sql-driver/mysql"
)

func TestM3TransactionSchemaNative(t *testing.T) {
	dsn := os.Getenv("IAM_M3_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Fatal("owned disposable IAM_M3_TEST_MYSQL_DSN is required")
	}
	c, err := driver.ParseDSN(dsn)
	if err != nil {
		t.Fatal("invalid disposable DSN")
	}
	c.DBName = ""
	c.MultiStatements = true
	server, err := sql.Open("mysql", c.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	name := fmt.Sprintf("iam_m3_tx_test_%d", time.Now().UnixNano())
	if _, err = server.ExecContext(t.Context(), "CREATE DATABASE `"+name+"`"); err != nil {
		t.Fatal(err)
	}
	defer server.ExecContext(context.Background(), "DROP DATABASE `"+name+"`")
	c.DBName = name
	db, err := sql.Open("mysql", c.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, path := range []string{"000006_add_domain_event_outbox.up.sql", "000042_transaction_messaging.up.sql"} {
		raw, err := migrations.ReadFile("migrations/" + path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = db.ExecContext(t.Context(), string(raw)); err != nil {
			t.Fatal(err)
		}
	}
	_, err = db.ExecContext(t.Context(), `INSERT INTO domain_event_outbox(event_id,event_type,aggregate_type,aggregate_id,topic_name,payload_json,status,attempt_count,next_attempt_at) VALUES('old','changed','PolicyVersion','2','topic','{ "version":2 }','failed',7,UTC_TIMESTAMP(3))`)
	if err != nil {
		t.Fatal(err)
	}
	m3SQLProof(t, db, "iam.mysql")
	nativeSchema, err := migrations.ReadFile("migrations/000045_native_transaction_limit_notices.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.ExecContext(t.Context(), string(nativeSchema)); err != nil {
		t.Fatal(err)
	}
	t.Run("resolved-native-governance-read-and-reimport", func(t *testing.T) { verifySQLNativeResolutionFixture(t, db, "iam.mysql") })
	var payload, state string
	var attempts int
	if err = db.QueryRowContext(t.Context(), `SELECT payload_json,status,attempt_count FROM domain_event_outbox WHERE event_id='old'`).Scan(&payload, &state, &attempts); err != nil || payload != `{ "version":2 }` || state != "failed" || attempts != 7 {
		t.Fatal("legacy responsibility changed", err)
	}
	raw, err := migrations.ReadFile("migrations/000042_transaction_messaging.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.ExecContext(t.Context(), string(raw)); err == nil {
		t.Fatal("downgrade erased protocol responsibility")
	}
}

func m3Plan(t *testing.T, owner, id string) transaction.FrozenPlan {
	t.Helper()
	m, err := message.New(message.Input{Producer: "schema-test", ID: id, Destination: "schema.logical", EventType: "schema.fact", SchemaVersion: "v1", Scope: "org:7", ContentType: "application/octet-stream", OccurredAt: "2026-10-19T00:00:00Z", Payload: []byte{0, 255, 1}})
	if err != nil {
		t.Fatal(err)
	}
	p, err := transaction.NewPlan(transaction.PlanInput{Owner: owner, Scope: "org:7", OperationID: "op-" + id, IdempotencyKey: id, InputHash: sha256.Sum256([]byte(id)), Branches: []transaction.BranchInput{{ID: "fact", PhysicalTopic: "schema.physical", RouteRevision: "m3-1", Message: m, Wire: []byte{0, 255, 3}}}})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// This is an installation/adapter proof, not proof that a business entry uses it.
func m3SQLProof(t *testing.T, db *sql.DB, owner string) {
	t.Helper()
	ctx := t.Context()
	b := transaction.Binding{Owner: owner, StoreID: "owned-test-primary", WriterEpoch: "m3-1"}
	s, err := txmysql.New(db, b)
	if err != nil {
		t.Fatal(err)
	}
	p := m3Plan(t, owner, "binding")
	if _, err = s.Reserve(ctx, transaction.RequestFor(p), p, transaction.ReserveOptions{Lease: time.Minute}); err == nil {
		t.Fatal("migration silently authorized a binding")
	}
	if _, err = db.ExecContext(ctx, `INSERT INTO rm_tx_binding VALUES(?,?,?,1)`, []byte(b.Owner), []byte(b.StoreID), []byte(b.WriterEpoch)); err != nil {
		t.Fatal(err)
	}
	wrong, err := txmysql.New(db, transaction.Binding{Owner: owner, StoreID: "WRONG", WriterEpoch: b.WriterEpoch})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = wrong.Reserve(ctx, transaction.RequestFor(p), p, transaction.ReserveOptions{Lease: time.Minute}); !errors.Is(err, transaction.ErrBinding) {
		t.Fatal("wrong original store accepted", err)
	}
	if _, err = db.ExecContext(ctx, `CREATE TABLE m3_install_fact(id VARBINARY(128) PRIMARY KEY,value BLOB NOT NULL) ENGINE=InnoDB`); err != nil {
		t.Fatal(err)
	}
	for _, committed := range []bool{false, true} {
		id := fmt.Sprintf("root-%v", committed)
		p = m3Plan(t, owner, id)
		r, err := s.Reserve(ctx, transaction.RequestFor(p), p, transaction.ReserveOptions{Lease: time.Minute})
		if err != nil || !r.Acquired {
			t.Fatal(r, err)
		}
		raw, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer raw.Rollback()
		g, err := s.Bind(ctx, raw, r.Execution)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = raw.ExecContext(ctx, `INSERT INTO m3_install_fact VALUES(?,?)`, id, []byte{0, 255, 3}); err != nil {
			t.Fatal(err)
		}
		if err = g.CommitWitness(ctx); err != nil {
			t.Fatal(err)
		}
		check := transaction.StoreChecker{Store: s}
		ref := r.Execution.References()[0]
		if check.Check(ctx, ref) != transaction.Unknown {
			t.Fatal("uncommitted root visible")
		}
		want := transaction.Unknown
		if committed {
			err = raw.Commit()
			want = transaction.Commit
		} else {
			err = raw.Rollback()
		}
		if err != nil {
			t.Fatal(err)
		}
		if check.Check(ctx, ref) != want {
			t.Fatal("original root decision mismatch")
		}
		var count int
		if err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM m3_install_fact WHERE id=?`, id).Scan(&count); err != nil || (count == 1) != committed {
			t.Fatal(count, err)
		}
		if !committed {
			ok, err := s.Abort(ctx, r.Execution)
			if err != nil || !ok || check.Check(ctx, ref) != transaction.Rollback {
				t.Fatal("rollback evidence did not converge", err)
			}
		}
		duplicate, err := s.Reserve(ctx, transaction.RequestFor(p), p, transaction.ReserveOptions{Lease: time.Minute})
		if err != nil || duplicate.Acquired || duplicate.Execution.Token != [32]byte{} {
			t.Fatal("duplicate obtained execution permission", err)
		}
	}
	if _, err = db.ExecContext(ctx, `UPDATE rm_tx_binding SET schema_version=2 WHERE owner=?`, []byte(owner)); err != nil {
		t.Fatal(err)
	}
	p = m3Plan(t, owner, "schema-version")
	if _, err = s.Reserve(ctx, transaction.RequestFor(p), p, transaction.ReserveOptions{Lease: time.Minute}); !errors.Is(err, transaction.ErrSchema) {
		t.Fatal("incompatible schema accepted", err)
	}
}

//go:build integration

package migration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	driver "github.com/go-sql-driver/mysql"
)

func TestM3DisabledDDLSchemaHeadNative(t *testing.T) {
	dsn := os.Getenv("IAM_M3_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Fatal("owned disposable IAM_M3_TEST_MYSQL_DSN is required")
	}
	cfg, err := driver.ParseDSN(dsn)
	if err != nil {
		t.Fatal("invalid disposable DSN")
	}
	cfg.DBName = ""
	server, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	name := fmt.Sprintf("iam_m3_readonly_head_%d", time.Now().UnixNano())
	if _, err := server.ExecContext(t.Context(), "CREATE DATABASE `"+name+"`"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := server.ExecContext(ctx, "DROP DATABASE `"+name+"`"); err != nil {
			t.Error(err)
		}
	}()
	cfg.DBName = name
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	check := func(t *testing.T, want bool) {
		t.Helper()
		ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
		defer cancel()
		if err := VerifyInstalledHead(ctx, db); (err == nil) != want {
			t.Fatalf("head match=%v, err=%v", want, err)
		}
	}
	t.Run("missing-journal-remains-absent", func(t *testing.T) {
		check(t, false)
		var count int
		if err := db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=?", name).Scan(&count); err != nil || count != 0 {
			t.Fatal("read-only verification created metadata", err)
		}
	})
	if _, err := db.ExecContext(t.Context(), "CREATE TABLE schema_migrations (version BIGINT NOT NULL PRIMARY KEY,dirty BOOLEAN NOT NULL) ENGINE=InnoDB"); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, values string
		valid        bool
	}{
		{"empty", "", false}, {"clean-current", "(45,0)", true}, {"old", "(44,0)", false}, {"future", "(46,0)", false}, {"dirty", "(45,1)", false}, {"multiple", "(45,0),(44,0)", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := db.ExecContext(t.Context(), "DELETE FROM schema_migrations"); err != nil {
				t.Fatal(err)
			}
			if tc.values != "" {
				if _, err := db.ExecContext(t.Context(), "INSERT INTO schema_migrations VALUES "+tc.values); err != nil {
					t.Fatal(err)
				}
			}
			check(t, tc.valid)
			var rows, dirty int
			var versionSum sql.NullInt64
			if err := db.QueryRowContext(t.Context(), "SELECT COUNT(*),SUM(version),COALESCE(SUM(dirty),0) FROM schema_migrations").Scan(&rows, &versionSum, &dirty); err != nil {
				t.Fatal(err)
			}
			// A second check must leave both existing rows and their dirty evidence unchanged.
			check(t, tc.valid)
			var rows2, dirty2 int
			var sum2 sql.NullInt64
			if err := db.QueryRowContext(t.Context(), "SELECT COUNT(*),SUM(version),COALESCE(SUM(dirty),0) FROM schema_migrations").Scan(&rows2, &sum2, &dirty2); err != nil || rows != rows2 || dirty != dirty2 || versionSum != sum2 {
				t.Fatal("journal mutated", err)
			}
		})
	}
}

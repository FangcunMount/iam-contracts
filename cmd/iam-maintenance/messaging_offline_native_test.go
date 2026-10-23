//go:build integration

package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/FangcunMount/iam/v5/internal/apiserver/maintenance/messagingcutover"
	driver "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
)

// Required dedicated CI test: missing dependencies fail, never skip. Only the
// explicitly created random schema is mutated/dropped; no Broker/SDK is started.
func TestM3MessagingOfflineNative(t *testing.T) {
	require.Equal(t, "true", os.Getenv("GITHUB_ACTIONS"), "native fixture only in isolated GitHub CI")
	raw := os.Getenv("IAM_M3_TEST_MYSQL_DSN")
	require.NotEmpty(t, raw)
	cfg, err := driver.ParseDSN(raw)
	require.NoError(t, err)
	connector, err := driver.NewConnector(cfg)
	require.NoError(t, err)
	admin := sql.OpenDB(connector)
	t.Cleanup(func() { require.NoError(t, admin.Close()) })
	random := make([]byte, 12)
	_, err = rand.Read(random)
	require.NoError(t, err)
	name := "iam_m3_offline_" + hex.EncodeToString(random)
	_, err = admin.Exec("CREATE DATABASE `" + name + "`")
	require.NoError(t, err)
	t.Cleanup(func() { _, err := admin.Exec("DROP DATABASE `" + name + "`"); require.NoError(t, err) })
	owned := *cfg
	owned.DBName = name
	owned.MultiStatements = true
	c, err := driver.NewConnector(&owned)
	require.NoError(t, err)
	db := sql.OpenDB(c)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	ctx := context.Background()
	_, err = db.Exec(`CREATE TABLE schema_migrations(version BIGINT NOT NULL PRIMARY KEY,dirty BOOLEAN NOT NULL);
 INSERT INTO schema_migrations VALUES(41,FALSE);
 CREATE TABLE rm_outbox(id INT PRIMARY KEY,state VARBINARY(24) NOT NULL);
 CREATE TABLE domain_event_outbox(id INT PRIMARY KEY,status VARBINARY(24) NOT NULL);
 CREATE TABLE retained_authn(id INT PRIMARY KEY,value INT NOT NULL);
 INSERT INTO retained_authn VALUES(1,1);`)
	require.NoError(t, err)
	host, port, err := net.SplitHostPort(cfg.Addr)
	require.NoError(t, err)
	for key, value := range map[string]string{"HOST": host, "PORT": port, "USERNAME": cfg.User, "PASSWORD": cfg.Passwd, "DATABASE": name, "TLS_MODE": "disabled"} {
		t.Setenv("IAM_APISERVER_MYSQL_"+key, value)
	}
	directory := t.TempDir()
	require.NoError(t, os.Chmod(directory, 0700))
	preview := func(action, path string) ([]byte, string, error) {
		var out bytes.Buffer
		err := run([]string{"m3-messaging", "--action", action, "--store-id", "owned-primary", "--writer-epoch", "m3-1", "--export-plan", path}, &out)
		if err != nil {
			return nil, "", err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, "", err
		}
		h := sha256.Sum256(data)
		return data, hex.EncodeToString(h[:]), nil
	}
	apply := func(path, sha string) error {
		return run([]string{"m3-messaging", "--apply", "--writers-excluded", "--plan", path, "--reviewed-sha256", sha}, &bytes.Buffer{})
	}
	path := filepath.Join(directory, "schema.json")
	var schemaSHA string
	t.Run("preview-is-read-only-and-exclusive", func(t *testing.T) {
		_, schemaSHA, err = preview("schema", path)
		require.NoError(t, err)
		_, _, err = preview("schema", path)
		require.Error(t, err)
		var count int
		require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=?", name).Scan(&count))
		require.Equal(t, 4, count)
		info, err := os.Stat(path)
		require.NoError(t, err)
		require.EqualValues(t, 0600, info.Mode().Perm())
	})
	t.Run("dirty-metadata-refuses-without-force", func(t *testing.T) {
		_, err := db.Exec("UPDATE schema_migrations SET dirty=TRUE")
		require.NoError(t, err)
		require.Error(t, apply(path, schemaSHA))
		_, err = db.Exec("UPDATE schema_migrations SET dirty=FALSE")
		require.NoError(t, err)
	})
	t.Run("independent-digest-and-current-primary-required", func(t *testing.T) {
		require.Error(t, apply(path, "wrong"))
		raw, err := os.ReadFile(path)
		require.NoError(t, err)
		var p messagingcutover.Plan
		require.NoError(t, json.Unmarshal(raw, &p))
		p.Before.ServerUUID = "different-primary"
		raw, err = messagingcutover.Encode(p)
		require.NoError(t, err)
		other := filepath.Join(directory, "wrong-primary.json")
		require.NoError(t, os.WriteFile(other, raw, 0600))
		h := sha256.Sum256(raw)
		require.Error(t, apply(other, hex.EncodeToString(h[:])))
	})
	t.Run("additive-offline-migration-keeps-unprotected-authn-facts", func(t *testing.T) {
		_, err := db.Exec("UPDATE retained_authn SET value=2 WHERE id=1")
		require.NoError(t, err)
		require.NoError(t, apply(path, schemaSHA))
		s, err := messagingcutover.Read(ctx, db)
		require.NoError(t, err)
		require.EqualValues(t, 45, s.Version)
		require.Nil(t, s.Binding)
		require.Equal(t, "outbox", s.Selection.Protocol)
		var value int
		require.NoError(t, db.QueryRow("SELECT value FROM retained_authn WHERE id=1").Scan(&value))
		require.Equal(t, 2, value)
		require.Error(t, apply(path, schemaSHA), "original pre-migration snapshot cannot be reused")
	})
	t.Run("both-legacy-ledgers-must-be-drained", func(t *testing.T) {
		for _, q := range []string{"INSERT INTO rm_outbox VALUES(1,'pending')", "INSERT INTO domain_event_outbox VALUES(1,'pending')"} {
			_, err := db.Exec(q)
			require.NoError(t, err)
			_, _, err = preview("select", filepath.Join(directory, "must-not-exist.json"))
			require.Error(t, err)
			_, err = db.Exec("DELETE FROM rm_outbox;DELETE FROM domain_event_outbox")
			require.NoError(t, err)
		}
	})
	selectionPath := filepath.Join(directory, "selection.json")
	var selectionSHA string
	t.Run("reviewed-selection-cannot-cross-new-protocol-responsibility", func(t *testing.T) {
		_, selectionSHA, err = preview("select", selectionPath)
		require.NoError(t, err)
		_, err = db.Exec("INSERT INTO rm_tx_request VALUES(UNHEX(REPEAT('01',32)),'iam.mysql','fixture','fixture',UNHEX(REPEAT('02',32)),UNHEX(REPEAT('03',32)))")
		require.NoError(t, err)
		require.Error(t, apply(selectionPath, selectionSHA))
		_, err = db.Exec("DELETE FROM rm_tx_request")
		require.NoError(t, err)
	})
	t.Run("binding-and-selection-commit-together-no-runtime", func(t *testing.T) {
		require.NoError(t, apply(selectionPath, selectionSHA))
		s, err := messagingcutover.Read(ctx, db)
		require.NoError(t, err)
		require.Equal(t, &messagingcutover.Binding{Owner: "iam.mysql", Store: "owned-primary", Epoch: "m3-1", Version: 1}, s.Binding)
		require.Equal(t, &messagingcutover.Selection{Protocol: "transaction", Store: "owned-primary", Epoch: "m3-1", Route: "m3-1", Generation: 2, Version: 1}, s.Selection)
		require.Error(t, apply(selectionPath, selectionSHA), "initial installation cannot reset a selected protocol")
		var value int
		require.NoError(t, db.QueryRow("SELECT value FROM retained_authn WHERE id=1").Scan(&value))
		require.Equal(t, 2, value)
	})
}

package conditionretire

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	driver "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func testDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	raw := os.Getenv("ROLE_MODEL_MYSQL_DSN")
	if raw == "" {
		if os.Getenv("ROLE_MODEL_REQUIRE_MYSQL") == "true" {
			t.Fatal("ROLE_MODEL_MYSQL_DSN required")
		}
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
		require.NoError(t, err)
		pool, _ := db.DB()
		pool.SetMaxOpenConns(1)
		t.Cleanup(func() { require.NoError(t, pool.Close()) })
		return db
	}
	cfg, err := driver.ParseDSN(raw)
	require.NoError(t, err)
	require.Empty(t, cfg.DBName, "test DSN must not select an existing database")
	admin, err := sql.Open("mysql", cfg.FormatDSN())
	require.NoError(t, err)
	name := fmt.Sprintf("iam_condition_test_%d", time.Now().UnixNano())
	_, err = admin.Exec("CREATE DATABASE `" + name + "`")
	require.NoError(t, err)
	cfg.DBName = name
	cfg.ParseTime = true
	db, err := gorm.Open(mysql.Open(cfg.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	t.Cleanup(func() {
		pool, _ := db.DB()
		require.NoError(t, pool.Close())
		_, err := admin.Exec("DROP DATABASE `" + name + "`")
		if err != nil {
			t.Error(err)
		}
		require.NoError(t, admin.Close())
	})
	return db
}

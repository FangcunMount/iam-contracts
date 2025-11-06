package testutil

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// firstNonEmpty returns the first non-empty string from the args
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// OpenDBForIntegrationTest returns a *gorm.DB for integration/concurrent tests.
// It prefers MySQL when relevant env vars are set (supports multiple common names),
// otherwise falls back to an in-memory SQLite shared DB. The helper also performs
// AutoMigrate on provided models.
func OpenDBForIntegrationTest(t *testing.T, models ...interface{}) *gorm.DB {
	t.Helper()

	host := firstNonEmpty(os.Getenv("MYSQL_HOST"), os.Getenv("IAM_APISERVER_MYSQL_HOST"), os.Getenv("MYSQL_HOSTNAME"))
	if host != "" {
		port := firstNonEmpty(os.Getenv("MYSQL_PORT"), "3306")
		user := firstNonEmpty(os.Getenv("MYSQL_USER"), os.Getenv("MYSQL_USERNAME"), os.Getenv("IAM_APISERVER_MYSQL_USERNAME"), os.Getenv("IAM_APISERVER_MYSQL_USER"))
		pass := firstNonEmpty(os.Getenv("MYSQL_PASSWORD"), os.Getenv("MYSQL_PASS"), os.Getenv("IAM_APISERVER_MYSQL_PASSWORD"))
		dbName := firstNonEmpty(os.Getenv("MYSQL_DATABASE"), os.Getenv("MYSQL_DBNAME"), os.Getenv("IAM_APISERVER_MYSQL_DATABASE"), os.Getenv("IAM_APISERVER_MYSQL_DBNAME"))
		if dbName == "" {
			dbName = "infrastructure_dev"
		}

		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&loc=Local", user, pass, host, port, dbName)
		db, err := gorm.Open(gormmysql.Open(dsn), &gorm.Config{})
		require.NoError(t, err, "failed to open MySQL connection for integration test")

		if len(models) > 0 {
			require.NoError(t, db.AutoMigrate(models...), "failed to auto-migrate models on MySQL")
		}

		return db
	}

	// Fallback to sqlite in-memory shared
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err, "failed to open sqlite in-memory DB for test")

	if len(models) > 0 {
		require.NoError(t, db.AutoMigrate(models...), "failed to auto-migrate models on sqlite")
	}

	return db
}

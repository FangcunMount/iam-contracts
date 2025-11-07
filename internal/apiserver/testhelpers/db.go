package testhelpers

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// SetupTempSQLiteDB 创建一个基于临时文件的 sqlite 数据库用于并发测试，避免导入 infra 包导致循环依赖。
// 返回的 *gorm.DB 在测试结束时会被关闭，并删除临时文件。
func SetupTempSQLiteDB(t *testing.T) *gorm.DB {
	t.Helper()

	tmp, err := os.CreateTemp("", "testdb-*.db")
	require.NoError(t, err)
	tmpPath := tmp.Name()
	_ = tmp.Close()

	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_busy_timeout=10000", tmpPath)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	if sqlDB, cerr := db.DB(); cerr == nil {
		sqlDB.SetMaxOpenConns(1)
		sqlDB.SetMaxIdleConns(1)
	}

	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
		_ = os.Remove(tmpPath)
	})

	return db
}

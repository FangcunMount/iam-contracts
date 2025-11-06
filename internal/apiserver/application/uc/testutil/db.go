package testutil

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	childpo "github.com/FangcunMount/iam-contracts/internal/apiserver/infra/mysql/child"
	guardpo "github.com/FangcunMount/iam-contracts/internal/apiserver/infra/mysql/guardianship"
	userpo "github.com/FangcunMount/iam-contracts/internal/apiserver/infra/mysql/user"
)

// SetupTestDB 创建内存数据库用于测试
// 使用 SQLite 内存模式，快速且无需外部依赖
func SetupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	// 创建内存数据库
	// 使用共享内存模式，确保在连接池中多个连接可以访问相同的内存数据库
	// 参考：https://www.sqlite.org/inmemorydb.html
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent), // 测试时静默日志
		DisableForeignKeyConstraintWhenMigrating: true,                                  // SQLite 兼容性
	})
	require.NoError(t, err, "failed to create in-memory database")

	// 自动迁移所有表
	err = db.AutoMigrate(
		&userpo.UserPO{},
		&childpo.ChildPO{},
		&guardpo.GuardianshipPO{},
	)
	require.NoError(t, err, "failed to auto-migrate tables")

	// 在测试结束时清理
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

// CleanupDB 清空数据库所有表（用于每个测试之间清理）
func CleanupDB(t *testing.T, db *gorm.DB) {
	t.Helper()

	// 按依赖顺序删除（先删除有外键的表）
	tables := []string{
		"guardianships", // 监护关系（依赖 users 和 children）
		"children",      // 儿童
		"users",         // 用户
	}

	for _, table := range tables {
		err := db.Exec("DELETE FROM " + table).Error
		require.NoError(t, err, "failed to cleanup table: %s", table)
	}
}

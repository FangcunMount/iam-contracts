package apiserver

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/config"
	"github.com/fangcun-mount/iam-contracts/pkg/database"
	"github.com/fangcun-mount/iam-contracts/pkg/database/databases"
	"github.com/fangcun-mount/iam-contracts/pkg/log"
)

// DatabaseManager 数据库管理器
type DatabaseManager struct {
	config   *config.Config
	registry *database.Registry
}

// NewDatabaseManager 创建数据库管理器
func NewDatabaseManager(config *config.Config) *DatabaseManager {
	return &DatabaseManager{
		config:   config,
		registry: database.NewRegistry(),
	}
}

// Initialize 初始化数据库连接
func (dm *DatabaseManager) Initialize() error {
	log.Info("🔌 Initializing database connections...")

	// 初始化MySQL连接
	if err := dm.initMySQL(); err != nil {
		log.Warnf("Failed to initialize MySQL: %v", err)
		// 不返回错误，允许应用在没有MySQL的情况下运行
	}

	// 初始化Redis连接
	if err := dm.initRedis(); err != nil {
		log.Warnf("Failed to initialize Redis: %v", err)
		// 不返回错误，允许应用在没有Redis的情况下运行
	}

	// 初始化数据库连接
	if err := dm.registry.Init(); err != nil {
		log.Warnf("Failed to initialize database connections: %v", err)
		// 不返回错误，允许应用在没有数据库的情况下运行
	}

	log.Info("Database connections initialization completed")
	return nil
}

// initMySQL 初始化MySQL连接
func (dm *DatabaseManager) initMySQL() error {
	mysqlConfig := &databases.MySQLConfig{
		Host:                  dm.config.MySQLOptions.Host,
		Username:              dm.config.MySQLOptions.Username,
		Password:              dm.config.MySQLOptions.Password,
		Database:              dm.config.MySQLOptions.Database,
		MaxIdleConnections:    dm.config.MySQLOptions.MaxIdleConnections,
		MaxOpenConnections:    dm.config.MySQLOptions.MaxOpenConnections,
		MaxConnectionLifeTime: dm.config.MySQLOptions.MaxConnectionLifeTime,
		LogLevel:              dm.config.MySQLOptions.LogLevel,
	}

	if mysqlConfig.Host == "" {
		log.Info("MySQL host not configured, skipping MySQL initialization")
		return nil
	}

	mysqlConn := databases.NewMySQLConnection(mysqlConfig)
	return dm.registry.Register(databases.MySQL, mysqlConfig, mysqlConn)
}

// initRedis 初始化Redis连接
func (dm *DatabaseManager) initRedis() error {
	redisConfig := &databases.RedisConfig{
		Host:      dm.config.RedisOptions.Host,
		Port:      dm.config.RedisOptions.Port,
		Password:  dm.config.RedisOptions.Password,
		Database:  dm.config.RedisOptions.Database,
		MaxIdle:   dm.config.RedisOptions.MaxIdle,
		MaxActive: dm.config.RedisOptions.MaxActive,
		Timeout:   dm.config.RedisOptions.Timeout,
	}

	if redisConfig.Host == "" {
		log.Info("Redis host not configured, skipping Redis initialization")
		return nil
	}

	redisConn := databases.NewRedisConnection(redisConfig)
	return dm.registry.Register(databases.Redis, redisConfig, redisConn)
}

// GetMySQLDB 获取MySQL数据库连接
func (dm *DatabaseManager) GetMySQLDB() (*gorm.DB, error) {
	client, err := dm.registry.GetClient(databases.MySQL)
	if err != nil {
		return nil, fmt.Errorf("failed to get MySQL client: %w", err)
	}

	mysqlClient, ok := client.(*gorm.DB)
	if !ok {
		return nil, fmt.Errorf("invalid MySQL client type")
	}

	return mysqlClient, nil
}

// GetRedisClient 获取Redis客户端
func (dm *DatabaseManager) GetRedisClient() (interface{}, error) {
	return dm.registry.GetClient(databases.Redis)
}

// Close 关闭所有数据库连接
func (dm *DatabaseManager) Close() error {
	return dm.registry.Close()
}

// HealthCheck 健康检查
func (dm *DatabaseManager) HealthCheck(ctx context.Context) error {
	return dm.registry.HealthCheck(ctx)
}

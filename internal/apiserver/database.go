package apiserver

import (
	"context"
	"fmt"

	redis "github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/FangcunMount/component-base/pkg/database"
	"github.com/FangcunMount/component-base/pkg/database/connecter"
	"github.com/FangcunMount/component-base/pkg/log"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/config"
	"github.com/FangcunMount/iam-contracts/internal/pkg/options"
)

// DatabaseManager 数据库管理器
// 支持双 Redis 客户端架构（Cache + Store）
type DatabaseManager struct {
	config   *config.Config
	registry *database.Registry

	// 双 Redis 客户端
	cacheRedisClient *redis.Client // 缓存 Redis（临时数据、会话等）
	storeRedisClient *redis.Client // 存储 Redis（持久化数据、Token等）
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

	// 初始化双 Redis 客户端（Cache + Store）
	if err := dm.initRedisClients(); err != nil {
		log.Warnf("Failed to initialize Redis clients: %v", err)
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
	mysqlConfig := &connecter.MySQLConfig{
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

	mysqlConn := connecter.NewMySQLConnection(mysqlConfig)
	return dm.registry.Register(connecter.MySQL, mysqlConfig, mysqlConn)
}

// initRedisClients 初始化双 Redis 客户端（Cache + Store）
func (dm *DatabaseManager) initRedisClients() error {
	var err error

	// 初始化 Cache Redis
	dm.cacheRedisClient, err = dm.initSingleRedis("cache", dm.config.RedisOptions.Cache)
	if err != nil {
		log.Warnf("Failed to initialize Cache Redis: %v", err)
	}

	// 初始化 Store Redis
	dm.storeRedisClient, err = dm.initSingleRedis("store", dm.config.RedisOptions.Store)
	if err != nil {
		log.Warnf("Failed to initialize Store Redis: %v", err)
	}

	// 至少有一个 Redis 连接成功即可
	if dm.cacheRedisClient == nil && dm.storeRedisClient == nil {
		return fmt.Errorf("both cache and store Redis initialization failed")
	}

	return nil
}

// initSingleRedis 初始化单个 Redis 客户端
func (dm *DatabaseManager) initSingleRedis(instanceName string, opts *options.SingleRedisOptions) (*redis.Client, error) {
	if opts == nil {
		return nil, fmt.Errorf("%s redis options is nil", instanceName)
	}

	if opts.Host == "" {
		log.Infof("Redis %s host not configured, skipping initialization", instanceName)
		return nil, nil
	}

	// 构建地址
	addr := fmt.Sprintf("%s:%d", opts.Host, opts.Port)

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: opts.Password,
		DB:       opts.Database,
	})

	// 测试连接
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis %s (%s): %w", instanceName, addr, err)
	}

	log.Infof("✅ Redis %s connected successfully: %s (db: %d)", instanceName, addr, opts.Database)
	return client, nil
} // GetMySQLDB 获取MySQL数据库连接
func (dm *DatabaseManager) GetMySQLDB() (*gorm.DB, error) {
	client, err := dm.registry.GetClient(connecter.MySQL)
	if err != nil {
		return nil, fmt.Errorf("failed to get MySQL client: %w", err)
	}

	mysqlClient, ok := client.(*gorm.DB)
	if !ok {
		return nil, fmt.Errorf("invalid MySQL client type")
	}

	return mysqlClient, nil
}

// GetCacheRedisClient 获取缓存 Redis 客户端
// 用于缓存、会话、限流等临时数据
func (dm *DatabaseManager) GetCacheRedisClient() (*redis.Client, error) {
	if dm.cacheRedisClient == nil {
		return nil, fmt.Errorf("cache redis client is not initialized")
	}
	return dm.cacheRedisClient, nil
}

// GetStoreRedisClient 获取存储 Redis 客户端
// 用于持久化存储、队列、发布订阅等
func (dm *DatabaseManager) GetStoreRedisClient() (*redis.Client, error) {
	if dm.storeRedisClient == nil {
		return nil, fmt.Errorf("store redis client is not initialized")
	}
	return dm.storeRedisClient, nil
}

// Close 关闭所有数据库连接
func (dm *DatabaseManager) Close() error {
	return dm.registry.Close()
}

// HealthCheck 健康检查
func (dm *DatabaseManager) HealthCheck(ctx context.Context) error {
	return dm.registry.HealthCheck(ctx)
}

package container

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v7"
	"gorm.io/gorm"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/container/assembler"
)

// Container 容器
// 负责管理所有模块的依赖注入和生命周期
type Container struct {
	// 数据库连接
	mysqlDB     *gorm.DB
	redisClient *redis.Client

	// 业务模块
	AuthnModule *assembler.AuthnModule
	UserModule  *assembler.UserModule
	AuthzModule *assembler.AuthzModule

	// 容器状态
	initialized bool
}

// NewContainer 创建容器
func NewContainer(mysqlDB *gorm.DB, redisClient *redis.Client) *Container {
	return &Container{
		mysqlDB:     mysqlDB,
		redisClient: redisClient,
	}
}

// Initialize 初始化容器
func (c *Container) Initialize() error {
	if c.initialized {
		return fmt.Errorf("container already initialized")
	}

	// 初始化认证模块
	if err := c.initAuthModule(); err != nil {
		return fmt.Errorf("failed to initialize auth module: %w", err)
	}

	// 初始化用户模块
	if err := c.initUserModule(); err != nil {
		return fmt.Errorf("failed to initialize user module: %w", err)
	}

	// 初始化授权模块
	if err := c.initAuthzModule(); err != nil {
		return fmt.Errorf("failed to initialize authz module: %w", err)
	}

	c.initialized = true
	fmt.Printf("🏗️  Container initialized with modules: user, auth, authz\n")

	return nil
}

// initAuthModule 初始化认证模块
func (c *Container) initAuthModule() error {
	authModule := assembler.NewAuthnModule()
	if err := authModule.Initialize(c.mysqlDB, c.redisClient); err != nil {
		return fmt.Errorf("failed to initialize auth module: %w", err)
	}
	c.AuthnModule = authModule
	return nil
}

// initUserModule 初始化用户模块
func (c *Container) initUserModule() error {
	userModule := assembler.NewUserModule()
	if err := userModule.Initialize(c.mysqlDB); err != nil {
		return fmt.Errorf("failed to initialize user module: %w", err)
	}
	c.UserModule = userModule
	return nil
}

// initAuthzModule 初始化授权模块
func (c *Container) initAuthzModule() error {
	authzModule := assembler.NewAuthzModule()
	if err := authzModule.Initialize(c.mysqlDB, c.redisClient); err != nil {
		return fmt.Errorf("failed to initialize authz module: %w", err)
	}
	c.AuthzModule = authzModule
	return nil
}

// HealthCheck 健康检查
func (c *Container) HealthCheck(ctx context.Context) error {
	// 检查MySQL连接
	if c.mysqlDB != nil {
		if err := c.mysqlDB.WithContext(ctx).Raw("SELECT 1").Error; err != nil {
			return fmt.Errorf("mysql health check failed: %w", err)
		}
	}

	// 检查Redis连接
	if c.redisClient != nil {
		if err := c.redisClient.Ping().Err(); err != nil {
			return fmt.Errorf("redis health check failed: %w", err)
		}
	}

	return nil
}

// GetMySQLDB 获取MySQL数据库连接
func (c *Container) GetMySQLDB() *gorm.DB {
	return c.mysqlDB
}

// IsInitialized 检查容器是否已初始化
func (c *Container) IsInitialized() bool {
	return c.initialized
}

// PrintStatus 打印容器状态
func (c *Container) PrintStatus() {
	fmt.Printf("📊 Container Status:\n")
	fmt.Printf("   • Initialized: %t\n", c.initialized)

	// 数据库连接状态
	fmt.Printf("   • MySQL: ")
	if c.mysqlDB != nil {
		fmt.Printf("✅\n")
	} else {
		fmt.Printf("❌\n")
	}

	fmt.Printf("   • Redis: ")
	if c.redisClient != nil {
		fmt.Printf("✅\n")
	} else {
		fmt.Printf("❌\n")
	}

	// 模块状态
	fmt.Printf("   • Authn Module: ")
	if c.AuthnModule != nil {
		fmt.Printf("✅\n")
	} else {
		fmt.Printf("❌\n")
	}

	fmt.Printf("   • User Module: ")
	if c.UserModule != nil {
		fmt.Printf("✅\n")
	} else {
		fmt.Printf("❌\n")
	}

	fmt.Printf("   • Authz Module: ")
	if c.AuthzModule != nil {
		fmt.Printf("✅\n")
	} else {
		fmt.Printf("❌\n")
	}
}

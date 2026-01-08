package container

import (
	"context"
	"fmt"

	"github.com/FangcunMount/component-base/pkg/log"
	"github.com/FangcunMount/component-base/pkg/messaging"
	redis "github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/container/assembler"
	policyDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authz/policy"
	messagingInfra "github.com/FangcunMount/iam-contracts/internal/apiserver/infra/messaging"
)

// Container 容器
// 负责管理所有模块的依赖注入和生命周期
type Container struct {
	// 数据库连接
	mysqlDB          *gorm.DB
	cacheRedisClient *redis.Client // 缓存 Redis（临时数据、会话等）
	storeRedisClient *redis.Client // 存储 Redis（持久化数据、Token等）

	// 消息总线（可选）
	eventBus messaging.EventBus

	// 业务模块
	AuthnModule   *assembler.AuthnModule
	UserModule    *assembler.UserModule
	AuthzModule   *assembler.AuthzModule
	IDPModule     *assembler.IDPModule
	SuggestModule *assembler.SuggestModule

	// IDP 模块加密密钥（32 字节 AES-256）
	idpEncryptionKey []byte

	// 容器状态
	initialized bool
}

// NewContainer 创建容器
// cacheRedisClient: 缓存 Redis 客户端（用于缓存、会话、限流等临时数据）
// storeRedisClient: 存储 Redis 客户端（用于持久化存储、队列、发布订阅等）
// eventBus: 消息总线（可选，用于事件驱动，传 nil 则不使用消息队列）
// encryptionKey: IDP 模块使用的加密密钥（32 字节 AES-256），传 nil 则使用默认密钥
func NewContainer(mysqlDB *gorm.DB, cacheRedisClient, storeRedisClient *redis.Client, eventBus messaging.EventBus, encryptionKey []byte) *Container {
	// 如果未提供加密密钥，使用默认密钥（仅用于开发环境）
	if encryptionKey == nil {
		// 默认密钥：32 字节（仅供开发使用，生产环境必须提供真实密钥）
		encryptionKey = []byte("0123456789abcdef0123456789abcdef")
	}

	return &Container{
		mysqlDB:          mysqlDB,
		cacheRedisClient: cacheRedisClient,
		storeRedisClient: storeRedisClient,
		eventBus:         eventBus,
		idpEncryptionKey: encryptionKey,
	}
}

// Initialize 初始化容器
func (c *Container) Initialize() error {
	if c.initialized {
		return fmt.Errorf("container already initialized")
	}

	var errors []error

	// 1. 初始化 IDP 模块（先初始化，因为 authn 模块依赖它）
	if err := c.initIDPModule(); err != nil {
		log.Warnf("Failed to initialize IDP module: %v", err)
		errors = append(errors, fmt.Errorf("idp module: %w", err))
	}

	// 2. 初始化认证模块（依赖 IDP 模块）
	if err := c.initAuthModule(); err != nil {
		log.Warnf("Failed to initialize Authn module: %v", err)
		errors = append(errors, fmt.Errorf("authn module: %w", err))
	}

	// 3. 初始化用户模块
	if err := c.initUserModule(); err != nil {
		log.Warnf("Failed to initialize User module: %v", err)
		errors = append(errors, fmt.Errorf("user module: %w", err))
	}

	// 4. 初始化授权模块
	if err := c.initAuthzModule(); err != nil {
		log.Warnf("Failed to initialize Authz module: %v", err)
		errors = append(errors, fmt.Errorf("authz module: %w", err))
	}

	// 5. 初始化 Suggest 模块（可选）
	if err := c.initSuggestModule(); err != nil {
		log.Warnf("Failed to initialize Suggest module: %v", err)
		errors = append(errors, fmt.Errorf("suggest module: %w", err))
	}

	c.initialized = true

	// 打印初始化状态
	log.Infof("🏗️  Container initialization completed:")
	if c.IDPModule != nil {
		log.Info("   ✅ IDP module")
	} else {
		log.Warn("   ❌ IDP module failed")
	}
	if c.AuthnModule != nil {
		log.Info("   ✅ Authn module")
	} else {
		log.Warn("   ❌ Authn module failed")
	}
	if c.UserModule != nil {
		log.Info("   ✅ User module")
	} else {
		log.Warn("   ❌ User module failed")
	}
	if c.AuthzModule != nil {
		log.Info("   ✅ Authz module")
	} else {
		log.Warn("   ❌ Authz module failed")
	}
	if c.SuggestModule != nil && c.SuggestModule.Service != nil {
		log.Info("   ✅ Suggest module")
	} else {
		log.Warn("   ⚠️  Suggest module not initialized or disabled")
	}

	// 如果有错误,返回组合错误(但容器仍然标记为已初始化)
	if len(errors) > 0 {
		return fmt.Errorf("some modules failed to initialize (%d errors)", len(errors))
	}

	return nil
}

// initAuthModule 初始化认证模块（依赖 IDP 模块）
// 认证模块使用 Store Redis 进行 Token 持久化存储
func (c *Container) initAuthModule() error {
	authModule := assembler.NewAuthnModule()
	// 传递 Store Redis（用于 Token 持久化）和 IDP 模块的服务
	if err := authModule.Initialize(c.mysqlDB, c.storeRedisClient, c.IDPModule); err != nil {
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
// 授权模块使用 EventBus 发布策略版本变更通知
func (c *Container) initAuthzModule() error {
	authzModule := assembler.NewAuthzModule()

	// 创建策略版本通知器
	var versionNotifier policyDomain.VersionNotifier
	if c.eventBus != nil {
		// 使用 NSQ EventBus
		versionNotifier = messagingInfra.NewVersionNotifier(c.eventBus)
		log.Info("   📨 Policy version notifier: NSQ EventBus")
	} else {
		// 没有消息队列时，不发送通知
		log.Warn("   ⚠️  Policy version notifier: disabled (no EventBus)")
	}

	if err := authzModule.Initialize(c.mysqlDB, versionNotifier); err != nil {
		return fmt.Errorf("failed to initialize authz module: %w", err)
	}
	c.AuthzModule = authzModule
	return nil
}

// initSuggestModule 初始化联想模块
func (c *Container) initSuggestModule() error {
	suggestModule := assembler.NewSuggestModule()
	if err := suggestModule.Initialize(c.mysqlDB); err != nil {
		return fmt.Errorf("failed to initialize suggest module: %w", err)
	}
	// 可能因配置关闭而 Service 为空
	if suggestModule.Service != nil {
		c.SuggestModule = suggestModule
	}
	return nil
}

// initIDPModule 初始化 IDP 模块（Identity Provider）
// IDP 模块使用 Cache Redis 缓存 Access Token
func (c *Container) initIDPModule() error {
	idpModule := assembler.NewIDPModule()
	// 传递 Cache Redis（用于 Access Token 缓存）
	if err := idpModule.Initialize(c.mysqlDB, c.cacheRedisClient, c.idpEncryptionKey); err != nil {
		return fmt.Errorf("failed to initialize idp module: %w", err)
	}
	c.IDPModule = idpModule
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

	// 检查 Cache Redis 连接
	if c.cacheRedisClient != nil {
		if err := c.cacheRedisClient.Ping(ctx).Err(); err != nil {
			return fmt.Errorf("cache redis health check failed: %w", err)
		}
	}

	// 检查 Store Redis 连接
	if c.storeRedisClient != nil {
		if err := c.storeRedisClient.Ping(ctx).Err(); err != nil {
			return fmt.Errorf("store redis health check failed: %w", err)
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

	fmt.Printf("   • Cache Redis: ")
	if c.cacheRedisClient != nil {
		fmt.Printf("✅\n")
	} else {
		fmt.Printf("❌\n")
	}

	fmt.Printf("   • Store Redis: ")
	if c.storeRedisClient != nil {
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

	fmt.Printf("   • IDP Module: ")
	if c.IDPModule != nil {
		fmt.Printf("✅\n")
	} else {
		fmt.Printf("❌\n")
	}

	fmt.Printf("   • Suggest Module: ")
	if c.SuggestModule != nil && c.SuggestModule.Service != nil {
		fmt.Printf("✅\n")
	} else {
		fmt.Printf("⚠️  (disabled or not initialized)\n")
	}
}

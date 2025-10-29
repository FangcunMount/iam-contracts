// Package infra 基础设施层 - 提供技术实现和外部系统集成
//
// 本包负责聚合和组装所有基础设施服务，包括：
// - 数据持久化（MySQL）
// - 缓存服务（Redis）
// - 加密服务（Crypto）
// - 微信 API 集成（WechatAPI）
//
// 架构原则：
// - 基础设施层不依赖领域层，使用原始类型（primitives）作为输入输出
// - 领域服务层负责在领域对象和原始类型之间进行适配
// - 遵循六边形架构（Ports & Adapters）模式
package infra

import (
	"github.com/redis/go-redis/v9"
	"github.com/silenceper/wechat/v2/cache"
	"gorm.io/gorm"

	wechatappPort "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/domain/wechatapp/port"
	wechatsessionPort "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/domain/wechatsession/port"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/infra/crypto"
	infraMysql "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/infra/mysql"
	infraRedis "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/infra/redis"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/infra/wechatapi"
)

// InfrastructureServices 基础设施服务聚合器
//
// 本结构体聚合了所有基础设施层服务，为应用层和领域层提供技术能力。
// 注意：领域层应该通过端口（port）接口使用这些服务，而不是直接依赖具体实现。
type InfrastructureServices struct {
	// MySQL 仓储
	WechatAppRepository wechatappPort.WechatAppRepository

	// Redis 缓存
	AccessTokenCache        wechatappPort.AccessTokenCache
	WechatSessionRepository wechatsessionPort.WechatSessionRepository

	// 加密服务
	SecretVault wechatappPort.SecretVault

	// 微信 API 服务
	WechatAuthService   *wechatapi.AuthService
	WechatTokenProvider *wechatapi.TokenProvider
}

// InfrastructureDependencies 基础设施层依赖
//
// 包含创建基础设施服务所需的外部依赖。
type InfrastructureDependencies struct {
	// 数据库连接
	DB *gorm.DB

	// Redis 客户端
	RedisClient *redis.Client

	// 加密密钥（用于 AES-256-GCM 加密）
	// 必须是 32 字节（256 位）
	EncryptionKey []byte

	// 微信 SDK 缓存（可选，传 nil 则使用内存缓存）
	// 推荐使用 Redis 缓存实现，以支持分布式部署
	WechatSDKCache cache.Cache
}

// NewInfrastructureServices 创建基础设施服务实例
//
// 参数：
//   - deps: 基础设施层依赖
//
// 返回：
//   - *InfrastructureServices: 基础设施服务聚合器实例
//
// 示例：
//
//	deps := &InfrastructureDependencies{
//	    DB:            gormDB,
//	    RedisClient:   redisClient,
//	    EncryptionKey: encryptionKey,
//	    WechatSDKCache: wechatSDKCache,
//	}
//	infra := NewInfrastructureServices(deps)
func NewInfrastructureServices(deps *InfrastructureDependencies) (*InfrastructureServices, error) {
	// 创建 MySQL 仓储
	wechatAppRepo := infraMysql.NewWechatAppRepository(deps.DB)

	// 创建 Redis 缓存
	accessTokenCache := infraRedis.NewAccessTokenCache(deps.RedisClient)
	wechatSessionRepo := infraRedis.NewWechatSessionRepository(deps.RedisClient)

	// 创建加密服务
	secretVault, err := crypto.NewSecretVault(deps.EncryptionKey)
	if err != nil {
		return nil, err
	}

	// 创建微信 API 服务
	wechatAuthService := wechatapi.NewAuthService(deps.WechatSDKCache)
	wechatTokenProvider := wechatapi.NewTokenProvider(deps.WechatSDKCache)

	return &InfrastructureServices{
		// MySQL 仓储
		WechatAppRepository: wechatAppRepo,

		// Redis 缓存
		AccessTokenCache:        accessTokenCache,
		WechatSessionRepository: wechatSessionRepo,

		// 加密服务
		SecretVault: secretVault,

		// 微信 API 服务
		WechatAuthService:   wechatAuthService,
		WechatTokenProvider: wechatTokenProvider,
	}, nil
}

package assembler

import (
	"context"
	"fmt"
	"time"

	redis "github.com/redis/go-redis/v9"
	"github.com/silenceper/wechat/v2/cache"
	"gorm.io/gorm"

	"github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/component-base/pkg/log"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/application/wechatapp"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/application/wechatsession"
	wechatappDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/domain/wechatapp"
	wechatappPort "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/domain/wechatapp/port"
	wechatappService "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/domain/wechatapp/service"
	wechatsessionPort "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/domain/wechatsession/port"
	wechatsessionService "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/domain/wechatsession/service"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/infra/crypto"
	infraMysql "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/infra/mysql"
	infraRedis "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/infra/redis"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/infra/wechatapi"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/interface/restful/handler"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
)

// IDPModule IDP 模块（Identity Provider）
// 负责组装 IDP 相关的所有组件
//
// 架构说明：
// - 直接在容器侧管理基础设施组件，无需中间聚合器
// - 遵循六边形架构：Infrastructure -> Domain -> Application -> Interface
//
// 职责：
// - 微信应用管理（HTTP 接口）
// - 提供基础设施服务（供 authn 模块使用）
// - 认证功能由 authn 模块统一提供
type IDPModule struct {
	// 应用服务（对外暴露）
	WechatAppService           wechatapp.WechatAppApplicationService
	WechatAppCredentialService wechatapp.WechatAppCredentialApplicationService
	WechatAppTokenService      wechatapp.WechatAppTokenApplicationService
	WechatAuthService          wechatsession.WechatAuthApplicationService

	// HTTP 处理器（对外暴露）
	WechatAppHandler *handler.WechatAppHandler
	// WechatAuthHandler 已移除 - 认证由 authn 模块统一提供

	// 基础设施组件（内部管理，供其他模块使用）
	wechatAppRepo       wechatappPort.WechatAppRepository
	accessTokenCache    wechatappPort.AccessTokenCache
	wechatSessionRepo   wechatsessionPort.WechatSessionRepository
	secretVault         wechatappPort.SecretVault
	wechatAuthProvider  *wechatapi.AuthProvider
	wechatTokenProvider *wechatapi.TokenProvider
}

// NewIDPModule 创建 IDP 模块
func NewIDPModule() *IDPModule {
	return &IDPModule{}
}

// Initialize 初始化模块
// params[0]: *gorm.DB - 数据库连接
// params[1]: *redis.Client - Redis 客户端
// params[2]: []byte - 加密密钥（32 字节 AES-256）
func (m *IDPModule) Initialize(params ...interface{}) error {
	// 验证参数
	db, redisClient, encryptionKey, err := m.validateParameters(params)
	if err != nil {
		return err
	}

	// 初始化基础设施层组件（直接创建）
	if err := m.initializeInfrastructure(db, redisClient, encryptionKey); err != nil {
		return err
	}

	// 初始化领域层
	domainServices, err := m.initializeDomain()
	if err != nil {
		return err
	}

	// 初始化应用层
	if err := m.initializeApplication(domainServices); err != nil {
		return err
	}

	// 初始化接口层
	if err := m.initializeInterface(); err != nil {
		return err
	}

	return nil
}

// validateParameters 验证初始化参数
func (m *IDPModule) validateParameters(params []interface{}) (*gorm.DB, *redis.Client, []byte, error) {
	if len(params) < 3 {
		log.Warnf("IDP module initialization requires 3 parameters, got %d", len(params))
		return nil, nil, nil, errors.WithCode(code.ErrModuleInitializationFailed,
			"missing required parameters: db, redis client, and encryption key")
	}

	db, ok := params[0].(*gorm.DB)
	if !ok || db == nil {
		log.Warnf("IDP module initialization requires a valid database connection")
		return nil, nil, nil, errors.WithCode(code.ErrModuleInitializationFailed,
			"database connection is nil or invalid")
	}

	redisClient, ok := params[1].(*redis.Client)
	if !ok || redisClient == nil {
		log.Warnf("IDP module initialization requires a valid Redis client")
		return nil, nil, nil, errors.WithCode(code.ErrModuleInitializationFailed,
			"redis client is nil or invalid")
	}

	encryptionKey, ok := params[2].([]byte)
	if !ok || len(encryptionKey) != 32 {
		log.Warnf("IDP module initialization requires a 32-byte encryption key")
		return nil, nil, nil, errors.WithCode(code.ErrModuleInitializationFailed,
			"encryption key must be 32 bytes for AES-256")
	}

	return db, redisClient, encryptionKey, nil
}

// initializeInfrastructure 初始化基础设施层组件
// 直接创建各个基础设施组件，无需中间聚合器（InfrastructureServices）
func (m *IDPModule) initializeInfrastructure(
	db *gorm.DB,
	redisClient *redis.Client,
	encryptionKey []byte,
) error {
	// 创建 MySQL 仓储
	m.wechatAppRepo = infraMysql.NewWechatAppRepository(db)

	// 创建 Redis 缓存
	m.accessTokenCache = infraRedis.NewAccessTokenCache(redisClient)
	m.wechatSessionRepo = infraRedis.NewWechatSessionRepository(redisClient)

	// 创建加密服务
	secretVault, err := crypto.NewSecretVault(encryptionKey)
	if err != nil {
		return fmt.Errorf("failed to create secret vault: %w", err)
	}
	m.secretVault = secretVault

	// 创建微信 API 服务（传 nil 使用内存缓存）
	var wechatSDKCache cache.Cache = nil
	m.wechatAuthProvider = wechatapi.NewAuthProvider(wechatSDKCache)
	m.wechatTokenProvider = wechatapi.NewTokenProvider(wechatSDKCache)

	return nil
}

// domainServices 领域层服务（内部结构）
type domainServices struct {
	// 微信应用领域服务
	wechatAppCreator  wechatappPort.WechatAppCreator
	wechatAppQuerier  wechatappPort.WechatAppQuerier
	credentialRotater wechatappPort.CredentialRotater
	accessTokenCacher wechatappPort.AccessTokenCacher
	appTokenProvider  wechatappPort.AppTokenProvider
}

// initializeDomain 初始化领域层
func (m *IDPModule) initializeDomain() (*domainServices, error) {
	// 创建微信应用领域服务
	wechatAppQuerier := wechatappService.NewWechatAppQuerier(
		m.wechatAppRepo,
	)

	wechatAppCreator := wechatappService.NewWechatAppCreator(
		wechatAppQuerier,
	)

	credentialRotater := wechatappService.NewCredentialRotater(
		m.secretVault,
		time.Now,
	)

	accessTokenCacher := wechatappService.NewAccessTokenCacher()

	// 创建应用令牌提供器适配器（连接基础设施层和领域层）
	appTokenProvider := &appTokenProviderAdapter{
		tokenProvider: m.wechatTokenProvider,
		wechatAppRepo: m.wechatAppRepo,
	}

	return &domainServices{
		wechatAppCreator:  wechatAppCreator,
		wechatAppQuerier:  wechatAppQuerier,
		credentialRotater: credentialRotater,
		accessTokenCacher: accessTokenCacher,
		appTokenProvider:  appTokenProvider,
	}, nil
}

// initializeApplication 初始化应用层
func (m *IDPModule) initializeApplication(
	domainServices *domainServices,
) error {
	// 创建微信认证器
	wechatAuthenticator := wechatsessionService.NewAuthenticator(
		m.wechatAuthProvider,
		domainServices.wechatAppQuerier,
		m.secretVault, // 添加 secretVault 参数
	)

	// 直接创建各个应用服务
	m.WechatAppService = wechatapp.NewWechatAppApplicationService(
		m.wechatAppRepo,
		domainServices.wechatAppCreator,
		domainServices.wechatAppQuerier,
		domainServices.credentialRotater,
	)

	m.WechatAppCredentialService = wechatapp.NewWechatAppCredentialApplicationService(
		m.wechatAppRepo,
		domainServices.wechatAppQuerier,
		domainServices.credentialRotater,
	)

	m.WechatAppTokenService = wechatapp.NewWechatAppTokenApplicationService(
		domainServices.wechatAppQuerier,
		domainServices.accessTokenCacher,
		domainServices.appTokenProvider,
		m.accessTokenCache,
	)

	m.WechatAuthService = wechatsession.NewWechatAuthApplicationService(
		wechatAuthenticator,
	)

	return nil
}

// initializeInterface 初始化接口层
func (m *IDPModule) initializeInterface() error {
	// 创建 HTTP 处理器（仅微信应用管理）
	m.WechatAppHandler = handler.NewWechatAppHandler(
		m.WechatAppService,
		m.WechatAppCredentialService,
		m.WechatAppTokenService,
	)

	// WechatAuthHandler 已移除 - 认证功能由 authn 模块统一提供
	// authn 模块通过容器依赖注入使用 IDP 模块的基础设施服务

	return nil
}

// ==================== 适配器 ====================

// appTokenProviderAdapter 应用令牌提供器适配器
// 将基础设施层的 TokenProvider 适配为领域层的 AppTokenProvider 接口
type appTokenProviderAdapter struct {
	tokenProvider *wechatapi.TokenProvider
	wechatAppRepo wechatappPort.WechatAppRepository
}

// Fetch 实现 AppTokenProvider 接口
func (a *appTokenProviderAdapter) Fetch(
	ctx context.Context,
	app *wechatappDomain.WechatApp,
) (*wechatappDomain.AppAccessToken, error) {
	// 获取凭据
	if app.Cred == nil || app.Cred.Auth == nil {
		return nil, fmt.Errorf("app credentials not found")
	}

	// 注意：这里需要解密密钥，但适配器不应该直接访问 SecretVault
	// 实际上，AppTokenProvider 应该由应用层调用，而不是在这里直接实现
	// 这个适配器的实现需要重新考虑
	//
	// 正确的做法是：
	// 1. 应用层获取 app 时已经解密了密钥
	// 2. 或者 AppTokenProvider 接口改为接受明文 appID 和 appSecret
	//
	// 这里暂时返回错误，表示需要调整架构
	return nil, fmt.Errorf("not implemented: AppTokenProvider should be called from application layer with decrypted credentials")
}

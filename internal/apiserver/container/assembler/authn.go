package assembler

import (
	"context"

	redis "github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"gorm.io/gorm"

	"github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/component-base/pkg/log"
	accountApp "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/application/account"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/application/adapter"
	jwksApp "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/application/jwks"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/application/login"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/application/token"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/application/uow"
	acctDriven "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account/port/driven"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/jwks"
	jwksDriven "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/jwks/port/driven"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/infra/crypto"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/infra/jwt"
	mysqlacct "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/infra/mysql/account"
	jwksMysql "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/infra/mysql/jwks"
	redistoken "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/infra/redis/token"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/infra/scheduler"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/infra/wechat"
	authhandler "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/interface/restful/handler"
	mysqluser "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/infra/mysql/user"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"

	authService "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/authentication/service/authenticator"
	tokenService "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/authentication/service/token"
	jwksService "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/jwks/service"
)

// AuthnModule 认证模块
// 负责组装认证相关的所有组件
type AuthnModule struct {
	// 账户应用服务
	AccountService          accountApp.AccountApplicationService
	OperationAccountService accountApp.OperationAccountApplicationService
	WeChatAccountService    accountApp.WeChatAccountApplicationService
	LookupService           accountApp.AccountLookupApplicationService

	// 认证服务
	LoginService *login.LoginService
	TokenService *token.TokenService

	// JWKS 应用服务
	KeyManagementApp *jwksApp.KeyManagementAppService
	KeyPublishApp    *jwksApp.KeyPublishAppService
	KeyRotationApp   *jwksApp.KeyRotationAppService

	// HTTP 处理器
	AccountHandler *authhandler.AccountHandler
	AuthHandler    *authhandler.AuthHandler
	JWKSHandler    *authhandler.JWKSHandler

	// 调度器
	RotationScheduler interface {
		Start(ctx context.Context) error
		Stop() error
		IsRunning() bool
		TriggerNow(ctx context.Context) error
	}
}

// NewAuthnModule 创建认证模块
func NewAuthnModule() *AuthnModule {
	return &AuthnModule{}
}

// Initialize 初始化模块
// params[0]: *gorm.DB - 数据库连接
// params[1]: *redis.Client - Redis 客户端
// params[2]: *IDPModule - IDP 模块（可选，用于微信认证）
func (m *AuthnModule) Initialize(params ...interface{}) error {
	// 验证参数
	db, redisClient, idpModule, err := m.validateParameters(params)
	if err != nil {
		return err
	}

	// 初始化基础设施层
	infra, err := m.initializeInfrastructure(db, redisClient, idpModule)
	if err != nil {
		return err
	}

	// 初始化领域层
	domain, err := m.initializeDomain(infra)
	if err != nil {
		return err
	}

	// 初始化应用层
	if err := m.initializeApplication(infra, domain); err != nil {
		return err
	}

	// 初始化接口层
	if err := m.initializeInterface(); err != nil {
		return err
	}

	// 初始化调度器（可选）
	m.initializeSchedulers()

	return nil
}

// validateParameters 验证初始化参数
func (m *AuthnModule) validateParameters(params []interface{}) (*gorm.DB, *redis.Client, *IDPModule, error) {
	if len(params) < 2 {
		return nil, nil, nil, errors.WithCode(code.ErrModuleInitializationFailed, "missing required parameters: db and redis client")
	}

	db := params[0].(*gorm.DB)
	if db == nil {
		return nil, nil, nil, errors.WithCode(code.ErrModuleInitializationFailed, "database connection is nil")
	}

	redisClient := params[1].(*redis.Client)
	if redisClient == nil {
		return nil, nil, nil, errors.WithCode(code.ErrModuleInitializationFailed, "redis client is nil")
	}

	// IDP 模块是可选的
	var idpModule *IDPModule
	if len(params) >= 3 && params[2] != nil {
		idpModule = params[2].(*IDPModule)
	}

	return db, redisClient, idpModule, nil
}

// infrastructureComponents 基础设施层组件
type infrastructureComponents struct {
	// 仓储（使用接口类型）
	accountRepo   acctDriven.AccountRepo
	operationRepo acctDriven.OperationRepo
	wechatRepo    acctDriven.WeChatRepo
	keyRepo       jwksDriven.KeyRepository

	// 适配器（使用接口类型）
	userAdapter       adapter.UserAdapter
	passwordAdapter   *mysqlacct.PasswordAdapter
	wechatAuthAdapter *wechat.AuthAdapter

	// 存储（使用指针类型）
	tokenStore *redistoken.RedisStore

	// 事务（使用接口类型）
	unitOfWork uow.UnitOfWork

	// JWKS 组件
	privateKeyStorage jwksDriven.PrivateKeyStorage
	keyGenerator      jwksDriven.KeyGenerator
	privKeyResolver   jwksDriven.PrivateKeyResolver
	jwtGenerator      *jwt.Generator
}

// initializeInfrastructure 初始化基础设施层
func (m *AuthnModule) initializeInfrastructure(db *gorm.DB, redisClient *redis.Client, idpModule *IDPModule) (*infrastructureComponents, error) {
	infra := &infrastructureComponents{}

	// MySQL 仓储
	infra.accountRepo = mysqlacct.NewAccountRepository(db)
	infra.operationRepo = mysqlacct.NewOperationRepository(db)
	infra.wechatRepo = mysqlacct.NewWeChatRepository(db)
	infra.keyRepo = jwksMysql.NewKeyRepository(db)

	// 用户仓储（防腐层）
	userRepo := mysqluser.NewRepository(db)
	infra.userAdapter = adapter.NewUserAdapter(userRepo)

	// 密码适配器
	infra.passwordAdapter = mysqlacct.NewPasswordAdapter(infra.operationRepo)

	// 微信适配器（使用 IDP 模块的应用服务）
	if idpModule != nil && idpModule.WechatAuthService != nil {
		infra.wechatAuthAdapter = wechat.NewAuthAdapter(
			idpModule.WechatAuthService,
		)
		log.Info("✅ WeChatAuthAdapter initialized with IDP application service")
	} else {
		log.Warn("⚠️  IDP module not provided, WeChat authentication will not be available")
	}

	// Redis Token Store
	infra.tokenStore = redistoken.NewRedisStore(redisClient)

	// 事务 UnitOfWork
	infra.unitOfWork = uow.NewUnitOfWork(db)

	// JWKS 基础设施
	if err := m.initializeJWKSInfrastructure(infra); err != nil {
		return nil, err
	}

	return infra, nil
}

// initializeJWKSInfrastructure 初始化 JWKS 基础设施组件
func (m *AuthnModule) initializeJWKSInfrastructure(infra *infrastructureComponents) error {
	keysDir := viper.GetString("jwks.keys_dir")

	// 私钥存储
	infra.privateKeyStorage = crypto.NewPEMPrivateKeyStorage(keysDir)

	// 带持久化的密钥生成器
	infra.keyGenerator = crypto.NewRSAKeyGeneratorWithStorage(infra.privateKeyStorage)

	// 私钥解析器
	infra.privKeyResolver = crypto.NewPEMPrivateKeyResolver(keysDir)

	// JWKS 领域服务（需要先创建，因为 JWT Generator 依赖它）
	keyManager := jwksService.NewKeyManager(infra.keyRepo, infra.keyGenerator)

	// JWT Generator
	infra.jwtGenerator = jwt.NewGenerator(
		viper.GetString("auth.jwt_issuer"),
		keyManager,
		infra.privKeyResolver,
	)

	return nil
}

// domainComponents 领域层组件
type domainComponents struct {
	// 认证器
	authenticator *authService.Authenticator

	// 令牌服务
	tokenIssuer    *tokenService.TokenIssuer
	tokenRefresher *tokenService.TokenRefresher
	tokenVerifyer  *tokenService.TokenVerifyer

	// JWKS 服务
	keyManager    *jwksService.KeyManager
	keySetBuilder *jwksService.KeySetBuilder
	keyRotation   *jwksService.KeyRotation
}

// initializeDomain 初始化领域层
func (m *AuthnModule) initializeDomain(infra *infrastructureComponents) (*domainComponents, error) {
	domain := &domainComponents{}

	// 认证器
	domain.authenticator = authService.NewAuthenticator(
		authService.NewBasicAuthenticator(infra.accountRepo, infra.operationRepo, infra.passwordAdapter),
		authService.NewWeChatAuthenticator(infra.accountRepo, infra.wechatRepo, infra.wechatAuthAdapter),
	)

	// 令牌服务
	accessTokenTTL := viper.GetDuration("auth.access_token_ttl")
	refreshTokenTTL := viper.GetDuration("auth.refresh_token_ttl")

	domain.tokenIssuer = tokenService.NewTokenIssuer(
		infra.jwtGenerator,
		infra.tokenStore,
		accessTokenTTL,
		refreshTokenTTL,
	)

	domain.tokenRefresher = tokenService.NewTokenRefresher(
		infra.jwtGenerator,
		infra.tokenStore,
		accessTokenTTL,
		refreshTokenTTL,
	)

	domain.tokenVerifyer = tokenService.NewTokenVerifyer(
		infra.jwtGenerator,
		infra.tokenStore,
	)

	// JWKS 领域服务
	domain.keyManager = jwksService.NewKeyManager(infra.keyRepo, infra.keyGenerator)
	domain.keySetBuilder = jwksService.NewKeySetBuilder(infra.keyRepo)

	// 密钥轮换服务
	rotationPolicy := jwks.DefaultRotationPolicy()
	logger := log.New(log.NewOptions())
	domain.keyRotation = jwksService.NewKeyRotation(
		infra.keyRepo,
		infra.keyGenerator,
		rotationPolicy,
		logger,
	)

	return domain, nil
}

// initializeApplication 初始化应用层
func (m *AuthnModule) initializeApplication(infra *infrastructureComponents, domain *domainComponents) error {
	// 账户应用服务
	m.AccountService = accountApp.NewAccountApplicationService(infra.unitOfWork, infra.userAdapter)
	m.OperationAccountService = accountApp.NewOperationAccountApplicationService(infra.unitOfWork)
	m.WeChatAccountService = accountApp.NewWeChatAccountApplicationService(infra.unitOfWork)
	m.LookupService = accountApp.NewAccountLookupApplicationService(infra.unitOfWork)

	// 认证服务
	m.LoginService = login.NewLoginService(domain.authenticator, domain.tokenIssuer)
	m.TokenService = token.NewTokenService(domain.tokenIssuer, domain.tokenRefresher, domain.tokenVerifyer)

	// JWKS 应用服务
	logger := log.New(log.NewOptions())
	m.KeyManagementApp = jwksApp.NewKeyManagementAppService(domain.keyManager, logger)
	m.KeyPublishApp = jwksApp.NewKeyPublishAppService(domain.keySetBuilder, logger)
	m.KeyRotationApp = jwksApp.NewKeyRotationAppService(domain.keyRotation, logger)

	return nil
}

// initializeInterface 初始化接口层
func (m *AuthnModule) initializeInterface() error {
	// HTTP 处理器
	m.AccountHandler = authhandler.NewAccountHandler(
		m.AccountService,
		m.OperationAccountService,
		m.WeChatAccountService,
		m.LookupService,
	)

	m.AuthHandler = authhandler.NewAuthHandler(
		m.LoginService,
		m.TokenService,
	)

	m.JWKSHandler = authhandler.NewJWKSHandler(
		m.KeyManagementApp,
		m.KeyPublishApp,
	)

	return nil
}

// initializeSchedulers 初始化调度器
func (m *AuthnModule) initializeSchedulers() {
	logger := log.New(log.NewOptions())

	// ========================================
	// 使用 Cron 调度器（推荐生产环境）
	// ========================================
	// 优势：资源节省 95.8%，精确时间控制
	cronSpec := "0 2 * * *" // 每天凌晨2点检查一次

	m.RotationScheduler = scheduler.NewKeyRotationCronScheduler(
		m.KeyRotationApp,
		cronSpec,
		logger,
	)

	log.Infow("Key rotation scheduler initialized",
		"type", "cron",
		"cronSpec", cronSpec,
		"description", "每天凌晨2点检查密钥轮换",
	)

	// ========================================
	// Ticker 调度器（已弃用，保留供参考）
	// ========================================
	// 如需切换回 Ticker 方式，取消以下注释并注释掉上面的 Cron 配置：
	//
	// checkInterval := 1 * time.Hour
	// m.RotationScheduler = scheduler.NewKeyRotationScheduler(
	// 	m.KeyRotationApp,
	// 	checkInterval,
	// 	logger,
	// )
	// log.Infow("Key rotation scheduler initialized",
	// 	"type", "ticker",
	// 	"checkInterval", checkInterval,
	// )

	_ = logger // 避免未使用的变量警告
}

// StartSchedulers 启动调度器
func (m *AuthnModule) StartSchedulers(ctx context.Context) error {
	if m.RotationScheduler == nil {
		log.Info("Key rotation scheduler not initialized, skipping")
		return nil
	}

	if err := m.RotationScheduler.Start(ctx); err != nil {
		return errors.WithCode(code.ErrUnknown, "failed to start rotation scheduler: %v", err)
	}

	log.Info("All schedulers started successfully")
	return nil
}

// StopSchedulers 停止调度器
func (m *AuthnModule) StopSchedulers() error {
	if m.RotationScheduler == nil {
		return nil
	}

	if err := m.RotationScheduler.Stop(); err != nil {
		return errors.WithCode(code.ErrUnknown, "failed to stop rotation scheduler: %v", err)
	}

	log.Info("All schedulers stopped successfully")
	return nil
}

// CheckHealth 检查模块健康状态
func (m *AuthnModule) CheckHealth() error {
	return nil
}

// Cleanup 清理模块资源
func (m *AuthnModule) Cleanup() error {
	// 停止所有调度器
	if err := m.StopSchedulers(); err != nil {
		log.Warnw("Failed to stop schedulers", "error", err)
	}
	return nil
}

// ModuleInfo 返回模块信息
func (m *AuthnModule) ModuleInfo() ModuleInfo {
	return ModuleInfo{
		Name:        "auth",
		Version:     "1.0.0",
		Description: "认证模块 - 支持多种认证方式和令牌管理",
	}
}

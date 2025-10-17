package assembler

import (
	"github.com/go-redis/redis/v7"
	"github.com/spf13/viper"
	"gorm.io/gorm"

	accountApp "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/application/account"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/application/adapter"
	jwksApp "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/application/jwks"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/application/login"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/application/token"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/application/uow"
	acctDriven "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/account/port/driven"
	jwksDriven "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/jwks/port/driven"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/infra/crypto"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/infra/jwt"
	mysqlacct "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/infra/mysql/account"
	jwksMysql "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/infra/mysql/jwks"
	redistoken "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/infra/redis/token"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/infra/wechat"
	authhandler "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/interface/restful/handler"
	mysqluser "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/infra/mysql/user"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	"github.com/fangcun-mount/iam-contracts/pkg/errors"
	"github.com/fangcun-mount/iam-contracts/pkg/log"

	authService "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/authentication/service/authenticator"
	tokenService "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/authentication/service/token"
	jwksService "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/jwks/service"
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

	// HTTP 处理器
	AccountHandler *authhandler.AccountHandler
	AuthHandler    *authhandler.AuthHandler
	JWKSHandler    *authhandler.JWKSHandler
}

// NewAuthnModule 创建认证模块
func NewAuthnModule() *AuthnModule {
	return &AuthnModule{}
}

// Initialize 初始化模块
// params[0]: *gorm.DB - 数据库连接
// params[1]: *redis.Client - Redis 客户端
func (m *AuthnModule) Initialize(params ...interface{}) error {
	// 验证参数
	db, redisClient, err := m.validateParameters(params)
	if err != nil {
		return err
	}

	// 初始化基础设施层
	infra, err := m.initializeInfrastructure(db, redisClient)
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

	return nil
}

// validateParameters 验证初始化参数
func (m *AuthnModule) validateParameters(params []interface{}) (*gorm.DB, *redis.Client, error) {
	if len(params) < 2 {
		return nil, nil, errors.WithCode(code.ErrModuleInitializationFailed, "missing required parameters: db and redis client")
	}

	db := params[0].(*gorm.DB)
	if db == nil {
		return nil, nil, errors.WithCode(code.ErrModuleInitializationFailed, "database connection is nil")
	}

	redisClient := params[1].(*redis.Client)
	if redisClient == nil {
		return nil, nil, errors.WithCode(code.ErrModuleInitializationFailed, "redis client is nil")
	}

	return db, redisClient, nil
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
func (m *AuthnModule) initializeInfrastructure(db *gorm.DB, redisClient *redis.Client) (*infrastructureComponents, error) {
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

	// 微信适配器
	infra.wechatAuthAdapter = wechat.NewAuthAdapter()
	// TODO: 从配置加载微信应用配置
	// infra.wechatAuthAdapter.WithAppConfig("wx1234567890", "your-app-secret")

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

// CheckHealth 检查模块健康状态
func (m *AuthnModule) CheckHealth() error {
	return nil
}

// Cleanup 清理模块资源
func (m *AuthnModule) Cleanup() error {
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

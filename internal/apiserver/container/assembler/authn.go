package assembler

import (
	"context"
	"fmt"

	redis "github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"gorm.io/gorm"

	"github.com/FangcunMount/component-base/pkg/log"
	accountApp "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/application/account"
	jwksApp "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/application/jwks"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/application/login"
	registerApp "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/application/register"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/application/token"
	authnUow "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/application/uow"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/authentication"
	authPort "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/authentication/port"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/jwks"
	jwksPort "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/jwks/port/driven"
	jwksService "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/jwks/service"
	tokenService "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/token/service"
	authnInfra "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/infra/authentication"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/infra/crypto"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/infra/jwt"
	acctrepo "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/infra/mysql/account"
	jwksMysql "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/infra/mysql/jwks"
	redisOTP "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/infra/redis"
	redistoken "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/infra/redis/token"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/infra/scheduler"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/infra/wechat"
	authhandler "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/interface/restful/handler"
	userPort "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/domain/user/port"
	mysqluser "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/infra/mysql/user"
)

// AuthnModule 认证模块
type AuthnModule struct {
	// 应用服务
	AccountService  accountApp.AccountApplicationService
	RegisterService registerApp.RegisterApplicationService
	LoginService    login.LoginApplicationService
	TokenService    token.TokenApplicationService

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
// params[0]: *gorm.DB
// params[1]: *redis.Client
// params[2]: authPort.PasswordHasher (可选)
func (m *AuthnModule) Initialize(params ...interface{}) error {
	if len(params) < 2 {
		log.Errorf("AuthnModule.Initialize requires at least 2 parameters: db, redisClient")
		return fmt.Errorf("requires at least 2 parameters")
	}

	db, ok := params[0].(*gorm.DB)
	if !ok {
		log.Errorf("params[0] must be *gorm.DB")
		return fmt.Errorf("invalid db parameter type")
	}

	redisClient, ok := params[1].(*redis.Client)
	if !ok {
		log.Errorf("params[1] must be *redis.Client")
		return fmt.Errorf("invalid redis parameter type")
	}

	// 获取可选的密码哈希器
	var hasher authPort.PasswordHasher
	if len(params) >= 3 {
		if h, ok := params[2].(authPort.PasswordHasher); ok {
			hasher = h
		}
	}
	if hasher == nil {
		hasher = crypto.NewArgon2Hasher("")
	}

	// 初始化基础设施层
	infra := m.initializeInfrastructure(db, redisClient)

	// 初始化领域层
	domain := m.initializeDomain(infra)

	// 初始化应用层
	m.initializeApplication(infra, domain, hasher)

	// 初始化接口层
	m.initializeInterface()

	// 初始化调度器
	m.initializeSchedulers()

	return nil
}

// infrastructureComponents 基础设施层组件
type infrastructureComponents struct {
	db         *gorm.DB
	redis      *redis.Client
	unitOfWork authnUow.UnitOfWork

	accountRepo    authPort.AccountRepository
	credentialRepo authPort.CredentialRepository
	otpVerifier    authPort.OTPVerifier
	idp            authPort.IdentityProvider
	tokenVerifier  authPort.TokenVerifier

	// JWKS 相关
	keyRepo           jwksPort.KeyRepository
	privateKeyStorage jwksPort.PrivateKeyStorage
	keyGenerator      jwksPort.KeyGenerator
	privKeyResolver   jwksPort.PrivateKeyResolver
	jwtGenerator      *jwt.Generator

	// Token 存储
	tokenStore *redistoken.RedisStore

	// User 仓储
	userRepo userPort.UserRepository
}

// initializeInfrastructure 初始化基础设施层
func (m *AuthnModule) initializeInfrastructure(db *gorm.DB, redisClient *redis.Client) *infrastructureComponents {
	infra := &infrastructureComponents{
		db:    db,
		redis: redisClient,
	}

	// UnitOfWork
	infra.unitOfWork = authnUow.NewUnitOfWork(db)
	infra.accountRepo = acctrepo.NewAccountRepository(db)
	infra.credentialRepo = acctrepo.NewCredentialRepository(db)

	// OTP 验证器
	infra.otpVerifier = redisOTP.NewOTPVerifier(redisClient)

	// 身份提供商 (微信)
	// 配置在使用时动态传递，支持多个小程序/企业微信
	infra.idp = wechat.NewIdentityProvider(nil)

	// JWKS 仓储
	infra.keyRepo = jwksMysql.NewKeyRepository(db)

	// JWKS 基础设施
	keysDir := viper.GetString("jwks.keys_dir")
	infra.privateKeyStorage = crypto.NewPEMPrivateKeyStorage(keysDir)
	infra.keyGenerator = crypto.NewRSAKeyGeneratorWithStorage(infra.privateKeyStorage)
	infra.privKeyResolver = crypto.NewPEMPrivateKeyResolver(keysDir)

	// Token Store
	infra.tokenStore = redistoken.NewRedisStore(redisClient)

	// User 仓储（跨模块依赖）
	infra.userRepo = mysqluser.NewRepository(db)

	return infra
}

// domainComponents 领域层组件
type domainComponents struct {
	// Token 服务
	tokenIssuer    *tokenService.TokenIssuer
	tokenRefresher *tokenService.TokenRefresher
	tokenVerifyer  *tokenService.TokenVerifyer

	// JWKS 服务
	keyManager    *jwksService.KeyManager
	keySetBuilder *jwksService.KeySetBuilder
	keyRotation   *jwksService.KeyRotation
}

// initializeDomain 初始化领域层
func (m *AuthnModule) initializeDomain(infra *infrastructureComponents) *domainComponents {
	domain := &domainComponents{}

	// JWKS 领域服务
	domain.keyManager = jwksService.NewKeyManager(infra.keyRepo, infra.keyGenerator)
	domain.keySetBuilder = jwksService.NewKeySetBuilder(infra.keyRepo)

	rotationPolicy := jwks.DefaultRotationPolicy()
	logger := log.New(log.NewOptions())
	domain.keyRotation = jwksService.NewKeyRotation(
		infra.keyRepo,
		infra.keyGenerator,
		rotationPolicy,
		logger,
	)

	// JWT Generator（依赖 JWKS）
	infra.jwtGenerator = jwt.NewGenerator(
		viper.GetString("auth.jwt_issuer"),
		domain.keyManager,
		infra.privKeyResolver,
	)

	// Token 领域服务
	accessTTL := viper.GetDuration("auth.access_token_ttl")
	if accessTTL == 0 {
		accessTTL = 15 * 60 * 1000000000 // 15分钟（纳秒）
	}
	refreshTTL := viper.GetDuration("auth.refresh_token_ttl")
	if refreshTTL == 0 {
		refreshTTL = 7 * 24 * 60 * 60 * 1000000000 // 7天（纳秒）
	}

	domain.tokenIssuer = tokenService.NewTokenIssuer(infra.jwtGenerator, infra.tokenStore, accessTTL, refreshTTL)
	domain.tokenRefresher = tokenService.NewTokenRefresher(infra.jwtGenerator, infra.tokenStore, accessTTL, refreshTTL)
	domain.tokenVerifyer = tokenService.NewTokenVerifyer(infra.jwtGenerator, infra.tokenStore)

	// 创建 TokenVerifier 适配器供 authentication 模块使用
	infra.tokenVerifier = authnInfra.NewTokenVerifierAdapter(domain.tokenVerifyer)

	return domain
}

// initializeApplication 初始化应用层
func (m *AuthnModule) initializeApplication(
	infra *infrastructureComponents,
	domain *domainComponents,
	hasher authPort.PasswordHasher,
) {
	// 账户应用服务
	m.AccountService = accountApp.NewAccountApplicationService(infra.unitOfWork)

	// 注册服务
	m.RegisterService = registerApp.NewRegisterApplicationService(
		infra.unitOfWork,
		hasher,
		infra.userRepo,
	)

	m.LoginService = login.NewLoginApplicationService(
		domain.tokenIssuer,
		domain.tokenRefresher,
		authentication.NewAuthenticater(
			infra.credentialRepo,
			infra.accountRepo,
			hasher,
			infra.otpVerifier,
			infra.idp,
			infra.tokenVerifier,
		),
		nil, // TODO: 注入 wechatAppQuerier（需要初始化 idp 模块）
		nil, // TODO: 注入 secretVault（需要初始化 idp 模块）
	)

	// Token 服务
	m.TokenService = token.NewTokenApplicationService(
		domain.tokenIssuer,
		domain.tokenRefresher,
		domain.tokenVerifyer,
	)

	// JWKS 应用服务
	logger := log.New(log.NewOptions())
	m.KeyManagementApp = jwksApp.NewKeyManagementAppService(domain.keyManager, logger)
	m.KeyPublishApp = jwksApp.NewKeyPublishAppService(domain.keySetBuilder, logger)
	m.KeyRotationApp = jwksApp.NewKeyRotationAppService(domain.keyRotation, logger)
}

// initializeInterface 初始化接口层
func (m *AuthnModule) initializeInterface() {
	m.AccountHandler = authhandler.NewAccountHandler(
		m.AccountService,
		nil, // TODO: CredentialApplicationService
		m.RegisterService,
	)

	m.AuthHandler = authhandler.NewAuthHandler(
		m.LoginService,
		m.TokenService,
	)

	m.JWKSHandler = authhandler.NewJWKSHandler(
		m.KeyManagementApp,
		m.KeyPublishApp,
	)
}

// initializeSchedulers 初始化调度器
func (m *AuthnModule) initializeSchedulers() {
	logger := log.New(log.NewOptions())
	cronSpec := "0 2 * * *" // 每天凌晨2点

	m.RotationScheduler = scheduler.NewKeyRotationCronScheduler(
		m.KeyRotationApp,
		cronSpec,
		logger,
	)
}

// Cleanup 清理资源
func (m *AuthnModule) Cleanup(ctx context.Context) error {
	if m.RotationScheduler != nil && m.RotationScheduler.IsRunning() {
		if err := m.RotationScheduler.Stop(); err != nil {
			log.Warnf("Failed to stop rotation scheduler: %v", err)
		}
	}
	return nil
}

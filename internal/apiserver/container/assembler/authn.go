package assembler

import (
	"context"
	"fmt"

	redis "github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"gorm.io/gorm"

	"github.com/FangcunMount/component-base/pkg/log"
	accountApp "github.com/FangcunMount/iam-contracts/internal/apiserver/application/authn/account"
	jwksApp "github.com/FangcunMount/iam-contracts/internal/apiserver/application/authn/jwks"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/application/authn/login"
	registerApp "github.com/FangcunMount/iam-contracts/internal/apiserver/application/authn/register"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/application/authn/token"
	authnUow "github.com/FangcunMount/iam-contracts/internal/apiserver/application/authn/uow"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/authentication"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/jwks"
	tokenDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/token"
	idpPort "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/idp/wechatapp"
	userDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/uc/user"
	authenticationInfra "github.com/FangcunMount/iam-contracts/internal/apiserver/infra/authentication"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/infra/crypto"
	jwtinfra "github.com/FangcunMount/iam-contracts/internal/apiserver/infra/jwt"
	acctrepo "github.com/FangcunMount/iam-contracts/internal/apiserver/infra/mysql/account"
	credentialrepo "github.com/FangcunMount/iam-contracts/internal/apiserver/infra/mysql/credential"
	jwksMysql "github.com/FangcunMount/iam-contracts/internal/apiserver/infra/mysql/jwks"
	mysqluser "github.com/FangcunMount/iam-contracts/internal/apiserver/infra/mysql/user"
	redisInfra "github.com/FangcunMount/iam-contracts/internal/apiserver/infra/redis"
	schedulerInfra "github.com/FangcunMount/iam-contracts/internal/apiserver/infra/scheduler"
	wechatInfra "github.com/FangcunMount/iam-contracts/internal/apiserver/infra/wechat"
	authhandler "github.com/FangcunMount/iam-contracts/internal/apiserver/interface/authn/restful/handler"
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
// 可选参数：
//   - authentication.PasswordHasher 自定义密码哈希器
//   - *IDPModule              注入 IDP 模块提供的基础设施能力
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

	// 获取可选依赖
	var (
		hasher  authentication.PasswordHasher
		idpDeps *IDPModule
	)
	for _, opt := range params[2:] {
		switch v := opt.(type) {
		case authentication.PasswordHasher:
			hasher = v
		case *IDPModule:
			idpDeps = v
		}
	}
	if hasher == nil {
		hasher = crypto.NewArgon2Hasher("")
	}

	// 初始化基础设施层
	infra := m.initializeInfrastructure(db, redisClient, idpDeps)

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

	accountRepo    authentication.AccountRepository
	credentialRepo authentication.CredentialRepository
	otpVerifier    authentication.OTPVerifier
	idp            authentication.IdentityProvider
	tokenVerifier  authentication.TokenVerifier

	// JWKS 相关
	keyRepo           jwks.Repository
	privateKeyStorage jwks.PrivateKeyStorage
	keyGenerator      jwks.KeyGenerator
	privKeyResolver   jwks.PrivateKeyResolver
	jwtGenerator      *jwtinfra.Generator

	// Token 存储
	tokenStore *redisInfra.RedisStore

	// User 仓储
	userRepo userDomain.Repository

	// IDP 基础设施
	wechatAppQuerier idpPort.Repository
	secretVault      idpPort.SecretVault
}

// initializeInfrastructure 初始化基础设施层
func (m *AuthnModule) initializeInfrastructure(db *gorm.DB, redisClient *redis.Client, idpDeps *IDPModule) *infrastructureComponents {
	infra := &infrastructureComponents{
		db:    db,
		redis: redisClient,
	}

	// UnitOfWork
	infra.unitOfWork = authnUow.NewUnitOfWork(db)
	infra.accountRepo = acctrepo.NewAccountRepository(db)
	infra.credentialRepo = credentialrepo.NewRepository(db)

	// OTP 验证器
	infra.otpVerifier = redisInfra.NewOTPVerifier(redisClient)

	// 身份提供商 (微信)
	// 优先使用 IDP 模块提供的基础设施能力
	if idpDeps != nil {
		infra.wechatAppQuerier = idpDeps.Repository()
		infra.secretVault = idpDeps.SecretVault()
		if provider := idpDeps.WechatAuthProvider(); provider != nil {
			infra.idp = wechatInfra.NewIdentityProvider(provider, nil)
		}
	}
	if infra.idp == nil {
		infra.idp = wechatInfra.NewIdentityProvider(nil, nil)
	}

	// JWKS 仓储
	infra.keyRepo = jwksMysql.NewKeyRepository(db)

	// JWKS 基础设施
	keysDir := viper.GetString("jwks.keys_dir")
	infra.privateKeyStorage = crypto.NewPEMPrivateKeyStorage(keysDir)
	infra.keyGenerator = crypto.NewRSAKeyGeneratorWithStorage(infra.privateKeyStorage)
	infra.privKeyResolver = crypto.NewPEMPrivateKeyResolver(keysDir)

	// Token Store
	infra.tokenStore = redisInfra.NewRedisStore(redisClient)

	// User 仓储（跨模块依赖）
	infra.userRepo = mysqluser.NewRepository(db)

	return infra
}

// domainComponents 领域层组件
type domainComponents struct {
	// Token 服务
	tokenIssuer    *tokenDomain.TokenIssuer
	tokenRefresher *tokenDomain.TokenRefresher
	tokenVerifyer  *tokenDomain.TokenVerifyer

	// JWKS 服务
	keyManager    *jwks.KeyManager
	keySetBuilder *jwks.KeySetBuilder
	keyRotation   *jwks.KeyRotation
}

// initializeDomain 初始化领域层
func (m *AuthnModule) initializeDomain(infra *infrastructureComponents) *domainComponents {
	domain := &domainComponents{}

	// JWKS 领域服务
	domain.keyManager = jwks.NewKeyManager(infra.keyRepo, infra.keyGenerator)
	domain.keySetBuilder = jwks.NewKeySetBuilder(infra.keyRepo)

	rotationPolicy := jwks.DefaultRotationPolicy()
	logger := log.New(log.NewOptions())
	domain.keyRotation = jwks.NewKeyRotation(
		infra.keyRepo,
		infra.keyGenerator,
		rotationPolicy,
		logger,
	)

	// JWT Generator（依赖 JWKS）
	infra.jwtGenerator = jwtinfra.NewGenerator(
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

	domain.tokenIssuer = tokenDomain.NewTokenIssuer(infra.jwtGenerator, infra.tokenStore, accessTTL, refreshTTL)
	domain.tokenRefresher = tokenDomain.NewTokenRefresher(infra.jwtGenerator, infra.tokenStore, accessTTL, refreshTTL)
	domain.tokenVerifyer = tokenDomain.NewTokenVerifyer(infra.jwtGenerator, infra.tokenStore)

	// 创建 TokenVerifier 适配器供 authentication 模块使用
	infra.tokenVerifier = authenticationInfra.NewTokenVerifierAdapter(domain.tokenVerifyer)

	return domain
}

// initializeApplication 初始化应用层
func (m *AuthnModule) initializeApplication(
	infra *infrastructureComponents,
	domain *domainComponents,
	hasher authentication.PasswordHasher,
) {
	// 账户应用服务
	m.AccountService = accountApp.NewAccountApplicationService(infra.unitOfWork)

	// 注册服务
	m.RegisterService = registerApp.NewRegisterApplicationService(
		infra.unitOfWork,
		hasher,
		infra.idp, // 添加 IDP 参数
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
		infra.wechatAppQuerier,
		infra.secretVault,
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

	m.RotationScheduler = schedulerInfra.NewKeyRotationCronScheduler(
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

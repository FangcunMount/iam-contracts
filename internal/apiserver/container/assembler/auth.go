package assembler

import (
	"github.com/go-redis/redis/v7"
	"github.com/spf13/viper"
	"gorm.io/gorm"

	accountApp "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/application/account"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/application/adapter"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/application/login"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/application/token"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/application/uow"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/infra/jwt"
	mysqlacct "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/infra/mysql/account"
	redistoken "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/infra/redis/token"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/infra/wechat"
	authhandler "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/interface/restful/handler"
	mysqluser "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/infra/mysql/user"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	"github.com/fangcun-mount/iam-contracts/pkg/errors"

	authService "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/authentication/service/authenticator"
	tokenService "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/authentication/service/token"
)

// AuthModule 认证模块
// 负责组装认证相关的所有组件
type AuthModule struct {
	// 账户应用服务
	AccountService          accountApp.AccountApplicationService
	OperationAccountService accountApp.OperationAccountApplicationService
	WeChatAccountService    accountApp.WeChatAccountApplicationService
	LookupService           accountApp.AccountLookupApplicationService

	// 认证服务
	LoginService *login.LoginService
	TokenService *token.TokenService

	// HTTP 处理器
	AccountHandler *authhandler.AccountHandler
	AuthHandler    *authhandler.AuthHandler
}

// NewAuthModule 创建认证模块
func NewAuthModule() *AuthModule {
	return &AuthModule{}
}

// Initialize 初始化模块
// params[0]: *gorm.DB - 数据库连接
// params[1]: *redis.Client - Redis 客户端
func (m *AuthModule) Initialize(params ...interface{}) error {
	if len(params) < 2 {
		return errors.WithCode(code.ErrModuleInitializationFailed, "missing required parameters: db and redis client")
	}

	db := params[0].(*gorm.DB)
	if db == nil {
		return errors.WithCode(code.ErrModuleInitializationFailed, "database connection is nil")
	}

	redisClient := params[1].(*redis.Client)
	if redisClient == nil {
		return errors.WithCode(code.ErrModuleInitializationFailed, "redis client is nil")
	}

	// ========== 基础设施层 ==========

	// MySQL 仓储
	accountRepo := mysqlacct.NewAccountRepository(db)
	operationRepo := mysqlacct.NewOperationRepository(db)
	wechatRepo := mysqlacct.NewWeChatRepository(db)

	// 用户仓储（防腐层）
	userRepo := mysqluser.NewRepository(db)
	userAdapter := adapter.NewUserAdapter(userRepo)

	// 密码适配器
	passwordAdapter := mysqlacct.NewPasswordAdapter(operationRepo)

	// 微信适配器
	wechatAuthAdapter := wechat.NewAuthAdapter()
	// TODO: 从配置加载微信应用配置
	// wechatAuthAdapter.WithAppConfig("wx1234567890", "your-app-secret")

	// JWT Generator
	jwtGenerator := jwt.NewGenerator(
		viper.GetString("jwt.secret"), // TODO: 从配置加载
		"iam-apiserver",               // issuer
	)

	// Redis Token Store
	tokenStore := redistoken.NewRedisStore(redisClient)

	// 事务 UnitOfWork
	unitOfWork := uow.NewUnitOfWork(db)

	// ========== 领域层 ==========

	// 认证器
	authenticator := authService.NewAuthenticator(
		authService.NewBasicAuthenticator(accountRepo, operationRepo, passwordAdapter),
		authService.NewWeChatAuthenticator(accountRepo, wechatRepo, wechatAuthAdapter),
	)

	// 令牌服务
	tokenIssuer := tokenService.NewTokenIssuer(jwtGenerator, tokenStore, viper.GetDuration("auth.access_token_ttl"), viper.GetDuration("auth.refresh_token_ttl"))
	tokenRefresher := tokenService.NewTokenRefresher(jwtGenerator, tokenStore, viper.GetDuration("auth.access_token_ttl"), viper.GetDuration("auth.refresh_token_ttl"))
	tokenVerifyer := tokenService.NewTokenVerifyer(jwtGenerator, tokenStore)

	// ========== 应用层 ==========

	// 账户应用服务
	m.AccountService = accountApp.NewAccountApplicationService(unitOfWork, userAdapter)
	m.OperationAccountService = accountApp.NewOperationAccountApplicationService(unitOfWork)
	m.WeChatAccountService = accountApp.NewWeChatAccountApplicationService(unitOfWork)
	m.LookupService = accountApp.NewAccountLookupApplicationService(unitOfWork)

	// 认证服务
	m.LoginService = login.NewLoginService(authenticator, tokenIssuer)
	m.TokenService = token.NewTokenService(tokenIssuer, tokenRefresher, tokenVerifyer)

	// ========== 接口层 ==========

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

	return nil
}

// CheckHealth 检查模块健康状态
func (m *AuthModule) CheckHealth() error {
	return nil
}

// Cleanup 清理模块资源
func (m *AuthModule) Cleanup() error {
	return nil
}

// ModuleInfo 返回模块信息
func (m *AuthModule) ModuleInfo() ModuleInfo {
	return ModuleInfo{
		Name:        "auth",
		Version:     "1.0.0",
		Description: "认证模块 - 支持多种认证方式和令牌管理",
	}
}

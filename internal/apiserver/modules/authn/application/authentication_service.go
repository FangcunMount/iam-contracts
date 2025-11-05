package application

import (
	"context"

	"github.com/go-redis/redis/v8"
	wechatCache "github.com/silenceper/wechat/v2/cache"
	"gorm.io/gorm"

	// 领域层
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/authentication"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/authentication/service"
	tokenPort "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/token/port"

	// 基础设施层适配器
	authInfra "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/infra/authentication"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/infra/crypto"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/infra/mysql/account"
	redisAdapter "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/infra/redis"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/infra/wechat"
)

// AuthenticationService 认证应用服务（应用层入口）
type AuthenticationService struct {
	strategyFactory *service.StrategyFactory
}

// NewAuthenticationService 创建认证应用服务（依赖注入）
// 这是整个认证模块的组装点，遵循依赖倒置原则：
// 1. 领域层定义端口（接口）
// 2. 基础设施层实现适配器
// 3. 应用层负责组装（Wire/DI容器）
func NewAuthenticationService(
	db *gorm.DB, // 使用 gorm.DB 而不是 sql.DB
	redisClient *redis.Client,
	wechatSDKCache wechatCache.Cache, // 微信SDK使用的缓存
	pepper string,
	wxMinipSecrets map[string]string,
	wecomSecrets map[string]wechat.WecomConfig,
	tokenVerifier tokenPort.TokenVerifier, // JWT令牌验证器（来自token模块）
) *AuthenticationService {
	// 创建所有基础设施适配器（Driven Adapters）
	// 使用现有的仓储实现
	credRepo := account.NewCredentialRepository(db)
	accountRepo := account.NewAccountRepository(db)
	hasher := crypto.NewArgon2Hasher(pepper)
	otpVerifier := redisAdapter.NewOTPVerifier(redisClient)
	idp := wechat.NewIdentityProvider(wechatSDKCache, wxMinipSecrets, wecomSecrets)

	// 创建 TokenVerifier 适配器（将 token 模块的接口适配为 authentication 模块需要的接口）
	authTokenVerifier := authInfra.NewTokenVerifierAdapter(tokenVerifier)

	// 创建策略工厂（注入所有端口实现）
	factory := service.NewStrategyFactory(
		credRepo,
		accountRepo,
		hasher,
		otpVerifier,
		idp,
		authTokenVerifier, // 注入 TokenVerifier
	)

	return &AuthenticationService{
		strategyFactory: factory,
	}
}

// Authenticate 执行认证（用例入口）
// 这是外部调用的主要方法，遵循用例模式
func (s *AuthenticationService) Authenticate(
	ctx context.Context,
	scenario domain.Scenario,
	input domain.AuthInput,
) (domain.AuthDecision, error) {
	// 1. 通过工厂获取认证策略
	strategy := s.strategyFactory.CreateStrategy(scenario)
	if strategy == nil {
		return domain.AuthDecision{
			OK:      false,
			ErrCode: "unsupported_scenario",
		}, nil
	}

	// 2. 执行认证
	decision, err := strategy.Authenticate(ctx, input)
	if err != nil {
		// 系统异常（数据库、网络等错误）
		return domain.AuthDecision{}, err
	}

	// 3. 可选：记录审计日志
	// s.auditLogger.LogAuthAttempt(ctx, ...)

	// 4. 可选：处理密码rehash
	if decision.OK && decision.ShouldRotate && len(decision.NewMaterial) > 0 {
		// 异步更新密码哈希（算法升级）
		go s.updatePasswordHashAsync(decision.CredentialID, decision.NewMaterial)
	}

	return decision, nil
}

// updatePasswordHashAsync 异步更新密码哈希
func (s *AuthenticationService) updatePasswordHashAsync(credentialID int64, newHash []byte) {
	// TODO: 实现异步更新逻辑
	// 1. 更新数据库中的password_hash字段
	// 2. 记录更新日志
}

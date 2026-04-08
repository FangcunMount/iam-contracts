package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/component-base/pkg/util/idutil"
	registerApp "github.com/FangcunMount/iam-contracts/internal/apiserver/application/authn/register"
	authnUOW "github.com/FangcunMount/iam-contracts/internal/apiserver/application/authn/uow"
	accountDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/account"
	authnAuth "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/authentication"
	credentialDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/credential"
	wechatappDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/idp/wechatapp"
	userDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/uc/user"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/infra/crypto"
	accountRepo "github.com/FangcunMount/iam-contracts/internal/apiserver/infra/mysql/account"
	credentialRepo "github.com/FangcunMount/iam-contracts/internal/apiserver/infra/mysql/credential"
	userRepo "github.com/FangcunMount/iam-contracts/internal/apiserver/infra/mysql/user"
	wechatAppRepo "github.com/FangcunMount/iam-contracts/internal/apiserver/infra/mysql/wechatapp"
	wechatInfra "github.com/FangcunMount/iam-contracts/internal/apiserver/infra/wechat"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

// ==================== 认证 Seed 函数 ====================

const (
	simulatedWechatBusinessUserIDFloor uint64 = 100000
	simulatedWechatAppInternalID       uint64 = 613485615102571054
	simulatedWechatAppConfigAlias             = "questionnaire_notebook"
	simulatedWechatAccountAliasSuffix         = "_wechat_account"
)

// seedAuthn 创建认证账号数据
//
// 业务说明：
// 1. 为系统管理员和测试用户创建运营后台账号
// 2. 使用新的 RegisterApplicationService 进行账户注册
// 3. 为业务用户（user.id > 100000）自动补齐模拟微信小程序账号与登录凭据
// 4. 返回的 state 保存账号ID，供后续步骤使用
//
// 前置条件：必须先执行 user 步骤创建用户
// 幂等性：Register 服务内部会处理重复注册情况
func seedAuthn(ctx context.Context, deps *dependencies, state *seedContext) error {
	if len(state.Users) == 0 {
		return errors.New("user context is empty; run user step first")
	}

	config := deps.Config
	if config == nil {
		deps.Logger.Warnw("⚠️  配置文件为空，跳过账号初始化")
		return nil
	}

	// 初始化基础设施层
	unitOfWork := authnUOW.NewUnitOfWork(deps.DB)
	userRepository := userRepo.NewRepository(deps.DB)
	accountRepository := accountRepo.NewAccountRepository(deps.DB)
	credentialRepository := credentialRepo.NewRepository(deps.DB)
	wechatAppRepository := wechatAppRepo.NewWechatAppRepository(deps.DB)

	// 初始化领域服务（密码哈希器）
	pepper := os.Getenv("SEEDDATA_PASSWORD_PEPPER") // 与服务端保持一致，默认空字符串
	passwordHasher := crypto.NewArgon2Hasher(pepper)
	credentialIssuer := credentialDomain.NewIssuer(passwordHasher)

	// 初始化身份提供商（简单实现，用于 seeddata）
	idp := wechatInfra.NewIdentityProvider(nil, nil)

	// 初始化应用服务
	// 注意：seed 阶段仅支持密码注册，不需要微信相关功能，因此传入 nil
	registerService := registerApp.NewRegisterApplicationService(
		unitOfWork,
		passwordHasher,
		idp,
		userRepository,
		nil, // wechatAppQuerier - seed 阶段不需要
		nil, // secretVault - seed 阶段不需要
	)

	// 从配置文件读取账号数据
	for _, ac := range config.Accounts {
		// 当前仅支持 operation 账号（密码登录）
		if ac.Provider != "operation" {
			deps.Logger.Warnw("⚠️  暂不支持的账号类型，跳过",
				"account_alias", ac.Alias,
				"provider", ac.Provider)
			continue
		}
		if err := validateOperationAccountConfig(ac, 0); err != nil {
			return fmt.Errorf("invalid account config %s: %w", ac.Alias, err)
		}

		// 1. 获取用户基本信息
		userIDStr := state.Users[ac.UserAlias]
		if userIDStr == "" {
			deps.Logger.Warnw("⚠️  用户别名未找到，跳过账号创建",
				"account_alias", ac.Alias,
				"user_alias", ac.UserAlias)
			continue
		}

		// 2. 解析用户ID
		userID, err := parseAuthnUserID(userIDStr)
		if err != nil {
			return fmt.Errorf("parse user id %s: %w", userIDStr, err)
		}

		// 3. 获取用户完整信息（用于注册）
		user, err := userRepository.FindByID(ctx, userID)
		if err != nil {
			return fmt.Errorf("get user %s: %w", userID, err)
		}

		// 4. 执行注册（使用RegisterApplicationService）
		loginExternalID := accountOperaExternalID(ac, user.Email.String())
		req := registerApp.RegisterRequest{
			Name:           user.Name,
			Phone:          user.Phone,
			Email:          user.Email,
			ExistingUserID: userID,
			OperaLoginID:   loginExternalID,
			ScopedTenantID: meta.FromUint64(ac.ScopedTenantID),
			AccountType:    accountDomain.TypeOpera, // 运营账号类型
			CredentialType: registerApp.CredTypePassword,
			Password:       &ac.Password,
		}

		result, err := registerService.Register(ctx, req)
		if err != nil {
			// 支持重复运行：按策略选择跳过或覆盖
			if handled, accID, handleErr := handleAuthnConflict(ctx, deps, accountRepository, credentialRepository, passwordHasher, ac, userID, loginExternalID, err); handled {
				if handleErr != nil {
					return fmt.Errorf("register account %s: %w", ac.Alias, handleErr)
				}
				if accID != 0 {
					if syncErr := syncSeedAccountStatus(ctx, accountRepository, ac, meta.FromUint64(accID)); syncErr != nil {
						return fmt.Errorf("sync account status %s: %w", ac.Alias, syncErr)
					}
					state.Accounts[ac.Alias] = accID
				}
				continue
			}
			return fmt.Errorf("register account %s: %w", ac.Alias, err)
		}

		if err := syncSeedAccountStatus(ctx, accountRepository, ac, result.AccountID); err != nil {
			return fmt.Errorf("sync account status %s: %w", ac.Alias, err)
		}

		// 5. 保存账号ID到状态
		state.Accounts[ac.Alias] = result.AccountID.Uint64()
		deps.Logger.Infow("✅ 账号创建成功",
			"account_alias", ac.Alias,
			"account_id", result.AccountID.String(),
			"user_id", result.UserID.String(),
			"credential_id", result.CredentialID,
			"is_new_user", result.IsNewUser,
			"is_new_account", result.IsNewAccount)
	}

	if err := seedSimulatedWechatAuthn(
		ctx,
		deps,
		state,
		registerService,
		credentialIssuer,
		userRepository,
		accountRepository,
		credentialRepository,
		wechatAppRepository,
	); err != nil {
		return err
	}

	deps.Logger.Infow("✅ 认证账号数据已创建")
	return nil
}

func validateOperationAccountConfig(ac AccountConfig, scopedFallback uint64) error {
	appID := strings.TrimSpace(ac.AppID)
	if appID != "" && appID != "opera" {
		return fmt.Errorf("operation account app_id is fixed to opera, got %q", ac.AppID)
	}
	effective := ac.ScopedTenantID
	if effective == 0 {
		effective = scopedFallback
	}
	if effective == 0 {
		return fmt.Errorf("operation account requires scoped_tenant_id")
	}
	return nil
}

func configuredAccountStatus(ac AccountConfig) (accountDomain.AccountStatus, bool, error) {
	if ac.Status == nil {
		return 0, false, nil
	}

	switch *ac.Status {
	case int(accountDomain.StatusDisabled),
		int(accountDomain.StatusActive),
		int(accountDomain.StatusArchived),
		int(accountDomain.StatusDeleted):
		return accountDomain.AccountStatus(*ac.Status), true, nil
	default:
		return 0, false, fmt.Errorf("unsupported status %d", *ac.Status)
	}
}

func syncSeedAccountStatus(ctx context.Context, repo *accountRepo.AccountRepository, ac AccountConfig, accountID meta.ID) error {
	status, ok, err := configuredAccountStatus(ac)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	return repo.UpdateStatus(ctx, accountID, status)
}

// parseAuthnUserID 解析用户ID字符串为 meta.ID
func parseAuthnUserID(userIDStr string) (meta.ID, error) {
	var id uint64
	if _, err := fmt.Sscanf(userIDStr, "%d", &id); err != nil {
		return meta.FromUint64(0), fmt.Errorf("invalid user id format: %s", userIDStr)
	}
	return meta.FromUint64(id), nil
}

// handleAuthnConflict 处理账号/凭据已存在的场景，支持 skip/overwrite 策略
func handleAuthnConflict(
	ctx context.Context,
	deps *dependencies,
	accountRepo *accountRepo.AccountRepository,
	credentialRepo *credentialRepo.Repository,
	passwordHasher authnAuth.PasswordHasher,
	ac AccountConfig,
	userID meta.ID,
	externalID string,
	originalErr error,
) (handled bool, accountID uint64, err error) {
	// 非冲突错误，不处理
	if !isAuthnConflictError(originalErr) {
		return false, 0, nil
	}

	// 与 OperaCreatorStrategy 一致：运营账号 app_id 默认为 "opera"
	appID := accountDomain.AppId(ac.AppID)
	if appID == "" {
		appID = accountDomain.AppId("opera")
	}
	// 查询已存在账号（与注册时写入的 external_id 一致）
	existing, getErr := accountRepo.GetByExternalIDAppId(ctx,
		accountDomain.ExternalID(externalID),
		appID,
	)
	if getErr != nil {
		return true, 0, fmt.Errorf("fetch existing account: %w", getErr)
	}
	if existing == nil {
		return true, 0, fmt.Errorf("account already exists but not found by external_id=%s", externalID)
	}
	if existing.UserID != userID {
		return true, 0, fmt.Errorf("account %s belongs to another user", externalID)
	}

	switch deps.OnConflict {
	case "skip":
		deps.Logger.Infow("⚠️  账号已存在，按策略跳过",
			"account_alias", ac.Alias,
			"external_id", externalID,
			"strategy", "skip")
		return true, existing.ID.Uint64(), nil
	case "overwrite":
		// 覆盖密码：若有密码凭据则更新，否则创建
		cred, credErr := credentialRepo.GetByAccountIDAndType(ctx, existing.ID, credentialDomain.CredPassword)
		if credErr != nil {
			return true, 0, fmt.Errorf("get credential: %w", credErr)
		}

		issuer := credentialDomain.NewIssuer(passwordHasher)
		newCred, issueErr := issuer.IssuePassword(ctx, credentialDomain.IssuePasswordRequest{
			AccountID:     existing.ID,
			PlainPassword: ac.Password, // 由 issuer 内部加 pepper + hash
		})
		if issueErr != nil {
			return true, 0, fmt.Errorf("issue credential: %w", issueErr)
		}

		if cred != nil {
			if newCred.Algo == nil {
				return true, 0, fmt.Errorf("issued credential algo is nil")
			}
			if updErr := credentialRepo.UpdateMaterial(ctx, cred.ID, newCred.Material, *newCred.Algo); updErr != nil {
				return true, 0, fmt.Errorf("update credential: %w", updErr)
			}
		} else {
			if createErr := credentialRepo.Create(ctx, newCred); createErr != nil {
				return true, 0, fmt.Errorf("create credential: %w", createErr)
			}
		}

		deps.Logger.Infow("🔄  账号已存在，密码已覆盖",
			"account_alias", ac.Alias,
			"external_id", externalID,
			"strategy", "overwrite")
		return true, existing.ID.Uint64(), nil
	default:
		// fail 策略：交回调用方处理
		return false, 0, nil
	}
}

// isAuthnConflictError 识别账号/凭据唯一性冲突
func isAuthnConflictError(err error) bool {
	if err == nil {
		return false
	}
	if perrors.IsCode(err, code.ErrAccountExists) ||
		perrors.IsCode(err, code.ErrExternalExists) ||
		perrors.IsCode(err, code.ErrCredentialExists) {
		return true
	}
	// 兜底：字符串匹配
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "account already exists") ||
		strings.Contains(msg, "credential already exists") ||
		strings.Contains(msg, "duplicate")
}

func seedSimulatedWechatAuthn(
	ctx context.Context,
	deps *dependencies,
	state *seedContext,
	registerService registerApp.RegisterApplicationService,
	credentialIssuer credentialDomain.Issuer,
	userRepository userDomain.Repository,
	accountRepository *accountRepo.AccountRepository,
	credentialRepository *credentialRepo.Repository,
	wechatAppRepository wechatappDomain.Repository,
) error {
	userAliases, err := collectSimulatedWechatUserAliases(state.Users)
	if err != nil {
		return err
	}
	if len(userAliases) == 0 {
		deps.Logger.Debugw("ℹ️  没有需要生成模拟微信账号的业务用户")
		return nil
	}

	wechatAppID, err := resolveSimulatedWechatMiniProgramAppID(ctx, deps, wechatAppRepository)
	if err != nil {
		return err
	}

	deps.Logger.Infow("📱 开始为业务用户生成模拟微信账号",
		"user_count", len(userAliases),
		"wechat_app_internal_id", simulatedWechatAppInternalID,
		"wechat_app_id", wechatAppID)

	for _, userAlias := range userAliases {
		userIDStr := state.Users[userAlias]
		userID, err := parseAuthnUserID(userIDStr)
		if err != nil {
			return fmt.Errorf("parse user id for %s: %w", userAlias, err)
		}

		user, err := userRepository.FindByID(ctx, userID)
		if err != nil {
			return fmt.Errorf("get business user %s(%s): %w", userAlias, userIDStr, err)
		}

		accountAlias := simulatedWechatAccountAlias(userAlias)
		created, credentialCreated, accountID, err := ensureSimulatedWechatMinipAccount(
			ctx,
			deps,
			registerService,
			credentialIssuer,
			accountRepository,
			credentialRepository,
			user,
			userAlias,
			userID,
			wechatAppID,
		)
		if err != nil {
			return fmt.Errorf("ensure simulated wechat authn for %s: %w", userAlias, err)
		}

		state.Accounts[accountAlias] = accountID
		deps.Logger.Infow("✅ 模拟微信账号已收敛",
			"user_alias", userAlias,
			"user_id", userID.Uint64(),
			"account_alias", accountAlias,
			"account_id", accountID,
			"provider", string(accountDomain.ProviderWeChat),
			"app_id", wechatAppID,
			"created", created,
			"credential_created", credentialCreated)
	}

	return nil
}

func collectSimulatedWechatUserAliases(users map[string]string) ([]string, error) {
	aliases := make([]string, 0, len(users))
	for alias, userIDStr := range users {
		userID, err := parseAuthnUserID(userIDStr)
		if err != nil {
			return nil, fmt.Errorf("parse business user id for %s: %w", alias, err)
		}
		if shouldSeedSimulatedWechatAccount(userID.Uint64()) {
			aliases = append(aliases, alias)
		}
	}
	sort.Strings(aliases)
	return aliases, nil
}

func shouldSeedSimulatedWechatAccount(userID uint64) bool {
	return userID > simulatedWechatBusinessUserIDFloor
}

func simulatedWechatAccountAlias(userAlias string) string {
	return userAlias + simulatedWechatAccountAliasSuffix
}

func simulatedWechatIdentity(userID meta.ID) (openID string, unionID string) {
	return fmt.Sprintf("seed-wx-openid-%d", userID.Uint64()),
		fmt.Sprintf("seed-wx-unionid-%d", userID.Uint64())
}

func resolveSimulatedWechatMiniProgramAppID(
	ctx context.Context,
	deps *dependencies,
	wechatAppRepository wechatappDomain.Repository,
) (string, error) {
	if wechatAppRepository != nil {
		app, err := wechatAppRepository.GetByID(ctx, idutil.NewID(simulatedWechatAppInternalID))
		if err != nil {
			return "", fmt.Errorf("get simulated wechat app by internal id %d: %w", simulatedWechatAppInternalID, err)
		}
		if app != nil {
			appID := strings.TrimSpace(app.AppID)
			if appID != "" {
				return appID, nil
			}
		}
	}

	if fallback := configuredSimulatedWechatMiniProgramAppID(deps.Config); fallback != "" {
		deps.Logger.Warnw("⚠️  未找到指定内部微信应用ID，回退使用配置中的测评小程序 app_id",
			"wechat_app_internal_id", simulatedWechatAppInternalID,
			"wechat_app_id", fallback,
			"wechat_app_alias", simulatedWechatAppConfigAlias)
		return fallback, nil
	}

	return "", fmt.Errorf(
		"wechat mini-program internal id %d not found and no configured MiniProgram fallback available",
		simulatedWechatAppInternalID,
	)
}

func configuredSimulatedWechatMiniProgramAppID(config *SeedConfig) string {
	if config == nil {
		return ""
	}

	for _, app := range config.WechatApps {
		if strings.EqualFold(strings.TrimSpace(app.Alias), simulatedWechatAppConfigAlias) &&
			strings.EqualFold(strings.TrimSpace(app.Type), "MiniProgram") {
			return strings.TrimSpace(app.AppID)
		}
	}

	for _, app := range config.WechatApps {
		if strings.EqualFold(strings.TrimSpace(app.Type), "MiniProgram") {
			return strings.TrimSpace(app.AppID)
		}
	}

	return ""
}

func ensureSimulatedWechatMinipAccount(
	ctx context.Context,
	deps *dependencies,
	registerService registerApp.RegisterApplicationService,
	credentialIssuer credentialDomain.Issuer,
	accountRepository *accountRepo.AccountRepository,
	credentialRepository *credentialRepo.Repository,
	user *userDomain.User,
	userAlias string,
	userID meta.ID,
	wechatAppID string,
) (created bool, credentialCreated bool, accountID uint64, err error) {
	openID, unionID := simulatedWechatIdentity(userID)
	externalID := fmt.Sprintf("%s@%s", openID, wechatAppID)

	existing, err := accountRepository.GetByExternalIDAppId(
		ctx,
		accountDomain.ExternalID(externalID),
		accountDomain.AppId(wechatAppID),
	)
	if err != nil {
		return false, false, 0, fmt.Errorf("query wechat account by external id: %w", err)
	}
	if existing != nil {
		if existing.UserID != userID {
			return false, false, 0, fmt.Errorf("wechat account %s belongs to another user %s", externalID, existing.UserID.String())
		}
		if err := ensureSimulatedWechatAccountUniqueID(ctx, accountRepository, existing, unionID); err != nil {
			return false, false, 0, err
		}
		credentialCreated, err := ensureSimulatedWechatCredential(
			ctx,
			credentialIssuer,
			credentialRepository,
			existing.ID,
			wechatAppID,
			openID,
			unionID,
		)
		if err != nil {
			return false, false, 0, err
		}
		return false, credentialCreated, existing.ID.Uint64(), nil
	}

	req := registerApp.RegisterRequest{
		Name:           user.Name,
		Phone:          user.Phone,
		Email:          user.Email,
		ExistingUserID: userID,
		AccountType:    accountDomain.TypeWcMinip,
		CredentialType: registerApp.CredTypeWechat,
		WechatAppID:    stringPtr(wechatAppID),
		WechatOpenID:   stringPtr(openID),
		WechatUnionID:  stringPtr(unionID),
		Meta: map[string]string{
			"seed_provider": string(accountDomain.ProviderWeChat),
			"seed_source":   "seeddata",
			"seed_user":     userAlias,
		},
	}

	result, err := registerService.Register(ctx, req)
	if err != nil {
		if deps.OnConflict == "fail" || !isAuthnConflictError(err) {
			return false, false, 0, err
		}

		existing, lookupErr := accountRepository.GetByExternalIDAppId(
			ctx,
			accountDomain.ExternalID(externalID),
			accountDomain.AppId(wechatAppID),
		)
		if lookupErr != nil {
			return false, false, 0, fmt.Errorf("resolve conflicted wechat account: %w", lookupErr)
		}
		if existing == nil {
			return false, false, 0, err
		}
		if existing.UserID != userID {
			return false, false, 0, fmt.Errorf("wechat account %s belongs to another user %s", externalID, existing.UserID.String())
		}
		if err := ensureSimulatedWechatAccountUniqueID(ctx, accountRepository, existing, unionID); err != nil {
			return false, false, 0, err
		}
		credentialCreated, err := ensureSimulatedWechatCredential(
			ctx,
			credentialIssuer,
			credentialRepository,
			existing.ID,
			wechatAppID,
			openID,
			unionID,
		)
		if err != nil {
			return false, false, 0, err
		}
		return false, credentialCreated, existing.ID.Uint64(), nil
	}

	return result.IsNewAccount, false, result.AccountID.Uint64(), nil
}

func ensureSimulatedWechatAccountUniqueID(
	ctx context.Context,
	accountRepository *accountRepo.AccountRepository,
	existing *accountDomain.Account,
	unionID string,
) error {
	if unionID == "" || existing == nil || existing.UniqueID != "" {
		return nil
	}
	if err := accountRepository.UpdateUniqueID(ctx, existing.ID, accountDomain.UnionID(unionID)); err != nil {
		return fmt.Errorf("update wechat account unique_id: %w", err)
	}
	return nil
}

func ensureSimulatedWechatCredential(
	ctx context.Context,
	credentialIssuer credentialDomain.Issuer,
	credentialRepository *credentialRepo.Repository,
	accountID meta.ID,
	wechatAppID string,
	openID string,
	unionID string,
) (bool, error) {
	existing, err := credentialRepository.GetByAccountIDAndType(ctx, accountID, credentialDomain.CredOAuthWxMinip)
	if err != nil {
		return false, fmt.Errorf("get wechat credential: %w", err)
	}
	if existing != nil {
		return false, nil
	}

	idpIdentifier := unionID
	if idpIdentifier == "" {
		idpIdentifier = openID
	}

	cred, err := credentialIssuer.IssueWechatMinip(ctx, credentialDomain.IssueOAuthRequest{
		AccountID:     accountID,
		IDPIdentifier: idpIdentifier,
		AppID:         wechatAppID,
	})
	if err != nil {
		return false, fmt.Errorf("issue wechat credential: %w", err)
	}

	if err := credentialRepository.Create(ctx, cred); err != nil {
		if isAuthnConflictError(err) {
			return false, nil
		}
		return false, fmt.Errorf("create wechat credential: %w", err)
	}

	return true, nil
}

func stringPtr(s string) *string {
	return &s
}

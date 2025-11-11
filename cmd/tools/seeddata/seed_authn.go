package main

import (
	"context"
	"errors"
	"fmt"

	registerApp "github.com/FangcunMount/iam-contracts/internal/apiserver/application/authn/register"
	authnUOW "github.com/FangcunMount/iam-contracts/internal/apiserver/application/authn/uow"
	accountDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/account"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/infra/crypto"
	userRepo "github.com/FangcunMount/iam-contracts/internal/apiserver/infra/mysql/user"
	wechatInfra "github.com/FangcunMount/iam-contracts/internal/apiserver/infra/wechat"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

// ==================== 认证 Seed 函数 ====================

// seedAuthn 创建认证账号数据
//
// 业务说明：
// 1. 为系统管理员和测试用户创建运营后台账号
// 2. 使用新的 RegisterApplicationService 进行账户注册
// 3. 当前仅支持密码注册方式（operation 账号）
// 4. 返回的 state 保存账号ID，供后续步骤使用
//
// 前置条件：必须先执行 user 步骤创建用户
// 幂等性：Register 服务内部会处理重复注册情况
func seedAuthn(ctx context.Context, deps *dependencies, state *seedContext) error {
	if len(state.Users) == 0 {
		return errors.New("user context is empty; run user step first")
	}

	config := deps.Config
	if config == nil || len(config.Accounts) == 0 {
		deps.Logger.Warnw("⚠️  配置文件中没有账号数据，跳过")
		return nil
	}

	// 初始化基础设施层
	unitOfWork := authnUOW.NewUnitOfWork(deps.DB)
	userRepository := userRepo.NewRepository(deps.DB)

	// 初始化领域服务（密码哈希器）
	// TODO: pepper 应该从配置中读取
	passwordHasher := crypto.NewArgon2Hasher("default-pepper-change-me")

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
		req := registerApp.RegisterRequest{
			Name:           user.Name,
			Phone:          user.Phone,
			Email:          user.Email,
			AccountType:    accountDomain.TypeOpera, // 运营账号类型
			CredentialType: registerApp.CredTypePassword,
			Password:       &ac.Password,
		}

		result, err := registerService.Register(ctx, req)
		if err != nil {
			return fmt.Errorf("register account %s: %w", ac.Alias, err)
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

	deps.Logger.Infow("✅ 认证账号数据已创建")
	return nil
}

// parseAuthnUserID 解析用户ID字符串为 meta.ID
func parseAuthnUserID(userIDStr string) (meta.ID, error) {
	var id uint64
	if _, err := fmt.Sscanf(userIDStr, "%d", &id); err != nil {
		return meta.FromUint64(0), fmt.Errorf("invalid user id format: %s", userIDStr)
	}
	return meta.FromUint64(id), nil
}

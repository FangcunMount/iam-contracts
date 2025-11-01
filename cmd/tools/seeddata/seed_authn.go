package main

import (
	"context"
	"errors"
	"fmt"

	accountApp "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/application/account"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/application/adapter"
	authnUOW "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/application/uow"
	accountDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/authentication"
	userMysql "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/infra/mysql/user"
)

// ==================== 认证相关类型定义 ====================

// ==================== 认证 Seed 函数 ====================

// seedAuthn 创建认证账号数据
//
// 业务说明：
// 1. 为系统管理员和测试用户创建运营后台账号
// 2. 设置账号凭证（用户名和密码）
// 3. 密码使用 bcrypt 哈希存储
// 4. 返回的 state 保存账号ID，供后续步骤使用
//
// 前置条件：必须先执行 user 步骤创建用户
// 幂等性：已存在的账号会跳过创建，直接更新凭证
func seedAuthn(ctx context.Context, deps *dependencies, state *seedContext) error {
	if len(state.Users) == 0 {
		return errors.New("user context is empty; run user step first")
	}

	config := deps.Config
	if config == nil || len(config.Accounts) == 0 {
		deps.Logger.Warnw("⚠️  配置文件中没有账号数据，跳过")
		return nil
	}

	// 初始化仓储和适配器
	userRepo := userMysql.NewRepository(deps.DB)
	userAdapter := adapter.NewUserAdapter(userRepo)
	unitOfWork := authnUOW.NewUnitOfWork(deps.DB)

	// 初始化应用服务
	accountService := accountApp.NewAccountApplicationService(unitOfWork, userAdapter)
	operationService := accountApp.NewOperationAccountApplicationService(unitOfWork)

	// 从配置文件读取账号数据,只处理 operation provider 的账号
	for _, ac := range config.Accounts {
		// 只处理运营账号
		if ac.Provider != "operation" {
			continue
		}

		// 1. 获取用户ID
		userIDStr := state.Users[ac.UserAlias]
		if userIDStr == "" {
			return fmt.Errorf("user alias %s not found for account %s", ac.UserAlias, ac.Alias)
		}

		userID, err := accountDomain.ParseUserID(userIDStr)
		if err != nil {
			return fmt.Errorf("parse user id %s: %w", userIDStr, err)
		}

		// 2. 检查账号是否已存在
		existing, err := accountService.ListAccountsByUserID(ctx, accountDomain.UserID(userID))
		accountExists := false
		if err == nil {
			for _, acc := range existing {
				if acc.Provider == accountDomain.ProviderPassword {
					state.Accounts[ac.Alias] = acc.ID
					accountExists = true
					break
				}
			}
		}

		// 3. 账号不存在，创建新账号
		if !accountExists {
			accountResult, err := accountService.CreateOperationAccount(ctx, accountApp.CreateOperationAccountDTO{
				UserID:   userID,
				Username: ac.Username,
				HashAlgo: string(authentication.AlgorithmBcrypt),
			})
			if err != nil {
				return fmt.Errorf("create operation account %s: %w", ac.Username, err)
			}
			state.Accounts[ac.Alias] = accountResult.Account.ID
		}

		// 4. 更新账号凭证（密码）
		hash, err := authentication.HashPassword(ac.Password, authentication.AlgorithmBcrypt)
		if err != nil {
			return fmt.Errorf("hash password for %s: %w", ac.Username, err)
		}

		if err := operationService.UpdateCredential(ctx, accountApp.UpdateOperationCredentialDTO{
			Username: ac.Username,
			Password: hash.Hash,
			HashAlgo: string(authentication.AlgorithmBcrypt),
		}); err != nil {
			return fmt.Errorf("update credential for %s: %w", ac.Username, err)
		}
	}

	deps.Logger.Infow("✅ 认证账号数据已创建")
	return nil
}

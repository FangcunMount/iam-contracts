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

// operationAccountSeed 运营账号种子数据
type operationAccountSeed struct {
	Alias     string // 别名，对应用户别名
	UserAlias string // 关联的用户别名
	Username  string // 登录用户名
	Password  string // 明文密码
	HashAlgo  string // 密码哈希算法
}

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

	// 初始化仓储和适配器
	userRepo := userMysql.NewRepository(deps.DB)
	userAdapter := adapter.NewUserAdapter(userRepo)
	unitOfWork := authnUOW.NewUnitOfWork(deps.DB)

	// 初始化应用服务
	accountService := accountApp.NewAccountApplicationService(unitOfWork, userAdapter)
	operationService := accountApp.NewOperationAccountApplicationService(unitOfWork)

	// 定义运营账号数据
	passwordSeeds := []operationAccountSeed{
		{
			Alias:     "admin",
			UserAlias: "admin",
			Username:  "admin",
			Password:  "Admin@123",
			HashAlgo:  string(authentication.AlgorithmBcrypt),
		},
		{
			Alias:     "zhangsan",
			UserAlias: "zhangsan",
			Username:  "zhangsan",
			Password:  "Pass@123",
			HashAlgo:  string(authentication.AlgorithmBcrypt),
		},
	}

	for _, seed := range passwordSeeds {
		// 1. 获取用户ID
		userIDStr := state.Users[seed.UserAlias]
		if userIDStr == "" {
			return fmt.Errorf("user alias %s not found", seed.UserAlias)
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
					state.Accounts[seed.Alias] = acc.ID
					accountExists = true
					break
				}
			}
		}

		// 3. 账号不存在，创建新账号
		if !accountExists {
			accountResult, err := accountService.CreateOperationAccount(ctx, accountApp.CreateOperationAccountDTO{
				UserID:   userID,
				Username: seed.Username,
				HashAlgo: seed.HashAlgo,
			})
			if err != nil {
				return fmt.Errorf("create operation account %s: %w", seed.Username, err)
			}
			state.Accounts[seed.Alias] = accountResult.Account.ID
		}

		// 4. 更新账号凭证（密码）
		hash, err := authentication.HashPassword(seed.Password, authentication.AlgorithmBcrypt)
		if err != nil {
			return fmt.Errorf("hash password for %s: %w", seed.Username, err)
		}

		if err := operationService.UpdateCredential(ctx, accountApp.UpdateOperationCredentialDTO{
			Username: seed.Username,
			Password: hash.Hash,
			HashAlgo: seed.HashAlgo,
		}); err != nil {
			return fmt.Errorf("update credential for %s: %w", seed.Username, err)
		}
	}

	deps.Logger.Infow("✅ 认证账号数据已创建", "accounts", len(passwordSeeds))
	return nil
}

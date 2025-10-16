# 应用服务层事务管理分析

## 🔍 当前问题

### 问题描述

在重构后，领域服务不再包含UoW，但应用服务在调用领域服务时遇到了**事务传递问题**：

```go
// 应用服务
func (s *accountApplicationService) CreateOperationAccount(dto) error {
    return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
        // ❌ 问题：领域服务使用的是构造函数注入的仓储
        // 这些仓储不在事务中！
        account, operation, err := s.accountRegisterer.CreateOperationAccount(
            ctx, dto.UserID, dto.Username,
        )
        // ...
    })
}

// 领域服务
type RegisterService struct {
    accounts  drivenPort.AccountRepo  // 这是非事务仓储
    operation drivenPort.OperationRepo
    wechat    drivenPort.WeChatRepo
}
```

**核心矛盾**：
- 应用服务在事务中调用领域服务
- 但领域服务使用的是非事务仓储
- 导致数据可能不在同一事务中

## 📋 解决方案对比

### 方案1：领域服务接受仓储参数 ⭐️ 推荐

**设计**：
```go
// 领域服务接口
type AccountRegisterer interface {
    CreateOperationAccount(
        ctx context.Context,
        repos RepositorySet,  // 传入仓储集合
        userID domain.UserID,
        externalID string,
    ) (*domain.Account, *domain.OperationAccount, error)
}

// 应用服务使用
s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
    repos := RepositorySet{
        Accounts: tx.Accounts,
        Operation: tx.Operation,
    }
    return s.accountRegisterer.CreateOperationAccount(ctx, repos, ...)
})
```

**优点**：
- ✅ 领域服务完全无状态
- ✅ 事务控制完全在应用层
- ✅ 灵活性高

**缺点**：
- ❌ 接口签名变复杂
- ❌ 每次调用都要传仓储

### 方案2：应用服务直接调用仓储 + 领域服务作为工具类

**设计**：
```go
// 领域层提供工厂函数和验证函数
package service

func CreateAccount(...) (*domain.Account, error) {
    // 创建实体，验证业务规则
    return domain.NewAccount(...), nil
}

func ValidateUsername(username string) error {
    // 业务规则验证
}

// 应用服务直接操作仓储
func (s *accountApplicationService) CreateOperationAccount(dto) error {
    return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
        // 使用领域工具函数
        account, err := service.CreateAccount(dto.UserID, dto.Username)
        if err != nil {
            return err
        }
        
        // 直接使用事务仓储
        if err := tx.Accounts.Create(ctx, account); err != nil {
            return err
        }
        
        operation, err := service.CreateOperationAccount(account.ID, dto.Username)
        if err != nil {
            return err
        }
        
        return tx.Operation.Create(ctx, operation)
    })
}
```

**优点**：
- ✅ 事务控制清晰
- ✅ 领域服务简单（纯函数）
- ✅ 性能好（无额外抽象）

**缺点**：
- ❌ 应用服务代码较多
- ❌ 业务规则可能分散

### 方案3：使用Context传递事务 ❌ 不推荐

**设计**：
```go
// 在context中传递事务仓储
type txKey struct{}

func (s *accountApplicationService) CreateOperationAccount(dto) error {
    return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
        // 将tx放入context
        txCtx := context.WithValue(ctx, txKey{}, tx)
        
        // 领域服务从context获取
        return s.accountRegisterer.CreateOperationAccount(txCtx, ...)
    })
}
```

**优点**：
- ✅ 接口签名不变

**缺点**：
- ❌ 隐式依赖，难以测试
- ❌ 违反显式依赖原则
- ❌ 不推荐使用

### 方案4：领域服务懒加载仓储（当前旧代码模式）

**设计**：
```go
// 应用服务有helper函数
func pickAccountRepo(tx uow.TxRepositories, fallback drivenPort.AccountRepo) drivenPort.AccountRepo {
    if tx.Accounts != nil {
        return tx.Accounts
    }
    return fallback
}

// 领域服务接受可选的事务仓储
func (s *RegisterService) CreateOperationAccount(
    ctx context.Context,
    tx uow.TxRepositories,  // 可选的事务仓储
    userID domain.UserID,
    externalID string,
) (*domain.Account, *domain.OperationAccount, error) {
    // 选择使用事务仓储还是默认仓储
    accRepo := pickAccountRepo(tx, s.accounts)
    opRepo := pickOperationRepo(tx, s.operation)
    
    // 使用选中的仓储
    account, err := accRepo.Create(...)
}
```

**优点**：
- ✅ 向后兼容
- ✅ 支持事务和非事务调用

**缺点**：
- ❌ 接口设计不够清晰
- ❌ 可能导致误用

## 🎯 推荐方案

基于DDD最佳实践和代码清晰性，推荐使用**方案2的变体**：

### 最终推荐：应用服务协调 + 领域服务提供业务规则

**原则**：
1. **领域服务**：提供业务规则验证和实体创建的工厂方法（无状态函数）
2. **应用服务**：在事务中协调仓储操作，调用领域服务验证业务规则

**实现**：

```go
// domain/account/service/registerer.go
package service

// 工厂方法：创建Account实体
func CreateAccountEntity(
    userID domain.UserID,
    provider domain.Provider,
    externalID string,
    appID *string,
) (*domain.Account, error) {
    // 验证业务规则
    if userID.IsZero() {
        return nil, errors.New("user id cannot be empty")
    }
    
    // 创建实体
    account := domain.NewAccount(userID, provider, ...)
    return &account, nil
}

// 工厂方法：创建OperationAccount实体
func CreateOperationEntity(
    accountID domain.AccountID,
    username string,
) (*domain.OperationAccount, error) {
    // 验证业务规则
    if username == "" {
        return nil, errors.New("username cannot be empty")
    }
    
    // 创建实体
    operation := domain.NewOperationAccount(accountID, username, ...)
    return operation, nil
}

// 业务规则验证：检查账号是否已存在
func ValidateAccountNotExists(
    ctx context.Context,
    repo drivenPort.AccountRepo,
    provider domain.Provider,
    externalID string,
    appID *string,
) error {
    _, err := repo.FindByRef(ctx, provider, externalID, appID)
    if err == nil {
        return errors.New("account already exists")
    }
    if !errors.Is(err, gorm.ErrRecordNotFound) {
        return err
    }
    return nil
}
```

```go
// application/account/account_app_service.go
func (s *accountApplicationService) CreateOperationAccount(
    ctx context.Context,
    dto CreateOperationAccountDTO,
) (*AccountResult, error) {
    // 1. 验证用户存在（跨聚合）
    exists, err := s.userAdapter.ExistsUser(ctx, dto.UserID)
    if err != nil {
        return nil, err
    }
    if !exists {
        return nil, errors.New("user not found")
    }

    var result *AccountResult

    // 2. 在事务中执行
    err = s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
        // 3. 使用领域服务验证业务规则
        if err := service.ValidateAccountNotExists(
            ctx, tx.Accounts, domain.ProviderPassword, dto.Username, nil,
        ); err != nil {
            return err
        }

        // 4. 使用领域工厂方法创建实体
        account, err := service.CreateAccountEntity(
            dto.UserID, domain.ProviderPassword, dto.Username, nil,
        )
        if err != nil {
            return err
        }

        // 5. 使用事务仓储保存
        if err := tx.Accounts.Create(ctx, account); err != nil {
            return err
        }

        // 6. 创建运营账号实体
        operation, err := service.CreateOperationEntity(account.ID, dto.Username)
        if err != nil {
            return err
        }

        // 7. 保存运营账号
        if err := tx.Operation.Create(ctx, operation); err != nil {
            return err
        }

        // 8. 如果有密码，更新凭据
        if dto.Password != "" {
            if err := tx.Operation.UpdateHash(
                ctx, dto.Username, []byte(dto.Password), dto.HashAlgo, nil,
            ); err != nil {
                return err
            }
        }

        result = &AccountResult{
            Account:       account,
            OperationData: operation,
        }
        return nil
    })

    return result, err
}
```

## 📊 方案对比总结

| 方案 | 事务控制 | 代码清晰度 | 测试难度 | 推荐度 |
|------|----------|------------|----------|--------|
| 方案1: 传参仓储 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ |
| 方案2: 工厂+协调 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| 方案3: Context传递 | ⭐⭐⭐ | ⭐⭐ | ⭐⭐ | ⭐ |
| 方案4: 懒加载 | ⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ |

## 🎯 结论

**当前应用服务代码需要调整**，建议：

1. ✅ **保留当前的领域服务类** - 但改造为提供无状态的工厂方法和验证函数
2. ✅ **应用服务直接在事务中调用仓储** - 清晰的事务边界
3. ✅ **使用领域工厂方法创建实体** - 确保业务规则被执行
4. ✅ **使用领域验证函数验证规则** - 业务逻辑集中在领域层

这样既保证了事务的正确性，又保持了领域逻辑的封装。

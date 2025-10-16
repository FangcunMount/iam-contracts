# DI 容器更新完成报告

## 更新概述

已成功更新 DI 容器配置，将旧的领域服务注册替换为新的应用服务注册。

## 主要更改

### 1. 导入更新 (`auth.go`)

```go
// 添加别名导入以使用新的应用服务
accountApp "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/application/account"
```

### 2. AuthModule 结构体更新

#### 更新前
```go
type AuthModule struct {
	// 旧的应用层服务（实际是领域服务）
	RegisterService *account.RegisterService
	EditorService   *account.EditorService
	QueryService    *account.QueryService
	StatusService   *account.StatusService
	
	// 认证服务
	LoginService *login.LoginService
	TokenService *token.TokenService
	
	// HTTP 处理器
	AccountHandler *authhandler.AccountHandler
	AuthHandler    *authhandler.AuthHandler
}
```

#### 更新后
```go
type AuthModule struct {
	// 新的应用服务（面向用例）
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
```

### 3. Initialize 方法更新

#### 更新前
```go
// ========== 应用层 ==========

// 账户管理服务
accountServicer := account.NewAccountService(accountRepo, wechatRepo, userAdapter, passwordAdapter, unitOfWork)

// 认证服务
m.LoginService = login.NewLoginService(authenticator, tokenIssuer)
m.TokenService = token.NewTokenService(tokenIssuer, tokenRefresher, tokenVerifyer)

// ========== 接口层 ==========

m.AccountHandler = authhandler.NewAccountHandler()

m.AuthHandler = authhandler.NewAuthHandler(
	m.LoginService,
	m.TokenService,
)
```

#### 更新后
```go
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
```

## 依赖注入流程

### 新的依赖链
```
Infrastructure Layer
    ↓
Unit of Work (manages transactions)
    ↓
Application Services (use cases)
    ↓
Handler (HTTP interface)
```

### 服务实例化细节

1. **AccountApplicationService**
   - 依赖: `UnitOfWork`, `UserAdapter`
   - 职责: 账户基本管理用例（创建、查询、启用、禁用）

2. **OperationAccountApplicationService**
   - 依赖: `UnitOfWork`
   - 职责: 操作账户管理用例（更新凭证、修改用户名、重置失败、解锁）

3. **WeChatAccountApplicationService**
   - 依赖: `UnitOfWork`
   - 职责: 微信账户管理用例（绑定、更新资料、设置 UnionID）

4. **AccountLookupApplicationService**
   - 依赖: `UnitOfWork`
   - 职责: 账户查找用例（按提供商查找）

### Handler 依赖注入

Handler 不再依赖领域端口，而是直接依赖应用服务：

```go
func NewAccountHandler(
	accountService AccountApplicationService,
	operationService OperationAccountApplicationService,
	wechatService WeChatAccountApplicationService,
	lookupService AccountLookupApplicationService,
) *AccountHandler
```

## 清理工作

### 已删除的旧应用层文件

以下旧的应用层服务文件已被删除（它们实际上是领域服务的错误放置）：

- ❌ `internal/apiserver/modules/authn/application/account/register.go`
- ❌ `internal/apiserver/modules/authn/application/account/editor.go`
- ❌ `internal/apiserver/modules/authn/application/account/query.go`
- ❌ `internal/apiserver/modules/authn/application/account/status.go`

### 保留的新应用服务文件

应用层现在只包含正确的用例导向服务：

- ✅ `services.go` - 接口和 DTO 定义
- ✅ `account_app_service.go` - 账户应用服务实现
- ✅ `operation_app_service.go` - 操作账户应用服务实现
- ✅ `wechat_app_service.go` - 微信账户应用服务实现
- ✅ `lookup_app_service.go` - 账户查找应用服务实现
- ✅ `util.go` - 工具函数

## 编译验证

```bash
# 编译 DI 容器模块
$ go build -v ./internal/apiserver/container/assembler/...
✅ 成功

# 编译整个 apiserver
$ go build -v ./cmd/apiserver/...
✅ 成功
```

## 架构对比

### 重构前的问题
- ❌ 应用层服务直接实现领域端口
- ❌ 应用层和领域层职责混淆
- ❌ Handler 依赖领域端口（跨层依赖）
- ❌ 事务管理不一致（领域服务持有非事务性仓储）

### 重构后的优势
- ✅ 应用服务面向用例，清晰的业务流程编排
- ✅ 领域服务提供无状态工厂方法和验证函数
- ✅ Handler 依赖应用服务（正确的分层）
- ✅ 事务由应用层统一管理（`UnitOfWork.WithinTx`）
- ✅ 每个应用服务只注入 `UnitOfWork`，在事务中访问仓储
- ✅ 清晰的依赖方向：Interface → Application → Domain → Infrastructure

## 依赖关系图

```
┌─────────────────────────────────────────────────────────────┐
│                     Interface Layer                          │
│  ┌──────────────┐                                           │
│  │AccountHandler│                                            │
│  └──────┬───────┘                                            │
│         │ depends on                                         │
└─────────┼────────────────────────────────────────────────────┘
          │
          ↓
┌─────────┼────────────────────────────────────────────────────┐
│         │           Application Layer                        │
│  ┌──────▼──────────────────────────────────────────┐         │
│  │  Application Services (Use Cases)               │         │
│  │  - AccountApplicationService                    │         │
│  │  - OperationAccountApplicationService           │         │
│  │  - WeChatAccountApplicationService              │         │
│  │  - AccountLookupApplicationService              │         │
│  └──────┬──────────────────────────────────────────┘         │
│         │ uses                                               │
└─────────┼────────────────────────────────────────────────────┘
          │
          ↓
┌─────────┼────────────────────────────────────────────────────┐
│         │              Domain Layer                          │
│  ┌──────▼────────────────────────────────────┐               │
│  │  Domain Services (Factory Methods)        │               │
│  │  - CreateAccountEntity()                  │               │
│  │  - CreateOperationAccountEntity()         │               │
│  │  - CreateWeChatAccountEntity()            │               │
│  │  - ValidateAccountNotExists()             │               │
│  └───────────────────────────────────────────┘               │
│                                                               │
└───────────────────────────────────────────────────────────────┘
          │
          ↓
┌─────────┼────────────────────────────────────────────────────┐
│         │          Infrastructure Layer                      │
│  ┌──────▼─────────────────────┐                              │
│  │  Unit of Work              │                              │
│  │  - WithinTx()              │                              │
│  │  - TxRepositories          │                              │
│  │    - Accounts              │                              │
│  │    - Operation             │                              │
│  │    - WeChats               │                              │
│  └────────────────────────────┘                              │
└───────────────────────────────────────────────────────────────┘
```

## 下一步

DI 容器更新完成，重构工作已全部完成：

- ✅ 架构设计
- ✅ 领域服务重构
- ✅ 应用服务实现
- ✅ Handler 层简化
- ✅ DI 容器更新
- ✅ 旧代码清理

系统现在符合 DDD 六边形架构的最佳实践，各层职责清晰，事务管理正确。

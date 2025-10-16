# 应用服务层重构进度报告

## 已完成工作 ✅

### 1. 架构设计 (services.go)

定义了清晰的应用服务接口和DTO：

```
application/account/
└── services.go
    ├── DTOs (数据传输对象)
    │   ├── CreateOperationAccountDTO
    │   ├── UpdateOperationCredentialDTO
    │   ├── ChangeUsernameDTO
    │   ├── BindWeChatAccountDTO
    │   ├── UpdateWeChatProfileDTO
    │   └── AccountResult
    │
    └── 应用服务接口
        ├── AccountApplicationService
        ├── OperationAccountApplicationService
        ├── WeChatAccountApplicationService
        └── AccountLookupApplicationService
```

**设计原则**：
- DTO独立于HTTP请求，实现关注点分离
- 每个服务接口代表一组相关的业务用例
- 返回值使用聚合结果（AccountResult）

### 2. 领域服务实现 (domain层)

创建了纯粹的领域服务：

**文件**: `domain/account/service/registerer.go`

```go
type RegisterService struct {
    accounts  drivenPort.AccountRepo
    wechat    drivenPort.WeChatRepo
    operation drivenPort.OperationRepo
    // 注意：没有UoW，没有Adapter
}

// 实现 drivingPort.AccountRegisterer
- CreateAccount()
- CreateOperationAccount()
- CreateWeChatAccount()
```

**特点**：
- ✅ 专注业务规则
- ✅ 不包含事务管理
- ✅ 不调用外部系统（用户服务等）
- ✅ 实现 Driving Ports

### 3. 应用服务实现 (application层)

创建了4个应用服务实现：

#### a) AccountApplicationService
**文件**: `account_app_service.go`

```go
type accountApplicationService struct {
    // 领域服务
    accountRegisterer drivingPort.AccountRegisterer
    accountQueryer    drivingPort.AccountQueryer
    accountStatus     drivingPort.AccountStatusUpdater
    accountEditor     drivingPort.AccountEditor

    // 适配器
    userAdapter adapter.UserAdapter

    // 工作单元
    uow uow.UnitOfWork
}
```

**用例**：
- CreateOperationAccount - 创建运营账号（含用户验证、密码设置）
- GetAccountByID - 获取账号详情
- ListAccountsByUserID - 列出用户账号
- EnableAccount / DisableAccount - 启用/禁用账号

#### b) OperationAccountApplicationService
**文件**: `operation_app_service.go`

**用例**：
- UpdateCredential - 更新凭据（自动重置失败次数）
- ChangeUsername - 修改用户名（自动解锁）
- GetByUsername - 根据用户名查询
- ResetFailures / UnlockAccount - 管理账号状态

#### c) WeChatAccountApplicationService
**文件**: `wechat_app_service.go`

**用例**：
- BindWeChatAccount - 绑定微信账号
- UpdateProfile - 更新微信资料
- SetUnionID - 设置UnionID
- GetByWeChatRef - 根据微信引用查询

#### d) AccountLookupApplicationService
**文件**: `lookup_app_service.go`

**用例**：
- FindByProvider - 根据提供商查找账号

### 4. 架构分层

```
┌─────────────────────────────────────┐
│   Interface Layer (Handler)         │  ← 待更新
│   - 依赖 Application Services       │
└──────────────┬──────────────────────┘
               │
               ↓
┌─────────────────────────────────────┐
│   Application Layer                 │  ✅ 已完成
│   ├── services.go (接口+DTO)        │
│   ├── account_app_service.go        │
│   ├── operation_app_service.go      │
│   ├── wechat_app_service.go         │
│   └── lookup_app_service.go         │
│                                      │
│   职责：                             │
│   - 用例编排                         │
│   - 事务管理 (UoW)                  │
│   - 调用外部系统 (UserAdapter)      │
│   - DTO转换                          │
└──────────────┬──────────────────────┘
               │
               ↓
┌─────────────────────────────────────┐
│   Domain Layer (Services)           │  ✅ 已完成
│   └── service/                      │
│       ├── registerer.go (NEW!)      │
│       ├── editor.go                 │
│       ├── query.go                  │
│       └── status.go                 │
│                                      │
│   职责：                             │
│   - 实现 Driving Ports              │
│   - 封装业务规则                     │
│   - 操作聚合根和实体                 │
└──────────────┬──────────────────────┘
               │
               ↓
┌─────────────────────────────────────┐
│   Domain Layer (Ports)              │
│   ├── driving/                      │
│   │   └── service.go                │
│   └── driven/                       │
│       └── repo.go                   │
└─────────────────────────────────────┘
```

## 待完成工作 📋

### 1. 更新Interface层 (Handler)

**当前状态**：
```go
type AccountHandler struct {
    register drivingPort.AccountRegisterer  // ❌ 直接依赖领域端口
    editor   drivingPort.AccountEditor
    status   drivingPort.AccountStatusUpdater
    query    drivingPort.AccountQueryer
}
```

**目标状态**：
```go
type AccountHandler struct {
    accountApp   account.AccountApplicationService        // ✅ 依赖应用服务
    operationApp account.OperationAccountApplicationService
    wechatApp    account.WeChatAccountApplicationService
    lookupApp    account.AccountLookupApplicationService
}
```

**需要修改的Handler方法**：
- CreateOperationAccount - 使用 accountApp.CreateOperationAccount()
- UpdateOperationCredential - 使用 operationApp.UpdateCredential()
- ChangeOperationUsername - 使用 operationApp.ChangeUsername()
- BindWeChatAccount - 使用 wechatApp.BindWeChatAccount()
- UpsertWeChatProfile - 使用 wechatApp.UpdateProfile()
- SetWeChatUnionID - 使用 wechatApp.SetUnionID()
- GetAccount - 使用 accountApp.GetAccountByID()
- EnableAccount / DisableAccount - 使用 accountApp
- ListAccountsByUser - 使用 accountApp.ListAccountsByUserID()
- FindAccountByRef - 使用 lookupApp.FindByProvider()
- GetOperationAccountByUsername - 使用 operationApp.GetByUsername()

### 2. 更新DI容器配置

需要在容器中注册：

**领域服务**：
```go
// domain/account/service/registerer.go
RegisterService // 实现 drivingPort.AccountRegisterer
```

**应用服务**：
```go
// application/account/
accountApplicationService
operationAccountApplicationService
wechatAccountApplicationService
accountLookupApplicationService
```

### 3. 清理旧代码

可以删除或重构的文件：
- `application/account/register.go` (已被 registerer.go 和 account_app_service.go 替代)
- `application/account/editor.go` (如果domain层的editor已实现)
- `application/account/query.go` (如果domain层的query已实现)
- `application/account/status.go` (如果domain层的status已实现)

**注意**：先确认domain层服务完整实现后再删除

## 关键改进点 🎯

### 1. 职责清晰

**Before**:
```go
// application/account/register.go
func (s *RegisterService) CreateOperationAccount(...) {
    // 验证用户存在 (应用层职责) ✅
    exists, err := s.userAdapter.ExistsUser(ctx, userID)
    
    // 在事务中执行 (应用层职责) ✅
    s.uow.WithinTx(ctx, func(tx) {
        // 创建账号 (领域层职责) ✅
        // 保存到数据库 (领域层职责) ✅
    })
}

// 问题：应用层服务实现了领域端口，职责混乱
var _ drivingPort.AccountRegisterer = (*RegisterService)(nil)
```

**After**:
```go
// domain/account/service/registerer.go (领域服务)
type RegisterService struct {
    accounts  drivenPort.AccountRepo
    // 只依赖仓储，不依赖UoW或Adapter
}

func (s *RegisterService) CreateOperationAccount(...) {
    // 纯业务逻辑
    // 创建账号
    // 保存到仓储
}

var _ drivingPort.AccountRegisterer = (*RegisterService)(nil)  // ✅ 正确

// application/account/account_app_service.go (应用服务)
type accountApplicationService struct {
    accountRegisterer drivingPort.AccountRegisterer  // 依赖领域服务
    userAdapter       adapter.UserAdapter
    uow               uow.UnitOfWork
}

func (s *accountApplicationService) CreateOperationAccount(dto) {
    // 验证用户存在
    s.userAdapter.ExistsUser(...)
    
    // 在事务中执行
    s.uow.WithinTx(ctx, func(tx) {
        // 调用领域服务
        s.accountRegisterer.CreateOperationAccount(...)
        
        // 设置密码等应用层逻辑
        s.accountEditor.UpdateOperationCredential(...)
    })
}
```

### 2. 依赖方向正确

```
Handler 
  → Application Service (面向用例)
    → Domain Service (面向业务规则)
      → Repository (面向数据)
```

### 3. DTO隔离

**Before**: Handler方法参数混乱
```go
func (h *Handler) CreateOperationAccount(c *gin.Context) {
    var req CreateOperationAccountReq  // HTTP请求对象
    h.register.CreateOperationAccount(ctx, userID, username)  // 多次调用
    h.editor.UpdateOperationCredential(ctx, username, hash, ...)
}
```

**After**: 清晰的DTO
```go
func (h *Handler) CreateOperationAccount(c *gin.Context) {
    var req CreateOperationAccountReq
    
    dto := account.CreateOperationAccountDTO{  // 转换为DTO
        UserID:   parseUserID(req.UserID),
        Username: req.Username,
        Password: req.Password,
        HashAlgo: req.HashAlgo,
    }
    
    result, err := h.accountApp.CreateOperationAccount(ctx, dto)  // 一次调用
}
```

## 下一步计划 📝

1. **更新Handler** (高优先级)
   - 修改AccountHandler依赖
   - 重写所有Handler方法使用应用服务
   - 测试API功能

2. **更新容器配置** (高优先级)
   - 注册领域服务
   - 注册应用服务
   - 配置依赖注入

3. **完善领域服务** (中优先级)
   - 检查editor.go是否需要去除UoW
   - 检查query.go是否需要去除UoW
   - 检查status.go是否需要去除UoW

4. **清理旧代码** (低优先级)
   - 删除application层的旧实现
   - 更新文档

## 编译状态 ✅

所有新代码已通过编译：
```bash
✅ go build ./internal/apiserver/modules/authn/domain/account/service/...
✅ go build ./internal/apiserver/modules/authn/application/account/...
```

## 总结

重构完成了**核心架构调整**，实现了：
- ✅ 清晰的分层架构
- ✅ 正确的依赖方向
- ✅ 单一职责原则
- ✅ DTO隔离
- ✅ 用例导向的应用服务

现在可以继续更新Handler和容器配置，完成整个重构。

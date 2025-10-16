# 应用服务层架构设计

## 1. 分层职责划分

### Interface Layer (接口层)
- **职责**: HTTP请求处理、参数验证、响应格式化
- **依赖**: Application Services
- **不应该**: 包含业务逻辑、直接调用Domain Services

### Application Layer (应用层)
- **职责**: 
  - 用例编排（Use Case Orchestration）
  - 协调多个领域服务
  - 管理事务边界
  - DTO转换（API请求 → 领域对象）
  - 处理跨聚合的业务流程
- **依赖**: Domain Services (通过Driving Ports)
- **不应该**: 包含业务规则、直接操作仓储

### Domain Layer (领域层)
- **职责**:
  - 核心业务规则
  - 领域逻辑封装
  - 聚合根不变性维护
  - 领域事件发布
- **依赖**: Driven Ports (仓储接口)
- **不应该**: 关心HTTP、事务、DTO转换

## 2. 当前架构问题

### 问题1: 应用层实现领域端口
```go
// ❌ 错误：application/account/register.go
type RegisterService struct { ... }
var _ drivingPort.AccountRegisterer = (*RegisterService)(nil)
```
**问题**: RegisterService 位于 application 层却实现了 domain 层的 driving port

### 问题2: Handler直接依赖领域端口
```go
// ❌ 错误：interface/handler/account.go
type AccountHandler struct {
    register drivingPort.AccountRegisterer  // 直接依赖领域端口
}
```
**问题**: Handler 应该依赖 Application Services，而不是 Domain Services

### 问题3: 职责混乱
```go
// ❌ 当前：应用层既做流程编排，又实现领域接口
func (s *RegisterService) CreateOperationAccount(...)  // 既是应用服务又是领域服务？
```

## 3. 正确的架构设计

### 3.1 分层依赖关系
```
┌─────────────────────────────────────┐
│   Interface Layer (Handler)         │
│   - HTTP请求处理                     │
│   - 参数验证和格式化                 │
└──────────────┬──────────────────────┘
               │ 依赖
               ↓
┌─────────────────────────────────────┐
│   Application Layer                 │
│   - AccountApplicationService       │
│   - 用例编排                         │
│   - 事务管理                         │
│   - DTO转换                          │
└──────────────┬──────────────────────┘
               │ 依赖
               ↓
┌─────────────────────────────────────┐
│   Domain Layer (Services)           │
│   - AccountDomainService            │
│   - 实现 Driving Ports              │
│   - 包含业务规则                     │
└──────────────┬──────────────────────┘
               │ 依赖
               ↓
┌─────────────────────────────────────┐
│   Domain Layer (Ports)              │
│   - AccountRegisterer (driving)     │
│   - AccountRepo (driven)            │
└─────────────────────────────────────┘
```

### 3.2 应用服务接口设计

#### 原则
1. **面向用例**: 每个方法代表一个完整的业务用例
2. **DTO隔离**: 使用应用层DTO，不直接暴露HTTP请求对象
3. **完整编排**: 包含完整的业务流程，不需要Handler多次调用
4. **事务边界**: 应用服务方法即事务边界

#### 示例：创建运营账号用例
```go
// DTO定义 - 应用层的数据传输对象
type CreateOperationAccountDTO struct {
    UserID   domain.UserID
    Username string
    Password string
    HashAlgo string
}

// 应用服务接口
type AccountApplicationService interface {
    // CreateOperationAccount 创建运营账号用例
    // 完整流程：
    // 1. 创建账号聚合根
    // 2. 创建运营账号子实体
    // 3. 设置密码哈希
    // 4. 保存到数据库（事务）
    CreateOperationAccount(ctx context.Context, dto CreateOperationAccountDTO) (*AccountResult, error)
}
```

#### Handler调用示例
```go
// ✅ 正确：Handler只负责HTTP处理
func (h *AccountHandler) CreateOperationAccount(c *gin.Context) {
    var req CreateOperationAccountReq
    if err := h.BindJSON(c, &req); err != nil {
        h.Error(c, err)
        return
    }

    // 转换为DTO
    dto := account.CreateOperationAccountDTO{
        UserID:   parseUserID(req.UserID),
        Username: req.Username,
        Password: req.Password,
        HashAlgo: req.HashAlgo,
    }

    // 调用应用服务（一次调用完成整个用例）
    result, err := h.accountAppService.CreateOperationAccount(c.Request.Context(), dto)
    if err != nil {
        h.Error(c, err)
        return
    }

    h.Created(c, toResponse(result))
}
```

### 3.3 领域服务接口设计

领域服务实现 Driving Ports，专注于业务规则：

```go
// domain/account/port/driving/service.go
type AccountRegisterer interface {
    // CreateAccount 创建账号（纯业务逻辑）
    CreateAccount(ctx context.Context, userID domain.UserID, externalID string) (*domain.Account, error)
    
    // CreateOperationAccount 创建运营账号（纯业务逻辑）
    CreateOperationAccount(ctx context.Context, userID domain.UserID, externalID string) (*domain.Account, *domain.OperationAccount, error)
}

// domain/account/service/register.go
type RegisterDomainService struct {
    accounts  drivenPort.AccountRepo
    operation drivenPort.OperationRepo
}

var _ drivingPort.AccountRegisterer = (*RegisterDomainService)(nil)

// 实现业务规则
func (s *RegisterDomainService) CreateOperationAccount(ctx context.Context, userID domain.UserID, externalID string) (*domain.Account, *domain.OperationAccount, error) {
    // 纯业务逻辑：
    // 1. 验证业务规则
    // 2. 创建聚合根
    // 3. 调用仓储保存
    // 不关心：事务、密码哈希、HTTP请求
}
```

### 3.4 应用服务实现

应用服务编排领域服务，处理用例流程：

```go
// application/account/account_app_service.go
type accountApplicationService struct {
    // 依赖领域服务（通过driving ports）
    accountRegisterer drivingPort.AccountRegisterer
    accountEditor     drivingPort.AccountEditor
    accountQueryer    drivingPort.AccountQueryer
    
    // 依赖适配器
    passwordAdapter   adapter.PasswordAdapter
    userAdapter       adapter.UserAdapter
    
    // 依赖工作单元
    uow uow.UnitOfWork
}

func (s *accountApplicationService) CreateOperationAccount(ctx context.Context, dto CreateOperationAccountDTO) (*AccountResult, error) {
    // 应用层负责：
    // 1. 验证用户存在（跨聚合）
    // 2. 哈希密码（技术细节）
    // 3. 调用领域服务创建账号
    // 4. 设置密码
    // 5. 事务管理
    
    return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) (*AccountResult, error) {
        // 1. 验证用户存在
        if _, err := s.userAdapter.GetUser(ctx, dto.UserID); err != nil {
            return nil, err
        }
        
        // 2. 调用领域服务创建账号
        account, operation, err := s.accountRegisterer.CreateOperationAccount(ctx, dto.UserID, dto.Username)
        if err != nil {
            return nil, err
        }
        
        // 3. 哈希密码并更新
        hash, err := s.passwordAdapter.Hash(dto.Password, dto.HashAlgo)
        if err != nil {
            return nil, err
        }
        
        if err := s.accountEditor.UpdateOperationCredential(ctx, dto.Username, hash, dto.HashAlgo, nil); err != nil {
            return nil, err
        }
        
        return &AccountResult{
            Account:       account,
            OperationData: operation,
        }, nil
    })
}
```

## 4. 重构步骤

### Step 1: 定义应用服务接口和DTO
- [x] 创建 `application/account/services.go`
- [x] 定义应用服务接口
- [x] 定义应用层DTO

### Step 2: 将现有Application层代码移到Domain层
- [ ] `application/account/register.go` → `domain/account/service/register.go`
- [ ] `application/account/editor.go` → `domain/account/service/editor.go`
- [ ] `application/account/query.go` → `domain/account/service/query.go`
- [ ] `application/account/status.go` → `domain/account/service/status.go`

### Step 3: 实现应用服务
- [ ] 创建 `application/account/account_app_service.go`
- [ ] 创建 `application/account/operation_app_service.go`
- [ ] 创建 `application/account/wechat_app_service.go`
- [ ] 创建 `application/account/lookup_app_service.go`

### Step 4: 更新Handler依赖
- [ ] 修改 `interface/handler/account.go` 依赖应用服务
- [ ] 删除对领域端口的直接依赖

### Step 5: 更新容器配置
- [ ] 修改DI容器，注册应用服务
- [ ] 确保领域服务正确注入到应用服务

## 5. 核心设计原则

### 单一职责
- **Interface Layer**: HTTP协议处理
- **Application Layer**: 用例编排
- **Domain Layer**: 业务规则

### 依赖倒置
- Handler 依赖 Application Service 接口
- Application Service 依赖 Domain Service 接口（Driving Ports）
- Domain Service 依赖 Repository 接口（Driven Ports）

### 关注点分离
- DTO在应用层，与HTTP解耦
- 业务规则在领域层，与事务/技术细节解耦
- HTTP细节在接口层，与业务逻辑解耦

## 6. 收益

1. **清晰的分层**: 每层职责明确，易于理解和维护
2. **可测试性**: 各层可独立测试
3. **可扩展性**: 添加新用例不影响领域层
4. **技术独立**: 领域层不依赖HTTP、数据库等技术细节
5. **复用性**: 领域服务可被多个应用服务复用

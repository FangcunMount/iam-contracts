# 凭据颁发器 (Credential Issuer) 架构设计

## 概述

凭据颁发器（Issuer）是 credential 领域的核心服务，负责为账户颁发各种类型的登录凭据。它将凭据创建（Binder）和持久化（Repository）封装成统一的领域服务，供应用层调用。

## 架构分层

```
应用层 (Application Layer)
    ├── RegisterApplicationService
    └── 调用 → credDomain.Issuer

领域层 (Domain Layer)
    └── credential 领域
        ├── Issuer (凭据颁发器) ← 对外接口
        │   ├── IssuePassword()
        │   ├── IssuePhoneOTP()
        │   ├── IssueWechatMinip()
        │   └── IssueWecom()
        │
        ├── Binder (凭据绑定器) ← 内部使用
        │   └── Bind() - 创建凭据实体
        │
        └── Repository (凭据仓储) ← 内部使用
            └── Create() - 持久化凭据

基础设施层 (Infrastructure Layer)
    ├── PasswordHasher (密码哈希)
    └── Database (数据库持久化)
```

## 核心接口

### Issuer 接口

```go
type Issuer interface {
    // IssuePassword 颁发密码凭据
    IssuePassword(ctx context.Context, req IssuePasswordRequest) (*Credential, error)
    
    // IssuePhoneOTP 颁发手机OTP凭据
    IssuePhoneOTP(ctx context.Context, req IssuePhoneOTPRequest) (*Credential, error)
    
    // IssueWechatMinip 颁发微信小程序凭据
    IssueWechatMinip(ctx context.Context, req IssueOAuthRequest) (*Credential, error)
    
    // IssueWecom 颁发企业微信凭据
    IssueWecom(ctx context.Context, req IssueOAuthRequest) (*Credential, error)
}
```

### 颁发请求 DTOs

```go
// 密码凭据颁发请求
type IssuePasswordRequest struct {
    AccountID      meta.ID
    PlainPassword  string  // 明文密码
    HashedPassword string  // 已哈希密码（可选）
    Algo           string  // 哈希算法
}

// 手机OTP凭据颁发请求
type IssuePhoneOTPRequest struct {
    AccountID meta.ID
    Phone     meta.Phone
}

// OAuth凭据颁发请求（微信、企微等）
type IssueOAuthRequest struct {
    AccountID     meta.ID
    IDP           string  // 身份提供商（有默认值）
    IDPIdentifier string  // OpenID/UnionID/UserID
    AppID         string
    ParamsJSON    []byte  // 第三方返回JSON
}
```

## 职责划分

### Issuer vs Binder

| 特性 | Issuer | Binder |
|------|--------|--------|
| **层次** | 领域服务（面向应用层） | 领域对象工厂（内部） |
| **职责** | 颁发凭据（创建+持久化） | 创建凭据实体 |
| **依赖** | Repository + Binder + PasswordHasher | 无外部依赖 |
| **使用方** | 应用层 | Issuer（内部使用） |
| **状态** | 有状态（持有依赖） | 无状态 |

### AccountCreator vs Issuer

| 特性 | AccountCreator | Issuer |
|------|----------------|--------|
| **领域** | account 领域 | credential 领域 |
| **职责** | 账户创建（基于 AccountType） | 凭据颁发（基于 CredentialType） |
| **策略选择** | 根据 AccountType 选策略 | 根据凭据类型选方法 |
| **第三方调用** | 可能调用（如微信 code2session） | 不调用第三方 |
| **返回数据** | Account + CreationParams | Credential |

## 使用示例

### 应用层调用

```go
// 在应用服务中使用 Issuer
func (s *registerApplicationService) Register(ctx context.Context, req RegisterRequest) (*RegisterResult, error) {
    err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
        // 1. 创建用户
        user, _, err := s.createOrGetUser(ctx, req)
        
        // 2. 创建账户（使用 AccountCreator）
        accountCreator := domain.NewAccountCreator(tx.Accounts, strategies)
        account, params, err := accountCreator.CreateAccount(ctx, input)
        
        // 3. 颁发凭据（使用 Issuer）
        credIssuer := credDomain.NewIssuer(tx.Credentials, s.hasher)
        credential, err := s.issueCredential(ctx, credIssuer, account.ID, params, req)
        
        return nil
    })
    
    return result, err
}

// 根据凭据类型颁发凭据
func (s *registerApplicationService) issueCredential(
    ctx context.Context,
    issuer credDomain.Issuer,
    accountID meta.ID,
    params *domain.CreationParams,
    req RegisterRequest,
) (*credDomain.Credential, error) {
    switch req.CredentialType {
    case CredTypePassword:
        return issuer.IssuePassword(ctx, credDomain.IssuePasswordRequest{
            AccountID:     accountID,
            PlainPassword: *req.Password,
        })
        
    case CredTypeWechat:
        idpIdentifier := params.OpenID
        if params.UnionID != "" {
            idpIdentifier = params.UnionID
        }
        return issuer.IssueWechatMinip(ctx, credDomain.IssueOAuthRequest{
            AccountID:     accountID,
            IDPIdentifier: idpIdentifier,
            AppID:         *req.WechatAppID,
        })
    }
}
```

## 设计优势

### 1. 关注点分离

- **account 领域**：专注账户创建，处理账户类型差异
- **credential 领域**：专注凭据颁发，处理凭据类型差异

### 2. 单一职责

- `AccountCreator`：负责账户创建流程（可能涉及第三方调用）
- `Issuer`：负责凭据颁发流程（本地操作）
- `Binder`：负责凭据实体创建（纯领域逻辑）

### 3. 接口隔离

- Issuer 提供细粒度方法（IssuePassword、IssuePhoneOTP 等）
- 每个方法接收专用的请求 DTO
- 避免"万能方法"带来的参数混乱

### 4. 依赖反转

```
应用层 → Issuer (接口)
           ↑
领域层 → issuer (实现)
           ↓
基础设施层 → Repository, PasswordHasher
```

### 5. 可测试性

```go
// 单元测试 Issuer
func TestIssuer_IssuePassword(t *testing.T) {
    mockRepo := &MockRepository{}
    mockHasher := &MockHasher{}
    issuer := NewIssuer(mockRepo, mockHasher)
    
    cred, err := issuer.IssuePassword(ctx, IssuePasswordRequest{
        AccountID:     accountID,
        PlainPassword: "password123",
    })
    
    assert.NoError(t, err)
    assert.NotNil(t, cred)
}
```

## 迁移指南

### 从 CredentialBinder 迁移到 Issuer

#### 旧代码（使用 CredentialBinder）

```go
// 在 account 领域中
credBinder := domain.NewCredentialBinder(tx.Credentials, s.hasher)
credential, err := credBinder.BindPassword(ctx, accountID, password)
```

#### 新代码（使用 Issuer）

```go
// 在 credential 领域中
credIssuer := credDomain.NewIssuer(tx.Credentials, s.hasher)
credential, err := credIssuer.IssuePassword(ctx, credDomain.IssuePasswordRequest{
    AccountID:     accountID,
    PlainPassword: password,
})
```

### 关键变化

1. **位置变化**：从 `account` 领域移到 `credential` 领域
2. **命名变化**：`CredentialBinder` → `Issuer`
3. **方法变化**：`BindPassword()` → `IssuePassword()`
4. **参数变化**：直接参数 → 结构化请求 DTO

## 文件结构

```
internal/apiserver/domain/authn/credential/
├── issuer.go              # Issuer 接口和实现
├── binder.go              # Binder 实现（内部使用）
├── interfaces.go          # 领域接口定义
├── repository.go          # Repository 接口
└── types.go              # 类型定义

internal/apiserver/domain/authn/account/
├── creator.go            # AccountCreator 实现
├── creator_opera.go      # Opera 账户策略
├── creator_wechat_mini.go # 微信小程序账户策略
├── creator_wecom.go      # 企微账户策略
└── interfaces.go         # 账户领域接口
```

## 总结

通过引入 Issuer，我们实现了：

1. ✅ **领域边界清晰**：账户创建和凭据颁发分属不同领域
2. ✅ **职责单一**：每个组件只做一件事
3. ✅ **接口隔离**：细粒度的颁发方法
4. ✅ **依赖明确**：Issuer 协调 Binder + Repository
5. ✅ **易于扩展**：新增凭据类型只需添加新方法
6. ✅ **便于测试**：可独立测试每个组件

这种设计符合 DDD 的战术设计原则，为未来的扩展和维护奠定了良好基础。

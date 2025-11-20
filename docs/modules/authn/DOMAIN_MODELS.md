# 认证中心 - 领域模型设计

> [返回认证中心文档](./README.md)

本文档详细介绍认证中心的领域模型设计，包括聚合根、实体、值对象和领域服务，深入阐述每个模型的职责和实现的领域知识。

---

## 目录

1. [领域概述](#1-领域概述)
2. [领域模型总览](#2-领域模型总览)
3. [Account 聚合根](#3-account-聚合根)
4. [Credential 聚合根](#4-credential-聚合根)
5. [Authentication 领域服务](#5-authentication-领域服务)
6. [Token 领域服务](#6-token-领域服务)
7. [JWKS 聚合根](#7-jwks-聚合根)
8. [值对象](#8-值对象)
9. [仓储接口](#9-仓储接口)

---

## 1. 领域概述

认证中心（Authn）领域负责用户身份验证、凭据管理、Token 签发和 JWKS 公钥发布，是整个 IAM 系统的安全基石。

### 1.1 核心领域概念

- **Account（账户）**：用户的认证身份，一个用户可以有多个账户（微信、企业微信、运营后台等）
- **Credential（凭据）**：用于验证身份的凭据（密码哈希、OAuth token、手机号等）
- **Authentication（认证）**：验证用户身份的过程
- **Token（令牌）**：认证成功后签发的访问凭证（Access Token / Refresh Token）
- **JWKS（密钥集）**：用于签名和验证 JWT 的公钥集合

### 1.2 领域边界

**本领域负责**：

- ✅ 多渠道用户认证（微信、企业微信、密码、手机OTP）
- ✅ 凭据生命周期管理（创建、验证、禁用、删除）
- ✅ JWT Token 签发、刷新、撤销
- ✅ JWKS 公钥发布和密钥轮换
- ✅ 账户状态管理（激活、禁用、锁定）

**本领域不负责**：

- ❌ 用户基本信息管理（由 UC 模块负责）
- ❌ 权限控制（由 Authz 模块负责）
- ❌ 外部 IDP 具体实现（由 IDP 模块负责）
- ❌ 业务级审计日志（由业务模块负责）

---

## 2. 领域模型总览

### 2.1 聚合根设计

```text
┌─────────────────────────────────────────────────────────────┐
│                  Authn Domain Model                          │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────────┐         ┌──────────────────┐         │
│  │  Account (聚合根) │         │ Credential (聚合根)│         │
│  ├──────────────────┤         ├──────────────────┤         │
│  │ + ID             │   1:N   │ + ID             │         │
│  │ + UserID         │◄────────┤ + AccountID      │         │
│  │ + Type           │         │ + Type           │         │
│  │ + ExternalID     │         │ + Material       │         │
│  │ + Status         │         │ + Status         │         │
│  └──────────────────┘         │ + FailedAttempts │         │
│                                └──────────────────┘         │
│           │                                                  │
│           │ 认证                                             │
│           ▼                                                  │
│  ┌──────────────────────────────────────────────┐          │
│  │    Authentication (领域服务)                  │          │
│  │  ┌───────────────────────────────────────┐  │          │
│  │  │ PasswordAuthStrategy                  │  │          │
│  │  │ WeChatAuthStrategy                    │  │          │
│  │  │ WeComAuthStrategy                     │  │          │
│  │  │ JWTTokenAuthStrategy                  │  │          │
│  │  └───────────────────────────────────────┘  │          │
│  │  Authenticater (策略模式)                    │          │
│  └──────────────────────────────────────────────┘          │
│           │                                                  │
│           │ 签发                                             │
│           ▼                                                  │
│  ┌──────────────────────────────────────────────┐          │
│  │       Token (领域服务)                        │          │
│  │  - TokenIssuer (签发器)                       │          │
│  │  - TokenRefresher (刷新器)                    │          │
│  │  - TokenVerifyer (验证器)                     │          │
│  └──────────────────────────────────────────────┘          │
│           │                                                  │
│           │ 使用                                             │
│           ▼                                                  │
│  ┌──────────────────┐                                       │
│  │  JWKS (聚合根)    │                                       │
│  ├──────────────────┤                                       │
│  │ + Keys[]         │                                       │
│  │ + KeyManager     │                                       │
│  │ + KeyRotation    │                                       │
│  └──────────────────┘                                       │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 聚合根职责划分

| 聚合根/领域服务 | 核心职责 | 不变性维护 |
|----------------|---------|-----------|
| **Account** | 账户身份管理、状态生命周期 | 账户与用户关联、外部ID唯一性 |
| **Credential** | 凭据存储和验证、失败次数追踪 | 凭据与账户绑定、类型约束 |
| **Authentication** | 多策略认证、认证决策 | 认证流程一致性、策略隔离 |
| **Token** | JWT 签发、刷新、撤销 | Token 有效性、黑名单机制 |
| **JWKS** | 密钥生命周期、公钥发布 | 密钥状态转换、轮换规则 |

---

## 3. Account 聚合根

### 3.1 领域概念

**Account（账户）** 是用户在特定认证渠道的身份表示。一个用户（User）可以拥有多个账户，对应不同的登录方式（微信小程序、企业微信、运营后台等）。

### 3.2 聚合根定义

```go
// internal/apiserver/domain/authn/account/account.go
package account

type Account struct {
    ID         meta.ID              // 账户ID（唯一标识）
    UserID     meta.ID              // 关联的用户ID（UC模块）
    Type       AccountType          // 账户类型（值对象）
    AppID      AppId                // 应用ID（值对象）
    ExternalID ExternalID           // 外部平台标识（值对象）
    UniqueID   UnionID              // 全局唯一标识（值对象）
    Profile    map[string]string    // 用户资料（昵称、头像等）
    Meta       map[string]string    // 额外元数据
    Status     AccountStatus        // 账户状态
}

// 工厂方法
func NewAccount(userID meta.ID, accountType AccountType, 
    externalID ExternalID, opts ...AccountOption) *Account

// 领域方法 - 状态管理
func (a *Account) Activate()        // 激活账户
func (a *Account) Disable()         // 禁用账户
func (a *Account) Lock()            // 锁定账户
func (a *Account) Unlock()          // 解锁账户

// 领域方法 - 信息更新
func (a *Account) UpdateProfile(profile map[string]string)
func (a *Account) UpdateMeta(meta map[string]string)

// 领域方法 - 状态查询
func (a *Account) IsActive() bool   // 是否激活
func (a *Account) IsLocked() bool   // 是否锁定
func (a *Account) IsDisabled() bool // 是否禁用
```

### 3.3 领域职责

| 职责类别 | 具体职责 | 实现的领域知识 |
|---------|---------|---------------|
| **身份绑定** | 关联用户与外部身份 | 一个用户可以有多个账户，但每个账户只属于一个用户 |
| **状态管理** | 账户生命周期控制 | 禁用/锁定状态下无法登录 |
| **信息维护** | 保存外部平台资料 | 缓存昵称、头像等，减少外部API调用 |
| **类型区分** | 区分不同认证渠道 | 微信、企业微信、运营后台等有不同的验证逻辑 |

### 3.4 账户类型（AccountType）

```go
type AccountType string

const (
    TypeWeChatMiniProgram  AccountType = "wc-minip"  // 微信小程序
    TypeWeChatOfficial     AccountType = "wc-offi"   // 微信公众号
    TypeWeCom              AccountType = "wc-com"    // 企业微信
    TypeOperation          AccountType = "opera"     // 运营后台
)
```

**类型说明**：

- **wc-minip**：微信小程序账户，使用 OpenID/UnionID
- **wc-offi**：微信公众号账户，使用 OpenID/UnionID
- **wc-com**：企业微信账户，使用 UserID
- **opera**：运营后台账户，使用用户名/密码

### 3.5 业务不变性

Account 聚合根维护以下业务不变性：

1. **关联约束**
   - ✅ 每个账户必须关联到一个有效的用户
   - ✅ ExternalID 在同一 AppID + Type 下必须唯一
   - ✅ 账户 ID 全局唯一

2. **状态约束**
   - ✅ 禁用（Disabled）状态下无法登录
   - ✅ 锁定（Locked）状态下无法登录（通常由失败次数触发）
   - ✅ 只有激活（Active）状态才能正常使用

3. **数据完整性**
   - ✅ Type、ExternalID 不能为空
   - ✅ AppID 根据类型可能为空（operation类型无需AppID）

### 3.6 状态转换图

```text
     创建
      │
      ▼
   [Active] ◄────── Activate()
      │
      ├──► [Disabled] ──── Activate() ───┘
      │         │
      │         └──► [Archived] (归档)
      │
      └──► [Locked] ───── Unlock() ──────┘
            (失败次数过多)
```

---

## 4. Credential 聚合根

### 4.1 领域概念

**Credential（凭据）** 是用于验证用户身份的安全信息。不同类型的账户使用不同类型的凭据。

### 4.2 聚合根定义

```go
// internal/apiserver/domain/authn/credential/credential.go
package credential

type Credential struct {
    ID        meta.ID       // 凭据ID
    AccountID meta.ID       // 关联的账户ID
    
    // OAuth/Phone 凭据使用
    IDP           *string   // 身份提供商："wechat"|"wecom"|"phone"
    IDPIdentifier string    // 外部标识：unionid/openid/phone
    AppID         *string   // 应用ID：wechat=appid | wecom=corpid
    
    // Password 凭据使用
    Material   []byte       // PHC 哈希（密码）
    Algo       *string      // 算法："argon2id"|"bcrypt"
    ParamsJSON []byte       // 算法参数
    
    // 安全控制
    Status         CredentialStatus  // 凭据状态
    FailedAttempts int32             // 失败尝试次数
    LockedUntil    *time.Time        // 锁定截止时间
    LastChangedAt  time.Time         // 最后修改时间
}

// 工厂方法
func NewPasswordCredential(accountID meta.ID, material []byte, 
    algo string) *Credential
func NewOAuthCredential(accountID meta.ID, idp, identifier, 
    appID string, params []byte) *Credential

// 领域方法 - 验证
func (c *Credential) VerifyPassword(plaintext string, 
    hasher PasswordHasher) error
func (c *Credential) RecordFailedAttempt() error
func (c *Credential) ResetFailedAttempts()

// 领域方法 - 状态管理
func (c *Credential) Enable()
func (c *Credential) Disable()
func (c *Credential) Lock(duration time.Duration)
func (c *Credential) Unlock()

// 领域方法 - 状态查询
func (c *Credential) IsEnabled() bool
func (c *Credential) IsLocked() bool
```

### 4.3 领域职责

| 职责类别 | 具体职责 | 实现的领域知识 |
|---------|---------|---------------|
| **凭据存储** | 安全存储认证材料 | 密码使用 PHC 格式哈希，OAuth使用外部标识 |
| **验证** | 验证用户提供的凭据 | 密码验证、失败次数追踪 |
| **安全防护** | 防暴力破解 | N次失败后自动锁定 |
| **生命周期** | 凭据的启用/禁用/锁定 | 禁用的凭据无法使用 |

### 4.4 凭据类型

```go
type CredentialType string

const (
    TypePassword CredentialType = "password"  // 密码凭据
    TypeOAuth    CredentialType = "oauth"     // OAuth凭据
    TypePhone    CredentialType = "phone"     // 手机号凭据
)
```

### 4.5 业务不变性

1. **唯一性约束**
   - ✅ 一个账户可以有多个凭据（密码+微信）
   - ✅ 同一账户+凭据类型只能有一个有效凭据
   - ✅ IDP + AppID + IDPIdentifier 组合唯一（OAuth凭据）

2. **安全约束**
   - ✅ 密码必须哈希存储，不能明文
   - ✅ 失败次数达到阈值（如5次）自动锁定
   - ✅ 锁定期间无法进行验证

3. **状态约束**
   - ✅ 禁用的凭据不能用于认证
   - ✅ 锁定的凭据需要等待解锁或管理员手动解锁

### 4.6 密码哈希格式（PHC）

```text
$算法$参数$盐$哈希值

示例：
$argon2id$v=19$m=65536,t=3,p=4$c2FsdDEyMzQ$hashedvalue...
```

**PHC 格式优势**：
- 自包含：算法、参数、盐都在字符串中
- 可升级：支持算法升级而不影响旧密码
- 标准化：遵循 Password Hashing Competition 标准

---

## 5. Authentication 领域服务

### 5.1 领域概念

**Authentication（认证）** 是验证用户身份的过程。认证中心使用**策略模式**支持多种认证方式，每种方式有独立的认证策略。

### 5.2 认证器架构

```go
// internal/apiserver/domain/authn/authentication/authenticater.go
package authentication

// Authenticater 认证器（策略管理器）
type Authenticater struct {
    credRepo      CredentialRepository
    accountRepo   AccountRepository
    hasher        PasswordHasher
    otpVerifier   OTPVerifier
    idp           IdentityProvider
    tokenVerifier TokenVerifier
}

// Authenticate 统一认证入口
// 流程：
// 1. 根据场景构建认证凭据
// 2. 选择对应的认证策略
// 3. 执行认证
func (a *Authenticater) Authenticate(ctx context.Context, 
    scenario Scenario, input AuthInput) (AuthDecision, error)
```

### 5.3 认证策略

#### 5.3.1 策略接口

```go
// AuthStrategy 认证策略接口
type AuthStrategy interface {
    Kind() Scenario
    Authenticate(ctx context.Context, credential AuthCredential) (AuthDecision, error)
}
```

#### 5.3.2 支持的认证场景

```go
type Scenario string

const (
    AuthPassword   Scenario = "password"     // 用户名+密码
    AuthPhoneOTP   Scenario = "phone_otp"    // 手机号+验证码
    AuthWxMiniProg Scenario = "wx_minip"     // 微信小程序
    AuthWeCom      Scenario = "wecom"        // 企业微信
    AuthJWTToken   Scenario = "jwt_token"    // JWT Token验证
)
```

#### 5.3.3 密码认证策略

```go
// PasswordAuthStrategy 密码认证策略
type PasswordAuthStrategy struct {
    credRepo    CredentialRepository
    accountRepo AccountRepository
    hasher      PasswordHasher
}

// Authenticate 执行密码认证
// 认证流程：
// 1. 根据用户名查找账户
// 2. 检查账户状态
// 3. 查找密码凭据
// 4. 验证密码
// 5. 处理失败次数
// 6. 返回认证判决
func (p *PasswordAuthStrategy) Authenticate(ctx context.Context, 
    credential AuthCredential) (AuthDecision, error)
```

**实现的领域知识**：

1. **账户查找**：支持用户名、手机号、邮箱登录
2. **状态检查**：禁用/锁定账户拒绝登录
3. **密码验证**：使用 bcrypt/argon2 验证
4. **失败追踪**：记录失败次数，达到阈值锁定
5. **成功重置**：登录成功重置失败次数

#### 5.3.4 微信小程序认证策略

```go
// WeChatMiniProgramAuthStrategy 微信小程序认证策略
type WeChatMiniProgramAuthStrategy struct {
    credRepo CredentialRepository
    idp      IdentityProvider
}

// Authenticate 执行微信认证
// 认证流程：
// 1. code2Session 换取 OpenID/UnionID
// 2. 根据 OpenID/UnionID 查找凭据绑定
// 3. 检查账户状态
// 4. 返回认证判决
func (w *WeChatMiniProgramAuthStrategy) Authenticate(ctx context.Context, 
    credential AuthCredential) (AuthDecision, error)
```

**实现的领域知识**：

1. **code2Session**：调用微信API验证jsCode
2. **UnionID优先**：优先使用UnionID查找（跨应用身份）
3. **自动创建**：首次登录时可能需要创建账户和凭据
4. **会话管理**：保存session_key用于后续操作

#### 5.3.5 JWT Token 认证策略

```go
// JWTTokenAuthStrategy JWT Token 认证策略
type JWTTokenAuthStrategy struct {
    accountRepo   AccountRepository
    tokenVerifier TokenVerifier
}

// Authenticate 执行 JWT Token 认证
// 认证流程：
// 1. 验证 JWT Token（签名、过期、黑名单）
// 2. 从 Token 中提取用户ID、账户ID
// 3. 检查账户状态（是否锁定/禁用）
// 4. 返回认证判决
func (j *JWTTokenAuthStrategy) Authenticate(ctx context.Context, 
    credential AuthCredential) (AuthDecision, error)
```

**实现的领域知识**：

1. **Token解析**：验证JWT签名和有效期
2. **黑名单检查**：检查Token是否被撤销
3. **账户验证**：确保账户仍然有效
4. **Claims提取**：提取用户身份信息

### 5.4 认证决策

```go
// AuthDecision 认证判决
type AuthDecision struct {
    OK           bool        // 认证是否成功
    Principal    *Principal  // 认证主体（成功时）
    CredentialID meta.ID     // 使用的凭据ID
    ErrCode      ErrCode     // 错误码（失败时）
}

// Principal 认证主体
type Principal struct {
    UserID    meta.ID       // 用户ID
    AccountID meta.ID       // 账户ID
    TenantID  meta.ID       // 租户ID（可选）
    AMR       []string      // Authentication Methods References
    AuthTime  time.Time     // 认证时间
    Claims    map[string]any // 额外声明
}
```

### 5.5 认证流程图

```text
┌──────────────────────────────────────────────────────────┐
│                     认证流程                              │
└──────────────────────────────────────────────────────────┘
           │
           ▼
    [构建认证凭据]
    根据场景(password/wx_minip/wecom...)
    构建对应的 AuthCredential
           │
           ▼
    [选择认证策略]
    根据场景选择对应的 AuthStrategy
    (PasswordAuthStrategy/WeChatAuthStrategy...)
           │
           ▼
    [执行认证]
    调用 strategy.Authenticate()
           │
           ├──► [失败] ──► 返回错误码
           │                - invalid_credential
           │                - account_locked
           │                - account_disabled
           ▼
         [成功]
    返回 AuthDecision
           │
           ▼
    [签发 Token]
    使用 Principal 签发 JWT Token
```

---

## 6. Token 领域服务

### 6.1 领域概念

**Token（令牌）** 是认证成功后签发的访问凭证。系统使用 JWT（JSON Web Token）作为 Access Token，使用 UUID 作为 Refresh Token。

### 6.2 Token 类型

```go
// TokenPair 令牌对
type TokenPair struct {
    AccessToken  *AccessToken  // 访问令牌（JWT）
    RefreshToken *RefreshToken // 刷新令牌（UUID）
}

// AccessToken 访问令牌（JWT）
type AccessToken struct {
    Value     string        // JWT字符串
    ExpiresIn time.Duration // 有效期（如15分钟）
}

// RefreshToken 刷新令牌（UUID）
type RefreshToken struct {
    ID        string    // Token ID
    Value     string    // Token值（UUID）
    UserID    meta.ID   // 用户ID
    AccountID meta.ID   // 账户ID
    ExpiresAt time.Time // 过期时间
}
```

### 6.3 TokenIssuer（令牌签发器）

```go
// TokenIssuer 令牌颁发者
type TokenIssuer struct {
    tokenGenerator TokenGenerator // JWT 生成器
    tokenStore     TokenStore     // 令牌存储（Redis）
    accessTTL      time.Duration  // 访问令牌有效期（15分钟）
    refreshTTL     time.Duration  // 刷新令牌有效期（7天）
}

// IssueToken 颁发令牌对
// 流程：
// 1. 生成 Access Token（JWT）
// 2. 生成 Refresh Token（UUID）
// 3. 保存 Refresh Token 到 Redis
// 4. 返回令牌对
func (s *TokenIssuer) IssueToken(ctx context.Context, 
    principal *authentication.Principal) (*TokenPair, error)

// RevokeToken 撤销令牌
// 流程：
// 1. 解析 Token 获取 tokenID 和过期时间
// 2. 将 Token 加入黑名单（Redis）
// 3. TTL 设置为剩余有效期
func (s *TokenIssuer) RevokeToken(ctx context.Context, 
    tokenValue string) error
```

### 6.4 TokenRefresher（令牌刷新器）

```go
// TokenRefresher 令牌刷新器
type TokenRefresher struct {
    tokenGenerator TokenGenerator
    tokenStore     TokenStore
    accessTTL      time.Duration
    refreshTTL     time.Duration
}

// RefreshToken 刷新令牌
// 流程：
// 1. 验证 Refresh Token（是否存在、是否过期）
// 2. 从 Refresh Token 中提取用户信息
// 3. 签发新的 Access Token
// 4. 更新 Refresh Token 的最后使用时间
func (r *TokenRefresher) RefreshToken(ctx context.Context, 
    refreshTokenValue string) (*TokenPair, error)
```

### 6.5 TokenVerifyer（令牌验证器）

```go
// TokenVerifyer 令牌验证器
type TokenVerifyer struct {
    tokenGenerator TokenGenerator
    tokenStore     TokenStore
}

// VerifyAccessToken 验证访问令牌
// 流程：
// 1. 解析 JWT（验证签名、过期时间）
// 2. 检查黑名单
// 3. 提取用户信息
func (v *TokenVerifyer) VerifyAccessToken(ctx context.Context, 
    tokenValue string) (userID, accountID, tenantID meta.ID, err error)
```

### 6.6 Token 生命周期

```text
┌──────────────────────────────────────────────────────┐
│                 Token 生命周期                        │
└──────────────────────────────────────────────────────┘
    
    [登录成功]
         │
         ▼
    [签发 TokenPair]
    - Access Token (JWT, 15分钟)
    - Refresh Token (UUID, 7天)
         │
         ├──► Access Token ──► [使用] ──► [过期]
         │                         │
         │                         └──► [刷新] ◄─┐
         │                                       │
         └──► Refresh Token ──► [刷新] ─────────┘
                  │
                  ├──► [撤销] ──► [加入黑名单]
                  │
                  └──► [过期] ──► [自动清理]
```

### 6.7 实现的领域知识

| 领域知识 | 实现方式 | 目的 |
|---------|---------|------|
| **短期 Access Token** | 15分钟有效期 | 减少泄露风险 |
| **长期 Refresh Token** | 7天有效期 | 提升用户体验 |
| **黑名单机制** | Redis存储被撤销的Token ID | 支持强制登出 |
| **JWT自验证** | RS256签名 | 业务服务本地验证，无需调用认证中心 |
| **Token轮换** | 刷新时可选是否轮换Refresh Token | 提高安全性 |

---

## 7. JWKS 聚合根

### 7.1 领域概念

**JWKS（JSON Web Key Set）** 是用于签名和验证 JWT 的密钥集合。系统定期轮换密钥，支持多密钥并存。

### 7.2 聚合根定义

```go
// internal/apiserver/domain/authn/jwks/key.go
package jwks

type Key struct {
    ID        meta.ID      // 密钥ID
    KID       string       // Key ID（RFC 7517）
    Status    KeyStatus    // 密钥状态
    Kty       string       // Key Type: "RSA"
    Use       string       // 用途: "sig"
    Alg       string       // 算法: "RS256"
    JWKJSON   []byte       // 公钥 JWK JSON
    NotBefore *time.Time   // 生效时间
    NotAfter  *time.Time   // 过期时间
}

// KeyStatus 密钥状态
type KeyStatus string

const (
    KeyActive  KeyStatus = "active"  // 活跃（用于签名和验证）
    KeyGrace   KeyStatus = "grace"   // 宽限期（仅用于验证）
    KeyRetired KeyStatus = "retired" // 退役（不再使用）
)
```

### 7.3 KeyManager（密钥管理器）

```go
// KeyManager 密钥生命周期管理服务
type KeyManager struct {
    keyRepo      Repository
    keyGenerator KeyGenerator
}

// CreateKey 创建新密钥
func (s *KeyManager) CreateKey(ctx context.Context, 
    alg string, notBefore, notAfter *time.Time) (*Key, error)

// ActivateKey 激活密钥
func (s *KeyManager) ActivateKey(ctx context.Context, 
    keyID meta.ID) error

// RetireKey 退役密钥
func (s *KeyManager) RetireKey(ctx context.Context, 
    keyID meta.ID) error

// GetActiveKeys 获取活跃密钥
func (s *KeyManager) GetActiveKeys(ctx context.Context) ([]*Key, error)
```

### 7.4 KeyRotation（密钥轮换）

```go
// KeyRotation 密钥轮换服务
type KeyRotation struct {
    keyRepo    Repository
    keyManager *KeyManager
}

// RotateKey 执行密钥轮换
// 流程：
// 1. 创建新密钥
// 2. 激活新密钥
// 3. 将旧密钥转为宽限期状态
// 4. 定期清理退役密钥
func (r *KeyRotation) RotateKey(ctx context.Context) error
```

### 7.5 KeySetBuilder（密钥集构建器）

```go
// KeySetBuilder 构建 JWKS 响应
type KeySetBuilder struct {
    keyRepo Repository
}

// BuildJWKS 构建公钥集
// 返回：包含所有活跃和宽限期密钥的 JWKS
func (b *KeySetBuilder) BuildJWKS(ctx context.Context) (*JWKSet, error)
```

### 7.6 密钥生命周期

```text
┌──────────────────────────────────────────────────────┐
│                 密钥生命周期                          │
└──────────────────────────────────────────────────────┘

     [创建]
       │
       ▼
    [Active] ────────────────┐
   (用于签名和验证)            │ 轮换
       │                     │
       │ 30天后              ▼
       ▼                [新密钥Active]
    [Grace]                  │
   (仅用于验证)               │
       │                     │
       │ 7天后               │
       ▼                     │
    [Retired] ◄──────────────┘
   (不再使用)
       │
       │ 清理
       ▼
    [删除]
```

### 7.7 实现的领域知识

| 领域知识 | 实现方式 | 目的 |
|---------|---------|------|
| **多密钥并存** | Active + Grace 状态 | 密钥轮换期间不影响服务 |
| **定期轮换** | 30天轮换一次 | 提高安全性 |
| **宽限期** | 7天Grace Period | 给业务服务缓存更新时间 |
| **RS256算法** | RSA 2048位非对称加密 | 公钥验证，私钥签名 |
| **JWKS标准** | RFC 7517 | 标准化公钥发布 |

---

## 8. 值对象

### 8.1 值对象定义

```go
// AccountType 账户类型
type AccountType string
const (
    TypeWeChatMiniProgram AccountType = "wc-minip"
    TypeWeChatOfficial    AccountType = "wc-offi"
    TypeWeCom             AccountType = "wc-com"
    TypeOperation         AccountType = "opera"
)

// ExternalID 外部平台标识
type ExternalID string
func (e ExternalID) String() string
func (e ExternalID) IsEmpty() bool

// UnionID 全局唯一标识
type UnionID string
func (u UnionID) String() string
func (u UnionID) IsEmpty() bool

// AppId 应用ID
type AppId string
func (a AppId) String() string
func (a AppId) IsEmpty() bool

// CredentialStatus 凭据状态
type CredentialStatus string
const (
    CredStatusEnabled  CredentialStatus = "enabled"
    CredStatusDisabled CredentialStatus = "disabled"
    CredStatusLocked   CredentialStatus = "locked"
)
```

### 8.2 值对象特性

| 特性 | 说明 | 实现的领域知识 |
|-----|------|---------------|
| **不可变性** | 创建后不可修改 | 保证线程安全和语义清晰 |
| **值相等性** | 通过值而非引用比较 | 两个 ExternalID 值相同即相等 |
| **自包含验证** | 验证逻辑封装在内 | 无效的值对象无法被创建 |
| **领域语义** | 表达领域概念 | AccountType 比 string 更具语义 |

---

## 9. 仓储接口

### 9.1 Account 仓储

```go
// Repository 账户仓储接口
type Repository interface {
    Create(ctx context.Context, account *Account) error
    FindByID(ctx context.Context, id meta.ID) (*Account, error)
    FindByExternalID(ctx context.Context, accountType AccountType, 
        appID AppId, externalID ExternalID) (*Account, error)
    FindByUserID(ctx context.Context, userID meta.ID) ([]*Account, error)
    Update(ctx context.Context, account *Account) error
}
```

### 9.2 Credential 仓储

```go
// Repository 凭据仓储接口
type Repository interface {
    Create(ctx context.Context, credential *Credential) error
    FindByID(ctx context.Context, id meta.ID) (*Credential, error)
    FindByAccountID(ctx context.Context, accountID meta.ID) ([]*Credential, error)
    FindPasswordCredential(ctx context.Context, 
        accountID meta.ID) (*Credential, error)
    FindOAuthCredential(ctx context.Context, idp, appID, 
        identifier string) (*Credential, error)
    Update(ctx context.Context, credential *Credential) error
}
```

### 9.3 JWKS 仓储

```go
// Repository 密钥仓储接口
type Repository interface {
    Create(ctx context.Context, key *Key) error
    FindByID(ctx context.Context, id meta.ID) (*Key, error)
    FindByKID(ctx context.Context, kid string) (*Key, error)
    FindByStatus(ctx context.Context, status KeyStatus) ([]*Key, error)
    FindActiveKeys(ctx context.Context) ([]*Key, error)
    Update(ctx context.Context, key *Key) error
    Delete(ctx context.Context, id meta.ID) error
}
```

---

## 10. 总结

### 10.1 聚合根职责总结

| 聚合根/领域服务 | 核心职责 | 关键领域知识 |
|----------------|---------|------------|
| **Account** | 账户身份管理、状态生命周期 | 多账户支持、外部ID唯一性、状态控制 |
| **Credential** | 凭据存储和验证、安全防护 | PHC哈希、失败次数追踪、自动锁定 |
| **Authentication** | 多策略认证、认证决策 | 策略模式、认证流程、Principal构建 |
| **Token** | JWT签发、刷新、撤销 | 双Token机制、黑名单、自验证 |
| **JWKS** | 密钥生命周期、公钥发布 | 密钥轮换、多密钥并存、JWKS标准 |

### 10.2 设计亮点

1. **策略模式**：灵活支持多种认证方式，易于扩展
2. **安全第一**：PHC哈希、失败次数追踪、Token黑名单
3. **标准化**：JWT + JWKS 标准，易于集成
4. **高性能**：JWT本地验证，减少认证中心压力
5. **用户体验**：双Token机制，长期免登录

---

**最后更新**: 2025-11-20  
**维护团队**: Authn Team

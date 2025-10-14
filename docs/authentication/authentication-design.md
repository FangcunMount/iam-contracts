# 认证模块设计

## 1. 领域模型

### 1.1 核心概念

#### Credential (凭证)

- **职责**: 表示用户提供的身份证明信息
- **类型**:
  - `UsernamePasswordCredential`: 用户名密码凭证
  - `WeChatCodeCredential`: 微信授权码凭证
  - `TokenCredential`: Token 凭证（用于验证和刷新）

#### Authentication (认证结果)

- **职责**: 表示一次认证的结果
- **属性**:
  - `UserID`: 认证成功的用户 ID
  - `AccountID`: 使用的账号 ID
  - `AuthenticatedAt`: 认证时间
  - `Metadata`: 认证元数据（如 IP、设备信息等）

#### Token (令牌)

- **职责**: 表示访问令牌
- **类型**:
  - `AccessToken`: 访问令牌（短期有效）
  - `RefreshToken`: 刷新令牌（长期有效）
- **属性**:
  - `TokenID`: 令牌唯一标识
  - `UserID`: 关联的用户 ID
  - `Value`: 令牌值（JWT 字符串）
  - `ExpiresAt`: 过期时间
  - `IssuedAt`: 颁发时间

### 1.2 领域服务

#### AuthenticationService (认证服务)

- **职责**: 执行认证逻辑
- **方法**:
  - `Authenticate(ctx, credential Credential) (*Authentication, error)`: 执行认证
  - **依赖**:
    - `Authenticator` 接口（策略模式）
    - `AccountRepository`: 查询账号信息

#### TokenService (令牌服务)

- **职责**: 令牌的颁发、验证、刷新、销毁
- **方法**:
  - `IssueToken(ctx, authentication *Authentication) (*TokenPair, error)`: 颁发令牌对
  - `VerifyAccessToken(ctx, tokenValue string) (*TokenClaims, error)`: 验证访问令牌
  - `RefreshToken(ctx, refreshTokenValue string) (*TokenPair, error)`: 刷新令牌
  - `RevokeToken(ctx, tokenID string) error`: 销毁令牌
- **依赖**:
  - `TokenGenerator`: JWT 生成器
  - `TokenStore`: Token 存储（Redis）

### 1.3 策略模式：Authenticator

```go
type Authenticator interface {
    // Supports 判断是否支持该凭证类型
    Supports(credential Credential) bool
    
    // Authenticate 执行认证
    Authenticate(ctx context.Context, credential Credential) (*Authentication, error)
}
```

**实现类**:

- `BasicAuthenticator`: 基础认证（用户名密码）
- `WeChatAuthenticator`: 微信 OAuth 认证
- `BearerAuthenticator`: Bearer Token 认证（验证现有 Token）

## 2. 应用服务

### 2.1 AuthenticationApplicationService

- **职责**: 协调认证和令牌颁发流程
- **用例**:
  - `LoginWithPassword(ctx, username, password) (*TokenPair, error)`: 用户名密码登录
  - `LoginWithWeChat(ctx, code, appID) (*TokenPair, error)`: 微信登录
  - `VerifyToken(ctx, accessToken) (*UserInfo, error)`: 验证令牌
  - `RefreshAccessToken(ctx, refreshToken) (*TokenPair, error)`: 刷新访问令牌
  - `Logout(ctx, tokenID) error`: 登出

### 2.2 流程设计

#### 登录流程（以密码登录为例）

```text
1. 接收用户名和密码
2. 创建 UsernamePasswordCredential
3. 调用 AuthenticationService.Authenticate()
   3.1 选择合适的 Authenticator (BasicAuthenticator)
   3.2 验证凭证（查询账号、验证密码哈希）
   3.3 返回 Authentication 结果
4. 调用 TokenService.IssueToken()
   4.1 生成 AccessToken（JWT，15分钟有效）
   4.2 生成 RefreshToken（UUID，7天有效）
   4.3 将 RefreshToken 存储到 Redis
5. 返回 TokenPair

```

#### 验证令牌流程

```text
1. 接收 AccessToken
2. 调用 TokenService.VerifyAccessToken()
   2.1 解析 JWT
   2.2 验证签名
   2.3 检查过期时间
   2.4 （可选）检查是否被撤销（黑名单）
3. 返回 TokenClaims（包含 UserID）
```

#### 刷新令牌流程

```text
1. 接收 RefreshToken
2. 调用 TokenService.RefreshToken()
   2.1 从 Redis 查询 RefreshToken
   2.2 验证有效性（未过期、未撤销）
   2.3 颁发新的 TokenPair
   2.4 （可选）轮换 RefreshToken（删除旧的，存储新的）
3. 返回新的 TokenPair
```

## 3. 端口定义

### 3.1 Domain Ports

#### AccountPasswordPort

```go
type AccountPasswordPort interface {
    // GetPasswordHash 获取账号的密码哈希
    GetPasswordHash(ctx context.Context, accountID AccountID) (*PasswordHash, error)
}
```

#### WeChatAuthPort

```go
type WeChatAuthPort interface {
    // ExchangeOpenID 通过微信 code 换取 openID
    ExchangeOpenID(ctx context.Context, code, appID string) (openID string, err error)
}
```

### 3.2 Infrastructure Ports

#### TokenStore

```go
type TokenStore interface {
    // SaveRefreshToken 保存刷新令牌
    SaveRefreshToken(ctx context.Context, token *RefreshToken) error
    
    // GetRefreshToken 获取刷新令牌
    GetRefreshToken(ctx context.Context, tokenValue string) (*RefreshToken, error)
    
    // DeleteRefreshToken 删除刷新令牌
    DeleteRefreshToken(ctx context.Context, tokenValue string) error
    
    // AddToBlacklist 将令牌加入黑名单
    AddToBlacklist(ctx context.Context, tokenID string, expiry time.Duration) error
    
    // IsBlacklisted 检查令牌是否在黑名单中
    IsBlacklisted(ctx context.Context, tokenID string) (bool, error)
}
```

## 4. 基础设施实现

### 4.1 JWT Token Generator

- 使用 `github.com/golang-jwt/jwt/v5`
- AccessToken: JWT，包含 UserID、AccountID、过期时间
- RefreshToken: 随机 UUID

### 4.2 Redis Token Store

- RefreshToken 存储: `refresh_token:{token_value}` -> `{user_id, account_id, expires_at}`
- Token 黑名单: `token_blacklist:{token_id}` -> `1`（带 TTL）

## 5. 目录结构

```text
internal/apiserver/modules/authn/
├── domain/
│   ├── account/              # 已完成
│   └── authentication/       # 新增
│       ├── credential.go     # 凭证值对象
│       ├── authentication.go # 认证结果实体
│       ├── token.go          # 令牌值对象
│       ├── password.go       # 密码哈希值对象
│       └── port/
│           ├── authenticator.go   # 认证器接口
│           ├── account_password.go # 账号密码端口
│           ├── wechat_auth.go     # 微信认证端口
│           └── token_store.go     # 令牌存储端口
├── application/
│   ├── account/              # 已完成
│   ├── authentication/       # 新增
│   │   ├── login.go          # 登录应用服务
│   │   ├── token.go          # 令牌应用服务
│   │   └── dto/              # DTO 对象
│   └── adapter/              # 已完成
├── infrastructure/
│   ├── mysql/
│   │   └── account/          # 已完成
│   ├── redis/                # 新增
│   │   └── token/
│   │       └── store.go      # Redis Token Store 实现
│   └── jwt/                  # 新增
│       └── generator.go      # JWT 生成器
└── interface/
    └── restful/
        └── handler/
            ├── account.go    # 已完成
            └── auth.go       # 新增：认证 Handler
```

## 6. 实现计划

### Phase 1: 领域模型

1. 创建凭证值对象（Credential）
2. 创建认证结果实体（Authentication）
3. 创建令牌值对象（Token、TokenPair）
4. 创建密码哈希值对象（PasswordHash）
5. 定义端口接口

### Phase 2: 领域服务

1. 实现 BasicAuthenticator
2. 实现 WeChatAuthenticator
3. 实现 AuthenticationService（策略模式）
4. 实现 TokenService

### Phase 3: 基础设施

1. 实现 JWT Generator
2. 实现 Redis Token Store
3. 实现 AccountPasswordAdapter（查询密码哈希）

### Phase 4: 应用服务

1. 实现 LoginApplicationService
2. 实现 TokenApplicationService

### Phase 5: 接口层

1. 实现 AuthHandler（登录、登出、刷新令牌）
2. 定义 RESTful API

## 7. 关键设计决策

### 7.1 为什么使用策略模式？

- **扩展性**: 未来可以轻松添加新的认证方式（如指纹、人脸识别等）
- **单一职责**: 每个 Authenticator 只负责一种认证方式
- **开闭原则**: 新增认证方式不需要修改现有代码

### 7.2 为什么分离 Authentication 和 Token？

- **领域分离**: 认证（识别身份）和授权（颁发凭证）是两个不同的关注点
- **灵活性**: 可以在认证成功后决定是否颁发令牌（如需要二次验证）

### 7.3 为什么 RefreshToken 使用 Redis？

- **性能**: 快速验证和撤销
- **过期管理**: Redis TTL 自动清理过期令牌
- **分布式**: 支持多实例部署

### 7.4 Token 黑名单策略

- **使用场景**: 用户主动登出、密码修改、账号被封禁
- **实现**: 将 TokenID 加入 Redis 黑名单，TTL 设置为 Token 剩余有效期
- **性能优化**: 只在必要时检查黑名单（如敏感操作）

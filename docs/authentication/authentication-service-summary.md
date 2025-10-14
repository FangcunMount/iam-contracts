# 认证模块领域服务实现总结

## ✅ 已完成工作

### Phase 1: 领域模型（已完成）

#### 1. 凭证值对象 (`credential.go`)

- ✅ `Credential` 接口：定义凭证通用行为
- ✅ `UsernamePasswordCredential`：用户名密码凭证
- ✅ `WeChatCodeCredential`：微信授权码凭证
- ✅ `TokenCredential`：Bearer Token 凭证

#### 2. 密码哈希值对象 (`password.go`)

- ✅ `PasswordHash`：封装密码哈希和算法
- ✅ 支持 Bcrypt 算法
- ✅ `Verify()` 方法：安全验证密码
- ✅ `HashPassword()` 工厂方法

#### 3. 认证结果实体 (`authentication.go`)

- ✅ `Authentication`：表示一次成功的认证
- ✅ 包含：UserID、AccountID、Provider、认证时间、元数据

#### 4. 令牌值对象 (`token.go`)

- ✅ `Token`：访问令牌/刷新令牌
- ✅ `TokenPair`：令牌对
- ✅ `TokenClaims`：JWT 声明信息

#### 5. 端口接口 (`port/`)

- ✅ `Authenticator`：认证器策略接口
- ✅ `AccountPasswordPort`：账号密码查询端口
- ✅ `WeChatAuthPort`：微信认证端口
- ✅ `TokenStore`：令牌存储端口（Redis）
- ✅ `TokenGenerator`：JWT 生成器端口

### Phase 2: 领域服务（已完成）

#### 1. BasicAuthenticator (`basic_authenticator.go`)

**职责**: 用户名密码认证

**流程**:

1. 验证凭证格式
2. 根据用户名查找 OperationAccount
3. 获取对应的 Account
4. 检查账号状态（是否激活）
5. 获取密码哈希
6. 验证密码
7. 返回 Authentication

**依赖**:

- `AccountRepo`：查询账号
- `OperationRepo`：查询运营账号
- `AccountPasswordPort`：获取密码哈希

#### 2. WeChatAuthenticator (`wechat_authenticator.go`)

**职责**: 微信 OAuth 认证

**流程**:

1. 验证凭证格式
2. 通过微信 code 换取 openID
3. 根据 openID 查找 WeChatAccount
4. 获取对应的 Account
5. 检查账号状态
6. 返回 Authentication

**依赖**:

- `AccountRepo`：查询账号
- `WeChatRepo`：查询微信账号
- `WeChatAuthPort`：调用微信 API

#### 3. AuthenticationService (`authentication_service.go`)

**职责**: 认证服务编排器（策略模式）

**流程**:

1. 验证凭证
2. 根据凭证类型选择合适的 Authenticator
3. 执行认证
4. 返回 Authentication

**特性**:

- 支持动态注册 Authenticator
- 自动选择合适的认证策略
- 易于扩展新的认证方式

#### 4. TokenService (`token_service.go`)

**职责**: 令牌管理服务

**方法**:

1. **IssueToken** - 颁发令牌对
   - 生成 JWT 访问令牌（默认 15 分钟）
   - 生成 UUID 刷新令牌（默认 7 天）
   - 保存刷新令牌到 Redis

2. **VerifyAccessToken** - 验证访问令牌
   - 解析 JWT
   - 检查过期时间
   - 检查黑名单

3. **RefreshToken** - 刷新令牌
   - 从 Redis 获取刷新令牌
   - 验证有效性
   - 颁发新的令牌对
   - 轮换刷新令牌（删除旧的）

4. **RevokeToken** - 撤销访问令牌
   - 解析令牌获取 TokenID
   - 加入黑名单（TTL = 剩余有效期）

5. **RevokeRefreshToken** - 撤销刷新令牌
   - 从 Redis 删除

**配置选项**:

- `WithAccessTTL`：自定义访问令牌有效期
- `WithRefreshTTL`：自定义刷新令牌有效期

## 📋 错误码定义

已在 `internal/pkg/code/` 添加认证相关错误码：

```go
// base.go
const (
    ErrUnauthenticated    = 100501  // 认证失败
    ErrUnauthorized       = 100502  // 授权失败（无权限）
    ErrInvalidCredentials = 100503  // 无效凭证
)

// authn.go (已存在)
const (
    ErrTokenInvalid       = 100005  // Token 无效
    ErrExpired            = 100203  // Token 过期
    ErrPasswordIncorrect  = 100206  // 密码错误
)
```

## 🏗️ 架构设计亮点

### 1. 策略模式（Authenticator）

- ✅ 每种认证方式独立实现
- ✅ 易于扩展新的认证方式
- ✅ 符合开闭原则

### 2. 端口-适配器（Ports & Adapters）

- ✅ 领域层定义端口接口
- ✅ 基础设施层实现适配器
- ✅ 依赖倒置，领域层不依赖具体实现

### 3. 值对象设计

- ✅ `Credential`：封装凭证验证逻辑
- ✅ `PasswordHash`：封装密码哈希算法
- ✅ `Token`：封装令牌业务逻辑

### 4. 安全设计

- ✅ 密码使用 Bcrypt 哈希
- ✅ 令牌黑名单机制
- ✅ 刷新令牌轮换（Rotation）
- ✅ 防止时序攻击（SecureCompare）

## 📁 目录结构

```text
internal/apiserver/modules/authn/domain/authentication/
├── credential.go              # 凭证值对象
├── password.go                # 密码哈希值对象
├── authentication.go          # 认证结果实体
├── token.go                   # 令牌值对象
├── port/                      # 端口接口
│   ├── authenticator.go       # 认证器接口
│   ├── account_password.go    # 账号密码端口
│   ├── wechat_auth.go         # 微信认证端口
│   └── token.go               # 令牌相关端口
└── service/                   # 领域服务
    ├── basic_authenticator.go      # 基础认证器
    ├── wechat_authenticator.go     # 微信认证器
    ├── authentication_service.go   # 认证服务
    └── token_service.go            # 令牌服务
```

## 🎯 下一步计划

### Phase 3: 基础设施实现

1. **JWT Generator** (`infrastructure/jwt/generator.go`)
   - 实现 `TokenGenerator` 接口
   - 使用 `github.com/golang-jwt/jwt/v5`
   - 生成和解析 JWT

2. **Redis Token Store** (`infrastructure/redis/token/store.go`)
   - 实现 `TokenStore` 接口
   - RefreshToken 存储
   - Token 黑名单管理

3. **Account Password Adapter** (`infrastructure/mysql/account/password_adapter.go`)
   - 实现 `AccountPasswordPort` 接口
   - 从数据库查询密码哈希

4. **WeChat Auth Adapter** (`infrastructure/wechat/auth_adapter.go`)
   - 实现 `WeChatAuthPort` 接口
   - 调用微信 API 换取 openID

### Phase 4: 应用服务

1. **LoginApplicationService**
   - 用例：登录（密码、微信）
   - 协调认证和令牌颁发

2. **TokenApplicationService**
   - 用例：验证令牌、刷新令牌、登出

### Phase 5: 接口层

1. **AuthHandler**
   - POST `/auth/login` - 登录
   - POST `/auth/login/wechat` - 微信登录
   - POST `/auth/refresh` - 刷新令牌
   - POST `/auth/logout` - 登出
   - GET `/auth/verify` - 验证令牌

## 🔍 使用示例

### 认证流程

```go
// 1. 创建认证器
basicAuth := NewBasicAuthenticator(accountRepo, operationRepo, passwordPort)
wechatAuth := NewWeChatAuthenticator(accountRepo, wechatRepo, wechatPort)

// 2. 创建认证服务
authService := NewAuthenticationService(basicAuth, wechatAuth)

// 3. 执行认证
credential := NewUsernamePasswordCredential("admin", "password")
auth, err := authService.Authenticate(ctx, credential)

// 4. 颁发令牌
tokenService := NewTokenService(tokenGenerator, tokenStore)
tokenPair, err := tokenService.IssueToken(ctx, auth)

// 5. 验证令牌
claims, err := tokenService.VerifyAccessToken(ctx, tokenPair.AccessToken.Value)

// 6. 刷新令牌
newTokenPair, err := tokenService.RefreshToken(ctx, tokenPair.RefreshToken.Value)

// 7. 撤销令牌
err = tokenService.RevokeToken(ctx, tokenPair.AccessToken.Value)
```

## ✅ 编译验证

```bash
✅ go build ./internal/apiserver/modules/authn/domain/authentication/...
# 编译成功，无错误
```

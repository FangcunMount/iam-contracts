# 认证模块应用服务层实现总结

## ✅ 已完成工作

### Phase 4: 应用服务层（已完成）

我已经成功实现了认证模块的 **2 个应用服务**，它们负责编排领域服务和基础设施，提供完整的业务用例。

---

## 1. LoginService (`application/login/service.go`)

### 职责

编排登录流程，协调认证和令牌颁发

### 依赖

- `AuthenticationService` - 认证领域服务
- `TokenService` - 令牌领域服务

### 方法

#### LoginWithPassword - 用户名密码登录

**请求**:

```go
type LoginWithPasswordRequest struct {
    Username string  // 用户名
    Password string  // 密码
    IP       string  // 客户端IP（可选）
    Device   string  // 设备信息（可选）
}
```

**响应**:

```go
type LoginWithPasswordResponse struct {
    AccessToken  string `json:"access_token"`
    RefreshToken string `json:"refresh_token"`
    TokenType    string `json:"token_type"`    // "Bearer"
    ExpiresIn    int64  `json:"expires_in"`    // 秒
}
```

**流程**:

1. 创建用户名密码凭证
2. 调用 `AuthenticationService.Authenticate()` 执行认证
3. 添加认证元数据（IP、设备信息）
4. 调用 `TokenService.IssueToken()` 颁发令牌
5. 构造并返回响应

#### LoginWithWeChat - 微信登录

**请求**:

```go
type LoginWithWeChatRequest struct {
    Code   string  // 微信授权码
    AppID  string  // 微信应用ID
    IP     string  // 客户端IP（可选）
    Device string  // 设备信息（可选）
}
```

**响应**:

```go
type LoginWithWeChatResponse struct {
    AccessToken  string `json:"access_token"`
    RefreshToken string `json:"refresh_token"`
    TokenType    string `json:"token_type"`    // "Bearer"
    ExpiresIn    int64  `json:"expires_in"`    // 秒
}
```

**流程**:

1. 创建微信凭证
2. 调用 `AuthenticationService.Authenticate()` 执行认证
3. 添加认证元数据（IP、设备信息）
4. 调用 `TokenService.IssueToken()` 颁发令牌
5. 构造并返回响应

---

## 2. TokenService (`application/token/service.go`)

### 2.1 职责

令牌管理，包括验证、刷新、撤销

### 2.2 依赖

- `TokenService` (领域服务) - 令牌领域服务

### 2.3 方法

#### VerifyToken - 验证访问令牌

**请求**:

```go
type VerifyTokenRequest struct {
    AccessToken string
}
```

**响应**:

```go
type VerifyTokenResponse struct {
    Valid     bool   `json:"valid"`
    UserID    uint64 `json:"user_id"`
    AccountID uint64 `json:"account_id"`
    TokenID   string `json:"token_id"`
}
```

**流程**:

1. 调用 `TokenService.VerifyAccessToken()` 验证令牌
2. 如果令牌无效或过期，返回 `valid=false`（不抛错误）
3. 如果令牌有效，返回用户信息

**特性**:

- 优雅处理令牌无效/过期，返回 `valid=false` 而非错误
- 适合中间件或网关使用

#### RefreshToken - 刷新访问令牌

**请求**:

```go
type RefreshTokenRequest struct {
    RefreshToken string
}
```

**响应**:

```go
type RefreshTokenResponse struct {
    AccessToken  string `json:"access_token"`
    RefreshToken string `json:"refresh_token"`
    TokenType    string `json:"token_type"`
    ExpiresIn    int64  `json:"expires_in"`
}
```

**流程**:

1. 调用 `TokenService.RefreshToken()` 刷新令牌
2. 返回新的令牌对（访问令牌 + 刷新令牌）

**特性**:

- 实现刷新令牌轮换（Rotation）
- 旧刷新令牌被删除，只有新刷新令牌有效

#### Logout - 登出

**请求**:

```go
type LogoutRequest struct {
    AccessToken  string  // 访问令牌（必需）
    RefreshToken string  // 刷新令牌（可选）
}
```

**流程**:

1. 调用 `TokenService.RevokeToken()` 撤销访问令牌（加入黑名单）
2. 如果提供了刷新令牌，调用 `TokenService.RevokeRefreshToken()` 删除它
3. 刷新令牌撤销失败不影响主流程

**特性**:

- 访问令牌加入黑名单（TTL = 剩余有效期）
- 刷新令牌从 Redis 删除
- 支持部分撤销（只撤销访问令牌）

#### GetUserInfo - 获取用户信息

**请求**:

```go
type GetUserInfoRequest struct {
    AccessToken string
}
```

**响应**:

```go
type GetUserInfoResponse struct {
    UserID    uint64 `json:"user_id"`
    AccountID uint64 `json:"account_id"`
    // 可扩展更多用户信息
}
```

**流程**:

1. 验证并解析访问令牌
2. 返回令牌中的用户信息

**扩展性**:

- 可以注入 UserAdapter 查询用户中心获取更多信息
- 当前只返回令牌中的基本信息

---

## 📁 目录结构

```text
internal/apiserver/modules/authn/application/
├── account/           # 已有：账号管理
│   ├── register.go
│   └── query.go
├── adapter/           # 已有：适配器
│   ├── user_adapter.go
│   └── user_adapter_impl.go
├── login/             # 新增：登录应用服务
│   └── service.go
└── token/             # 新增：令牌应用服务
    └── service.go
```

---

## 🔗 依赖关系图

```text
接口层 (Handler)
    ↓ 调用
应用服务层 (Application Service)
    ├── LoginService
    │   ├─→ AuthenticationService (领域服务)
    │   └─→ TokenService (领域服务)
    └── TokenService (应用服务)
        └─→ TokenService (领域服务)
            ├─→ TokenGenerator (基础设施)
            └─→ TokenStore (基础设施)
```

---

## 💡 设计模式

### 1. 用例驱动设计

- 每个方法对应一个业务用例
- 清晰的请求/响应模型
- 便于理解和维护

### 2. 编排模式

- 应用服务只负责编排
- 不包含业务逻辑（业务逻辑在领域层）
- 协调多个领域服务完成用例

### 3. DTO 模式

- 使用独立的请求/响应对象
- 与领域模型解耦
- 适合跨层传输

### 4. 错误处理策略

- **VerifyToken**: 优雅处理，返回 `valid=false`
- **其他方法**: 传播领域层错误
- **Logout**: 部分失败不影响主流程

---

## ✅ 编译验证

```bash
✅ go build ./internal/apiserver/modules/authn/application/...
# 编译成功，无错误
```

---

## 🎯 下一步计划

### Phase 5: 接口层（RESTful Handler）

需要实现 HTTP Handler，对外提供 RESTful API：

#### AuthHandler

**路由**:

- `POST /api/v1/auth/login` - 密码登录
- `POST /api/v1/auth/login/wechat` - 微信登录
- `POST /api/v1/auth/refresh` - 刷新令牌
- `POST /api/v1/auth/logout` - 登出
- `GET /api/v1/auth/verify` - 验证令牌
- `GET /api/v1/auth/userinfo` - 获取用户信息

**职责**:

- 解析 HTTP 请求
- 调用应用服务
- 构造 HTTP 响应
- 错误处理和状态码映射

---

## 📝 使用示例

### 初始化应用服务

```go
// 1. 创建基础设施
jwtGenerator := jwt.NewGenerator("secret", "issuer")
redisStore := token.NewRedisStore(redisClient)
passwordAdapter := account.NewPasswordAdapter(operationRepo)
wechatAdapter := wechat.NewAuthAdapter(/* config */)

// 2. 创建领域服务
basicAuth := authentication.NewBasicAuthenticator(
    accountRepo,
    operationRepo,
    passwordAdapter,
)
wechatAuth := authentication.NewWeChatAuthenticator(
    accountRepo,
    wechatRepo,
    wechatAdapter,
)
authService := authentication.NewAuthenticationService(basicAuth, wechatAuth)
tokenService := authentication.NewTokenService(jwtGenerator, redisStore)

// 3. 创建应用服务
loginService := login.NewLoginService(authService, tokenService)
tokenAppService := tokenapp.NewTokenService(tokenService)
```

### 密码登录示例

```go
// 处理登录请求
req := &login.LoginWithPasswordRequest{
    Username: "admin",
    Password: "password123",
    IP:       "192.168.1.100",
    Device:   "iPhone 13",
}

resp, err := loginService.LoginWithPassword(ctx, req)
if err != nil {
    // 处理错误
    return err
}

// 返回响应
// {
//   "access_token": "eyJhbGc...",
//   "refresh_token": "550e8400-e29b-41d4-a716-446655440000",
//   "token_type": "Bearer",
//   "expires_in": 900
// }
```

### 验证令牌示例

```go
req := &tokenapp.VerifyTokenRequest{
    AccessToken: "eyJhbGc...",
}

resp, err := tokenAppService.VerifyToken(ctx, req)
if err != nil {
    return err
}

if !resp.Valid {
    // 令牌无效或过期
    return errors.New("token invalid")
}

// 使用用户信息
userID := resp.UserID
```

### 刷新令牌示例

```go
req := &tokenapp.RefreshTokenRequest{
    RefreshToken: "550e8400-e29b-41d4-a716-446655440000",
}

resp, err := tokenAppService.RefreshToken(ctx, req)
if err != nil {
    return err
}

// 返回新令牌对
// {
//   "access_token": "eyJhbGc...(new)",
//   "refresh_token": "660f9511-f39c-52e5-b827-557766551111(new)",
//   "token_type": "Bearer",
//   "expires_in": 900
// }
```

### 登出示例

```go
req := &tokenapp.LogoutRequest{
    AccessToken:  "eyJhbGc...",
    RefreshToken: "550e8400-e29b-41d4-a716-446655440000",
}

err := tokenAppService.Logout(ctx, req)
if err != nil {
    return err
}

// 令牌已撤销
```

---

## 🔒 安全考虑

### 1. 认证元数据

- 记录 IP 地址（用于异常检测）
- 记录设备信息（用于会话管理）
- 可扩展：位置、浏览器等

### 2. 令牌安全

- 访问令牌短期有效（15分钟）
- 刷新令牌长期有效（7天）
- 刷新令牌轮换（防止重放攻击）
- 黑名单机制（撤销令牌）

### 3. 错误处理

- 不泄露敏感信息
- 统一错误码
- 区分客户端错误和服务器错误

### 4. 审计日志

- 记录所有认证尝试
- 记录令牌颁发和撤销
- 便于安全审计

---

## 🚀 性能优化建议

### 1. 缓存用户信息

```go
// 可以在 GetUserInfo 中添加缓存
type TokenService struct {
    tokenService *authService.TokenService
    userCache    cache.Cache  // 添加缓存
}
```

### 2. Redis 连接池

- 使用连接池避免频繁建连
- 设置合理的超时时间

### 3. 并发控制

- 限制并发登录请求（防止暴力破解）
- 使用滑动窗口限流

### 4. 预热和健康检查

- 启动时预热 Redis 连接
- 定期检查基础设施健康状态

---

## ✨ 特性亮点

1. **清晰的职责分离**
   - 应用服务：用例编排
   - 领域服务：业务逻辑
   - 基础设施：技术实现

2. **优雅的错误处理**
   - VerifyToken 返回 valid 标志而非抛错
   - 部分失败不影响主流程

3. **可扩展的设计**
   - 易于添加新的登录方式
   - 易于扩展用户信息
   - 易于添加审计日志

4. **安全的令牌管理**
   - 刷新令牌轮换
   - 黑名单机制
   - 记录认证元数据

5. **符合行业标准**
   - OAuth 2.0 令牌响应格式
   - JWT 标准声明
   - RESTful API 设计

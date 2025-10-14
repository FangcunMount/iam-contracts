# 认证模块基础设施层实现总结

## ✅ 已完成工作

### Phase 3: 基础设施层（已完成）

我已经成功实现了认证模块所需的 4 个基础设施适配器：

---

## 1. JWT Generator (`infra/jwt/generator.go`)

### 职责

生成和解析 JWT 访问令牌

### 实现细节

- **使用库**: `github.com/golang-jwt/jwt/v4`
- **签名算法**: HS256 (HMAC-SHA256)
- **自定义 Claims**:

  ```go
  type CustomClaims struct {
      UserID    uint64 `json:"user_id"`
      AccountID uint64 `json:"account_id"`
      jwt.StandardClaims
  }
  ```

### 方法

#### GenerateAccessToken

- 生成 JWT 访问令牌
- 包含：TokenID、UserID、AccountID、签发时间、过期时间
- 返回 `authentication.Token` 领域对象

#### ParseAccessToken

- 解析 JWT 字符串
- 验证签名和有效期
- 返回 `authentication.TokenClaims` 领域对象

### 配置

- `secretKey`: JWT 签名密钥（建议使用强随机字符串）
- `issuer`: 令牌颁发者标识

---

## 2. Redis Token Store (`infra/redis/token/store.go`)

### 2.1 职责

存储和管理刷新令牌及令牌黑名单

### 2.2 实现细节

- **使用库**: `github.com/go-redis/redis/v7`
- **存储结构**:

  ```go
  type refreshTokenData struct {
      TokenID   string
      UserID    uint64
      AccountID uint64
      ExpiresAt time.Time
  }
  ```

### 2.3 方法

#### SaveRefreshToken

- 保存刷新令牌到 Redis
- **Key 格式**: `refresh_token:{token_value}`
- **TTL**: 令牌剩余有效期
- **Value**: JSON 序列化的 refreshTokenData

#### GetRefreshToken

- 从 Redis 获取刷新令牌
- 如果不存在或已过期返回 nil

#### DeleteRefreshToken

- 删除刷新令牌（用于撤销或轮换）
- **Key**: `refresh_token:{token_value}`

#### AddToBlacklist

- 将访问令牌加入黑名单
- **Key 格式**: `token_blacklist:{token_id}`
- **TTL**: 令牌剩余有效期
- **Value**: "1"（标记）

#### IsBlacklisted

- 检查令牌是否在黑名单中
- 通过检查 Redis key 是否存在判断

### Redis Key 设计

| Key 模式 | 用途 | TTL | Value |
|---------|------|-----|-------|
| `refresh_token:{value}` | 存储刷新令牌 | 令牌有效期 | JSON 数据 |
| `token_blacklist:{id}` | 令牌黑名单 | 令牌剩余有效期 | "1" |

---

## 3. Account Password Adapter (`infra/mysql/account/password_adapter.go`)

### 3.1 职责

从数据库查询账号的密码哈希信息

### 3.2 实现细节

- **依赖**: `accountPort.OperationRepo`
- **查询流程**:
  1. 根据 AccountID 查询 OperationAccount
  2. 提取密码哈希字段
  3. 转换为领域模型 `authentication.PasswordHash`

### 3.3 方法

#### GetPasswordHash

- 输入: `accountID account.AccountID`
- 输出: `*authentication.PasswordHash`
- 字段映射:
  - `PasswordHash` → Hash
  - `Algo` → Algorithm
  - `Params` → Parameters

### 数据库字段

```go
type OperationAccount struct {
    AccountID      AccountID
    Username       string
    PasswordHash   []byte     // 密码哈希
    Algo           string     // 算法: bcrypt, argon2id, scrypt
    Params         []byte     // 算法参数（JSON）
    FailedAttempts int
    LockedUntil    *time.Time
    LastChangedAt  time.Time
}
```

---

## 4. WeChat Auth Adapter (`infra/wechat/auth_adapter.go`)

### 4.1 职责

调用微信 API 换取用户 openID

### 4.2 实现细节

- **API**: 微信小程序登录 `code2session`
- **文档**: <https://developers.weixin.qq.com/miniprogram/dev/api-backend/open-api/login/auth.code2Session.html>
- **HTTP 客户端**: 支持自定义（默认使用 `http.DefaultClient`）

### 4.3 方法

#### ExchangeOpenID

- 输入: `code`（微信授权码）, `appID`
- 输出: `openID`
- 流程:
  1. 根据 appID 获取 appSecret
  2. 构建请求 URL（包含 appid、secret、js_code）
  3. 发送 GET 请求到微信 API
  4. 解析响应获取 openID
  5. 错误处理（检查 errcode）

### 配置选项

- `WithHTTPClient`: 自定义 HTTP 客户端
- `WithAppConfig`: 添加应用配置（appID → appSecret）

### 微信 API 响应

```go
type WeChatAuthResponse struct {
    OpenID     string `json:"openid"`
    SessionKey string `json:"session_key"`
    UnionID    string `json:"unionid"`
    ErrCode    int    `json:"errcode"`
    ErrMsg     string `json:"errmsg"`
}
```

### 安全建议

- ⚠️ appSecret 应该从配置文件或数据库读取，不要硬编码
- ⚠️ 生产环境建议使用 HTTPS 证书验证
- ⚠️ 建议设置 HTTP 超时时间

---

## 📁 目录结构

```text
internal/apiserver/modules/authn/infra/
├── jwt/
│   └── generator.go           # JWT 生成器
├── redis/
│   └── token/
│       └── store.go          # Redis 令牌存储
├── mysql/
│   └── account/
│       ├── mapper.go         # 已有
│       ├── repo_account.go   # 已有
│       ├── repo_operation.go # 已有
│       ├── repo_wechat.go    # 已有
│       └── password_adapter.go # 新增：密码适配器
└── wechat/
    └── auth_adapter.go       # 微信认证适配器
```

---

## 🔗 依赖关系图

```text
领域服务 (domain/authentication/service)
    ↓ 依赖
端口接口 (domain/authentication/port)
    ↓ 实现
基础设施适配器 (infra)
    ├── JWT Generator      → TokenGenerator 接口
    ├── Redis Token Store  → TokenStore 接口
    ├── Password Adapter   → AccountPasswordPort 接口
    └── WeChat Adapter     → WeChatAuthPort 接口
```

---

## ✅ 编译验证

```bash
✅ go build ./internal/apiserver/modules/authn/infra/...
# 编译成功，无错误
```

---

## 🎯 下一步计划

### Phase 4: 应用服务层

需要实现 2 个应用服务：

#### 1. LoginApplicationService

**职责**: 协调登录流程

**用例**:

- `LoginWithPassword(ctx, username, password) (*TokenPair, error)`
  - 创建用户名密码凭证
  - 调用 AuthenticationService.Authenticate()
  - 调用 TokenService.IssueToken()
  - 返回令牌对

- `LoginWithWeChat(ctx, code, appID) (*TokenPair, error)`
  - 创建微信凭证
  - 调用 AuthenticationService.Authenticate()
  - 调用 TokenService.IssueToken()
  - 返回令牌对

#### 2. TokenApplicationService

**职责**: 令牌管理

**用例**:

- `VerifyToken(ctx, accessToken) (*UserInfo, error)`
  - 调用 TokenService.VerifyAccessToken()
  - 返回用户信息

- `RefreshAccessToken(ctx, refreshToken) (*TokenPair, error)`
  - 调用 TokenService.RefreshToken()
  - 返回新令牌对

- `Logout(ctx, accessToken, refreshToken) error`
  - 调用 TokenService.RevokeToken()
  - 调用 TokenService.RevokeRefreshToken()

### Phase 5: 接口层

实现 RESTful Handler:

- `POST /auth/login` - 密码登录
- `POST /auth/login/wechat` - 微信登录
- `POST /auth/refresh` - 刷新令牌
- `POST /auth/logout` - 登出
- `GET /auth/verify` - 验证令牌

---

## 💡 使用示例

### 初始化基础设施

```go
// 1. 创建 JWT Generator
jwtGenerator := jwt.NewGenerator("your-secret-key", "iam-contracts")

// 2. 创建 Redis Token Store
redisClient := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
})
tokenStore := token.NewRedisStore(redisClient)

// 3. 创建 Password Adapter
passwordAdapter := account.NewPasswordAdapter(operationRepo)

// 4. 创建 WeChat Adapter
wechatAdapter := wechat.NewAuthAdapter(
    wechat.WithAppConfig("wx123456", "your-app-secret"),
)

// 5. 创建认证器
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

// 6. 创建认证服务
authService := authentication.NewAuthenticationService(basicAuth, wechatAuth)

// 7. 创建令牌服务
tokenService := authentication.NewTokenService(
    jwtGenerator,
    tokenStore,
    authentication.WithAccessTTL(15 * time.Minute),
    authentication.WithRefreshTTL(7 * 24 * time.Hour),
)
```

### 完整认证流程

```go
// 密码登录
credential := authentication.NewUsernamePasswordCredential("admin", "password123")
auth, err := authService.Authenticate(ctx, credential)
if err != nil {
    // 处理认证失败
}

// 颁发令牌
tokenPair, err := tokenService.IssueToken(ctx, auth)
if err != nil {
    // 处理令牌颁发失败
}

// 返回给客户端
response := map[string]string{
    "access_token":  tokenPair.AccessToken.Value,
    "refresh_token": tokenPair.RefreshToken.Value,
    "expires_in":    "900", // 15分钟
}
```

---

## 🔒 安全注意事项

1. **JWT Secret**:
   - 使用强随机字符串（至少 32 字节）
   - 定期轮换密钥
   - 不要硬编码，从环境变量或配置文件读取

2. **Redis 连接**:
   - 生产环境启用密码认证
   - 使用 TLS 加密连接
   - 限制网络访问（只允许应用服务器连接）

3. **微信 AppSecret**:
   - 永远不要暴露在客户端
   - 从安全的配置管理系统读取
   - 定期检查并轮换

4. **密码验证**:
   - 使用 Bcrypt（已实现）
   - 考虑添加登录失败限流
   - 实现账号锁定机制（OperationAccount 已有字段）

5. **令牌管理**:
   - 访问令牌设置较短有效期（15分钟）
   - 刷新令牌设置较长有效期（7天）
   - 敏感操作需要验证黑名单
   - 用户修改密码或登出时撤销所有令牌

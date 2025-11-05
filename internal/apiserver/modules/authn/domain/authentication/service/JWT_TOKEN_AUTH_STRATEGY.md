# JWT Token 认证策略

## 概述

JWT Token 认证策略是一种使用 JWT 访问令牌进行身份认证的策略，主要用于 API 调用场景。用户在首次登录后获得 JWT 访问令牌，后续的 API 请求可以携带此令牌进行认证，无需每次都提供用户名密码。

## 认证场景

- **Scenario**: `AuthJWTToken` (`"jwt_token"`)
- **AMR**: `AMRJWTToken` (`"jwt"`)

## 认证流程

```text
┌─────────────┐
│   客户端    │
└──────┬──────┘
       │ 1. 携带 JWT Token
       ▼
┌─────────────────────────┐
│  JWTTokenAuthStrategy   │
└──────────┬──────────────┘
           │
           │ 2. 验证 Token
           ▼
┌─────────────────────────┐
│   TokenVerifier         │ ◄── 调用 token 模块的验证服务
│  (TokenVerifierAdapter) │
└──────────┬──────────────┘
           │
           │ 3. 提取 UserID, AccountID
           ▼
┌─────────────────────────┐
│   AccountRepository     │ ◄── 检查账户状态
└──────────┬──────────────┘
           │
           │ 4. 返回 Principal
           ▼
┌─────────────────────────┐
│     AuthDecision        │
└─────────────────────────┘
```

## 关键步骤

1. **Token 验证**: 调用 `TokenVerifier.VerifyAccessToken()` 验证 JWT Token
   - 验证签名是否有效
   - 检查是否过期
   - 检查是否在黑名单中

2. **提取身份信息**: 从 Token Claims 中提取：
   - UserID（用户ID）
   - AccountID（账户ID）
   - TenantID（租户ID，可选）

3. **账户状态检查**: 验证账户是否：
   - 已启用 (enabled)
   - 未锁定 (not locked)

4. **构造认证主体**: 返回包含用户身份信息的 `Principal`

## 使用示例

### 1. 应用层调用

```go
import (
    "context"
    domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/authentication"
)

// 使用 JWT Token 进行认证
func authenticateWithToken(authService *AuthenticationService, accessToken string) error {
    ctx := context.Background()
    
    input := domain.AuthInput{
        AccessToken: accessToken,
        RemoteIP:    "192.168.1.100",
        UserAgent:   "MyApp/1.0",
    }
    
    decision, err := authService.Authenticate(ctx, domain.AuthJWTToken, input)
    if err != nil {
        return fmt.Errorf("authentication failed: %w", err)
    }
    
    if !decision.OK {
        return fmt.Errorf("authentication failed: %s", decision.ErrCode)
    }
    
    // 认证成功，使用 Principal
    principal := decision.Principal
    fmt.Printf("Authenticated user: %d, account: %d\n", 
        principal.UserID, principal.AccountID)
    
    return nil
}
```

### 2. HTTP Handler 示例

```go
func (h *Handler) AuthenticateWithToken(w http.ResponseWriter, r *http.Request) {
    // 从 Header 中提取 Bearer Token
    authHeader := r.Header.Get("Authorization")
    if !strings.HasPrefix(authHeader, "Bearer ") {
        http.Error(w, "Missing Bearer token", http.StatusUnauthorized)
        return
    }
    
    accessToken := strings.TrimPrefix(authHeader, "Bearer ")
    
    // 调用认证服务
    input := domain.AuthInput{
        AccessToken: accessToken,
        RemoteIP:    getClientIP(r),
        UserAgent:   r.UserAgent(),
    }
    
    decision, err := h.authService.Authenticate(r.Context(), domain.AuthJWTToken, input)
    if err != nil {
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }
    
    if !decision.OK {
        switch decision.ErrCode {
        case domain.ErrInvalidCredential:
            http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
        case domain.ErrDisabled:
            http.Error(w, "Account disabled", http.StatusForbidden)
        case domain.ErrLocked:
            http.Error(w, "Account locked", http.StatusForbidden)
        default:
            http.Error(w, "Authentication failed", http.StatusUnauthorized)
        }
        return
    }
    
    // 认证成功，将 Principal 放入 context
    ctx := context.WithValue(r.Context(), "principal", decision.Principal)
    r = r.WithContext(ctx)
    
    // 继续处理请求
    h.handleProtectedResource(w, r)
}
```

### 3. 依赖注入配置

在创建 `AuthenticationService` 时，需要注入 `TokenVerifier`：

```go
import (
    tokenService "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/token/service"
)

func setupAuthenticationService(
    db *gorm.DB,
    redisClient *redis.Client,
    // ... 其他参数
) *AuthenticationService {
    // 1. 创建 Token 模块的服务
    tokenGenerator := jwt.NewGenerator(...)
    tokenStore := redisToken.NewRedisStore(redisClient)
    tokenVerifyer := tokenService.NewTokenVerifyer(tokenGenerator, tokenStore)
    
    // 2. 创建 Authentication 服务（传入 tokenVerifyer）
    authService := NewAuthenticationService(
        db,
        redisClient,
        wechatCache,
        pepper,
        wxMinipSecrets,
        wecomSecrets,
        tokenVerifyer, // 注入 TokenVerifier
    )
    
    return authService
}
```

## 错误处理

### 常见错误码

| 错误码 | 说明 | HTTP 状态码建议 |
|--------|------|----------------|
| `ErrInvalidCredential` | Token 无效、过期或被撤销 | 401 Unauthorized |
| `ErrDisabled` | 账户已被禁用 | 403 Forbidden |
| `ErrLocked` | 账户已被锁定 | 403 Forbidden |

### Token 验证失败原因

1. **签名无效**: JWT 签名验证失败
2. **Token 过期**: Token 的 `exp` 字段已过期
3. **Token 被撤销**: Token 在黑名单中
4. **Token 格式错误**: 无法解析 JWT

## 与其他认证策略的比较

| 策略 | 用途 | 凭据类型 | 交互次数 |
|------|------|----------|----------|
| Password | 用户登录 | 用户名+密码 | 每次都需要验证密码 |
| PhoneOTP | 手机验证码登录 | 手机号+OTP | 每次都需要发送/验证OTP |
| JWTToken | API调用 | JWT访问令牌 | 仅验证Token签名和状态 |
| WeChatMini | 微信小程序登录 | 微信code | 需要与微信服务器交互 |

## 安全注意事项

1. **HTTPS**: 必须在 HTTPS 上传输 JWT Token，防止中间人攻击
2. **Token 有效期**: 设置合理的 Token 过期时间（建议15分钟-1小时）
3. **Token 撤销**: 支持将 Token 加入黑名单，实现强制登出
4. **刷新机制**: 配合 Refresh Token 使用，避免频繁重新登录
5. **存储安全**: 客户端应安全存储 Token（避免 XSS 攻击）

## 架构说明

### 依赖关系

```text
authentication 模块 ──┐
                     ├──► TokenVerifier 接口（authentication/port/driven）
                     │
                     └──► TokenVerifierAdapter（infra/authentication）
                          │
                          └──► TokenVerifier 实现（token 模块）
                               │
                               ├──► TokenGenerator（JWT 解析）
                               └──► TokenStore（黑名单检查）
```

### 端口与适配器

- **Driven Port**: `authentication.port.TokenVerifier` - 认证模块需要的接口
- **Adapter**: `TokenVerifierAdapter` - 适配器，将 token 模块的接口转换为认证模块需要的接口
- **Implementation**: `token.service.TokenVerifyer` - token 模块的实际实现

这种设计遵循了**依赖倒置原则**和**适配器模式**，使得 authentication 模块不直接依赖 token 模块的具体实现。

## 扩展

### 添加 TenantID 支持

如果需要在 Token 中携带租户信息，可以：

1. 在 `token.TokenClaims` 中添加 `TenantID` 字段
2. 在 JWT 生成时包含 `tenant_id` claim
3. 修改 `TokenVerifierAdapter` 提取 `TenantID`

### 添加权限信息

可以在 Token Claims 中包含用户权限：

```go
principal.Claims["roles"] = []string{"admin", "user"}
principal.Claims["permissions"] = []string{"read", "write"}
```

### 审计日志

在认证成功/失败时记录审计日志：

```go
if decision.OK {
    auditLogger.LogAuthAttempt(ctx, AuthAuditEvent{
        AccountID: decision.Principal.AccountID,
        Scenario:  "jwt_token",
        Success:   true,
        RemoteIP:  input.RemoteIP,
        UserAgent: input.UserAgent,
    })
}
```

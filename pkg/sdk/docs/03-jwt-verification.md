# JWT 本地验证

## 🎯 30 秒搞懂

### 金字塔架构

```text
                    JWT Token
                        │
                  Token Verifier
                 (验证器/策略选择)
                        │
        ┌───────────────┼───────────────┐
        ↓               ↓               ↓
    本地验证         远程验证         组合策略
    (最快)          (兜底)          (智能)
        ↓               ↓               ↓
    JWKS Manager    gRPC Call      缓存+降级
    (职责链)
        │
        ├─ Cache    (内存/最快)
        ├─ HTTP     (主要)
        ├─ gRPC     (降级)
        └─ Seed     (兜底)
```

### 双重设计模式

```text
┌──────────────────────────────────────────────────────┐
│ TokenVerifier (Strategy 策略模式)                     │
│                                                       │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐    │
│  │Local验证    │  │Remote验证   │  │Caching验证 │    │
│  │<1ms        │  │~50ms       │  │<1ms (缓存) │    │
│  └────────────┘  └────────────┘  └────────────┘    │
└──────────────────────────────────────────────────────┘
                        ↓
┌──────────────────────────────────────────────────────┐
│ JWKSManager (Chain of Responsibility 职责链)          │
│                                                       │
│  Cache → CircuitBreaker → HTTP → gRPC → Seed        │
│  ↓        ↓                ↓      ↓      ↓          │
│  快速     保护             主要   降级    兜底         │
└──────────────────────────────────────────────────────┘
```

### 工作流程

```text
1️⃣ 请求到达
   Token: "eyJhbGciOiJSUzI1..."
          ↓
2️⃣ Verifier 选择策略
   ┌─ 有 JWKS? → Local (本地验证)
   ├─ 无 JWKS? → Remote (远程验证)
   └─ 启用缓存? → Caching (缓存策略)
          ↓
3️⃣ JWKS Manager 获取密钥
   ┌─ Cache 命中? → 直接返回 (0.1ms)
   ├─ HTTP 成功? → 返回并缓存 (10ms)
   ├─ gRPC 成功? → 降级返回 (20ms)
   └─ Seed 存在? → 兜底返回 (0.5ms)
          ↓
4️⃣ 验证 JWT
   ✓ 签名验证
   ✓ 过期时间检查
   ✓ Audience/Issuer 校验
          ↓
5️⃣ 返回结果
   Valid: true
   UserID: "user-123"
   Roles: ["admin", "user"]
```

### 性能对比

| 验证方式 | 延迟 | 可靠性 | 适用场景 |
|---------|------|-------|---------|
| 🚀 **本地+缓存** | <1ms | ⭐⭐⭐⭐⭐ | 高并发 API |
| ⚡ **本地验证** | 1-5ms | ⭐⭐⭐⭐ | 常规 API |
| 🌐 **远程验证** | 20-50ms | ⭐⭐⭐ | 非幂等操作 |
| 🔄 **Fallback** | 自适应 | ⭐⭐⭐⭐⭐ | 生产推荐 |

### 降级链路

```text
正常: HTTP JWKS (10ms)
  ↓ 失败
降级1: gRPC JWKS (20ms)
  ↓ 失败
降级2: 内存 Cache (0.1ms)
  ↓ 失败
降级3: 本地 Seed (0.5ms)
  ↓ 失败
兜底: Remote 验证 (50ms)
```

### 3 行代码开始

```go
// 1️⃣ 创建验证器
verifier, _ := sdk.NewTokenVerifier(
    &sdk.TokenVerifyConfig{AllowedAudience: []string{"my-app"}},
    &sdk.JWKSConfig{URL: "https://iam.example.com/.well-known/jwks.json"},
    nil,
)

// 2️⃣ 验证 Token
result, _ := verifier.Verify(ctx, token, nil)

// 3️⃣ 使用结果
if result.Valid {
    log.Printf("用户: %s, 角色: %v", result.Claims.UserID, result.Claims.Roles)
}
```

---

## 📖 详细说明

### 为什么需要本地验证？

| 对比项 | 远程验证 | 本地验证 |
|-------|---------|---------|
| 性能 | ❌ 50ms+ | ✅ <1ms |
| 可靠性 | ❌ 依赖IAM服务 | ✅ 本地独立 |
| 网络开销 | ❌ 每次请求 | ✅ 定期刷新 |
| 适合场景 | 低频操作 | 高频API |

---

## 🏗️ 架构设计

```text
TokenVerifier (Strategy 模式)
├── LocalVerifyStrategy     ← 使用 JWKS 本地验证
├── RemoteVerifyStrategy    ← 调用 IAM 服务验证
├── FallbackVerifyStrategy  ← 先本地，失败后远程
└── CachingVerifyStrategy   ← 添加结果缓存

JWKSManager (Chain of Responsibility 模式)
├── CacheFetcher           ← 内存缓存
├── CircuitBreakerFetcher  ← 熔断保护
├── HTTPFetcher            ← HTTP 获取
├── GRPCEndpointFetcher    ← gRPC 降级
└── SeedFetcher            ← 本地种子备份
```

## 快速开始

### 1. 最简配置

```go
import sdk "github.com/FangcunMount/iam-contracts/pkg/sdk"

// 创建验证器
verifier, err := sdk.NewTokenVerifier(
    &sdk.TokenVerifyConfig{
        AllowedAudience: []string{"my-app"},
        AllowedIssuer:   "https://iam.example.com",
    },
    &sdk.JWKSConfig{
        URL: "https://iam.example.com/.well-known/jwks.json",
    },
    nil, // 不需要 gRPC 客户端
)

// 验证 Token
result, err := verifier.Verify(ctx, accessToken, nil)
if err != nil {
    log.Printf("验证失败: %v", err)
    return
}

if !result.Valid {
    log.Println("Token 无效")
    return
}

// 使用 Claims
log.Printf("用户 ID: %s", result.Claims.UserID)
log.Printf("角色: %v", result.Claims.Roles)
log.Printf("过期时间: %v", result.Claims.ExpiresAt)
```

### 2. 带远程降级

```go
// 创建 IAM 客户端
client, err := sdk.NewClient(ctx, &sdk.Config{
    Endpoint: "iam.example.com:8081",
    TLS: &sdk.TLSConfig{
        Enabled: true,
        CACert:  "/etc/iam/certs/ca.crt",
    },
})

// 创建验证器（带远程降级）
verifier, err := sdk.NewTokenVerifier(
    &sdk.TokenVerifyConfig{
        AllowedAudience:         []string{"my-app"},
        AllowedIssuer:           "https://iam.example.com",
        ForceRemoteVerification: false, // 本地优先
    },
    &sdk.JWKSConfig{
        URL:             "https://iam.example.com/.well-known/jwks.json",
        RefreshInterval: 5 * time.Minute,
    },
    client, // 提供客户端用于远程验证
)
```

## TokenVerifyConfig 配置

```go
type TokenVerifyConfig struct {
    // AllowedAudience 允许的 audience 列表
    AllowedAudience []string
    
    // AllowedIssuer 允许的 issuer
    AllowedIssuer string
    
    // ClockSkew 时钟偏差容忍度（默认 1 分钟）
    ClockSkew time.Duration
    
    // RequireExpirationTime 是否要求 exp 声明
    RequireExpirationTime bool
    
    // ForceRemoteVerification 强制使用远程验证
    ForceRemoteVerification bool
    
    // RequiredClaims 必须存在的声明列表
    // 例如: []string{"sub", "aud", "exp", "iat", "user_id"}
    RequiredClaims []string
    
    // Algorithms 允许的签名算法列表
    // 支持: RS256, RS384, RS512, ES256, ES384, ES512, PS256, PS384, PS512, EdDSA
    // 如果为空，默认只允许 RS256
    Algorithms []string
}
```

### 配置示例

```go
&TokenVerifyConfig{
    AllowedAudience:         []string{"app-1", "app-2"},
    AllowedIssuer:           "https://iam.example.com",
    ClockSkew:               time.Minute,
    RequireExpirationTime:   true,
    ForceRemoteVerification: false,
    RequiredClaims:          []string{"sub", "user_id"},  // 必须包含这些声明
    Algorithms:              []string{"RS256", "ES256"},  // 只允许这些算法
}
```

## JWKSConfig 配置

```go
type JWKSConfig struct {
    // URL JWKS 端点 URL (HTTP/HTTPS)
    URL string
    
    // GRPCEndpoint gRPC 降级端点（HTTP 失败时使用）
    GRPCEndpoint string
    
    // RefreshInterval 刷新间隔
    RefreshInterval time.Duration
    
    // RequestTimeout HTTP 请求超时
    RequestTimeout time.Duration
    
    // CacheTTL 缓存 TTL
    CacheTTL time.Duration
    
    // HTTPClient 自定义 HTTP 客户端
    HTTPClient *http.Client
    
    // CustomHeaders 自定义请求头
    CustomHeaders map[string]string
    
    // FallbackOnError 失败时使用缓存
    FallbackOnError bool
}
```

### JWKS 配置示例

```go
&JWKSConfig{
    URL:             "https://iam.example.com/.well-known/jwks.json",
    GRPCEndpoint:    "iam.example.com:8081", // HTTP 失败时降级到 gRPC
    RefreshInterval: 5 * time.Minute,
    RequestTimeout:  10 * time.Second,
    CacheTTL:        1 * time.Hour,
    FallbackOnError: true,
    CustomHeaders: map[string]string{
        "X-API-Key": "your-api-key",
    },
}
```

## 验证结果

```go
type VerifyResult struct {
    Valid  bool               // Token 是否有效
    Claims *StandardClaims    // JWT Claims
    Source VerifySource       // 验证来源（本地/远程/缓存）
    Error  string             // 错误信息（如果有）
}

type StandardClaims struct {
    UserID    string    // 用户 ID (sub)
    Audience  []string  // Audience (aud)
    Issuer    string    // Issuer (iss)
    IssuedAt  time.Time // 签发时间 (iat)
    ExpiresAt time.Time // 过期时间 (exp)
    NotBefore time.Time // 生效时间 (nbf)
    Roles     []string  // 角色列表（自定义 claim）
    Scope     []string  // 权限范围（自定义 claim）
}
```

### 使用示例

```go
result, err := verifier.Verify(ctx, token, nil)
if err != nil {
    log.Printf("验证错误: %v", err)
    return
}

if !result.Valid {
    log.Printf("Token 无效: %s", result.Error)
    return
}

// 检查角色
if contains(result.Claims.Roles, "admin") {
    log.Println("管理员用户")
}

// 检查过期
if time.Now().After(result.Claims.ExpiresAt) {
    log.Println("Token 已过期")
}

// 查看验证来源
log.Printf("验证来源: %s", result.Source) // "local", "remote", "cache"
```

## 高级用法

### 1. 自定义 JWKS Manager

```go
import "github.com/FangcunMount/iam-contracts/pkg/sdk/auth"

// 创建自定义 JWKS Manager
jwksManager, err := auth.NewJWKSManager(
    &config.JWKSConfig{
        URL:             "https://iam.example.com/.well-known/jwks.json",
        RefreshInterval: 5 * time.Minute,
    },
    auth.WithCacheEnabled(true),
    auth.WithAuthClient(client.Auth()), // 添加 gRPC 降级
    auth.WithCircuitBreaker(&observability.CircuitBreakerConfig{
        FailureThreshold: 3,
        OpenDuration:     30 * time.Second,
    }),
)
defer jwksManager.Stop()

// 手动刷新
err = jwksManager.Refresh(ctx)

// 获取统计信息
stats := jwksManager.Stats()
log.Printf("JWKS Stats: %+v", stats)
```

### 2. 验证选项

```go
// 跳过过期检查
result, err := verifier.Verify(ctx, token, &auth.VerifyOptions{
    SkipExpiryCheck: true,
})

// 自定义 audience 验证
result, err := verifier.Verify(ctx, token, &auth.VerifyOptions{
    RequiredAudience: []string{"specific-app"},
})

// 获取额外 Claims
result, err := verifier.Verify(ctx, token, &auth.VerifyOptions{
    ExtractCustomClaims: true,
})
```

### 3. 策略选择

```go
// 强制本地验证
verifier.SetStrategy("local")

// 强制远程验证
verifier.SetStrategy("remote")

// 自动选择（默认）
verifier.SetStrategy("fallback")

// 带缓存
verifier.SetStrategy("caching")
```

## JWKS 职责链

JWKS Manager 使用职责链模式，按顺序尝试：

1. **CacheFetcher** - 内存缓存（最快）
2. **CircuitBreakerFetcher** - 熔断保护
3. **HTTPFetcher** - HTTP 获取（主要方式）
4. **GRPCEndpointFetcher** - gRPC 降级（HTTP 失败时）
5. **SeedFetcher** - 本地种子备份（兜底）

### 配置职责链

```go
jwksManager, err := auth.NewJWKSManager(cfg,
    auth.WithCacheEnabled(true),              // 启用内存缓存
    auth.WithAuthClient(client.Auth()),       // gRPC 降级
    auth.WithSeedCache("/var/cache/jwks"),    // 本地种子缓存
    auth.WithCircuitBreaker(&observability.CircuitBreakerConfig{
        FailureThreshold: 5,
        OpenDuration:     30 * time.Second,
    }),
)
```

## 性能优化

### 1. 启用缓存

```go
&JWKSConfig{
    RefreshInterval: 5 * time.Minute,  // 定期刷新
    CacheTTL:        1 * time.Hour,    // 缓存 1 小时
    FallbackOnError: true,             // 失败时使用缓存
}
```

### 2. 使用 CachingVerifyStrategy

```go
verifier.SetStrategy("caching")

// 验证结果会被缓存 5 分钟
result, err := verifier.Verify(ctx, token, &auth.VerifyOptions{
    CacheTTL: 5 * time.Minute,
})
```

### 3. 预热缓存

```go
// 启动时预热 JWKS
err := jwksManager.Refresh(ctx)
if err != nil {
    log.Printf("预热失败，将在后台刷新: %v", err)
}
```

## 错误处理

```go
import (
    sdk "github.com/FangcunMount/iam-contracts/pkg/sdk"
    "github.com/FangcunMount/iam-contracts/pkg/sdk/errors"
)

result, err := verifier.Verify(ctx, token, nil)
if err != nil {
    switch {
    case errors.IsTokenExpired(err):
        log.Println("Token 已过期，请刷新")
        // 触发刷新流程
    case errors.IsTokenInvalid(err):
        log.Println("Token 格式无效")
        // 返回 401
    case errors.IsServiceUnavailable(err):
        log.Println("验证服务不可用，稍后重试")
        // 降级处理
    default:
        log.Printf("验证失败: %v", err)
    }
    return
}

if !result.Valid {
    log.Printf("Token 无效: %s", result.Error)
    // 可能的原因：签名不匹配、audience 不匹配、issuer 不匹配等
}
```

## 监控和观测

### 统计信息

```go
// JWKS Manager 统计
stats := jwksManager.Stats()
log.Printf("刷新次数: %d", stats.RefreshCount)
log.Printf("成功次数: %d", stats.SuccessCount)
log.Printf("失败次数: %d", stats.FailureCount)
log.Printf("上次刷新: %v", stats.LastRefresh)

// Verifier 统计
verifyStats := verifier.Stats()
log.Printf("验证次数: %d", verifyStats.TotalVerifications)
log.Printf("本地验证: %d", verifyStats.LocalVerifications)
log.Printf("远程验证: %d", verifyStats.RemoteVerifications)
log.Printf("缓存命中: %d", verifyStats.CacheHits)
```

### Prometheus Metrics

```go
import "github.com/prometheus/client_golang/prometheus"

// 自定义 Metrics
verifyCounter := prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "jwt_verifications_total",
        Help: "Total JWT verifications",
    },
    []string{"source", "valid"},
)
prometheus.MustRegister(verifyCounter)

// 验证时记录
result, err := verifier.Verify(ctx, token, nil)
verifyCounter.WithLabelValues(
    string(result.Source),
    fmt.Sprintf("%t", result.Valid),
).Inc()
```

## 生产环境建议

### 配置建议

```go
&TokenVerifyConfig{
    AllowedAudience:         []string{"your-app"},
    AllowedIssuer:           "https://iam.example.com",
    ClockSkew:               time.Minute,         // 容忍 1 分钟时钟偏差
    RequireExpirationTime:   true,               // 强制要求 exp
    ForceRemoteVerification: false,              // 本地优先
}

&JWKSConfig{
    URL:             "https://iam.example.com/.well-known/jwks.json",
    GRPCEndpoint:    "iam.example.com:8081",     // gRPC 降级
    RefreshInterval: 5 * time.Minute,            // 每 5 分钟刷新
    RequestTimeout:  10 * time.Second,           // HTTP 超时 10 秒
    CacheTTL:        1 * time.Hour,              // 缓存 1 小时
    FallbackOnError: true,                       // 失败时使用缓存
}
```

### 启动流程

```go
func setupJWTVerifier(ctx context.Context, client *sdk.Client) (*auth.TokenVerifier, error) {
    // 1. 创建 JWKS Manager
    jwksManager, err := auth.NewJWKSManager(
        cfg.JWKS,
        auth.WithCacheEnabled(true),
        auth.WithAuthClient(client.Auth()),
        auth.WithSeedCache("/var/cache/iam/jwks"),
        auth.WithCircuitBreaker(cfg.CircuitBreaker),
    )
    if err != nil {
        return nil, err
    }

    // 2. 预热缓存
    if err := jwksManager.Refresh(ctx); err != nil {
        log.Printf("JWKS 预热失败，将在后台刷新: %v", err)
    }

    // 3. 创建 Verifier
    verifier := auth.NewTokenVerifier(
        cfg.TokenVerify,
        jwksManager,
        client.Auth(),
    )

    // 4. 启动监控
    go monitorJWKS(jwksManager)

    return verifier, nil
}

func monitorJWKS(manager *auth.JWKSManager) {
    ticker := time.NewTicker(time.Minute)
    defer ticker.Stop()

    for range ticker.C {
        stats := manager.Stats()
        if stats.FailureCount > 0 {
            log.Printf("JWKS failures detected: %d", stats.FailureCount)
        }
    }
}
```

## 常见问题

### Q: 如何处理 JWKS 密钥轮换？

A: SDK 会自动刷新 JWKS。配置合理的 `RefreshInterval`（如 5 分钟）即可。

### Q: 本地验证失败会怎样？

A: 如果提供了 gRPC 客户端，会自动降级到远程验证。

### Q: 如何减少对 IAM 服务的依赖？

A: 使用本地种子缓存：

```go
auth.WithSeedCache("/var/cache/iam/jwks-seed.json")
```

### Q: Token 验证性能如何？

A: 本地验证通常在 1ms 以内。使用 CachingStrategy 可进一步优化。

## 下一步

- [服务间认证](./04-service-auth.md)
- [错误处理](./06-error-handling.md)
- [可观测性](./05-observability.md)

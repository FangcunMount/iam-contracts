# 服务间认证

## 🎯 30 秒搞懂

### 金字塔架构

```text
                   ServiceAuthHelper
                (智能 Token 生命周期管理器)
                        │
        ┌───────────────┼───────────────┐
        ↓               ↓               ↓
    自动刷新        故障处理         可观测性
    (主动)          (被动)          (监控)
        ↓               ↓               ↓
   定时 Ticker     熔断+降级        Stats+Hooks
   + Jitter       指数退避         实时状态
```

### Token 生命周期

```text
时间线:
│────────────────────────────────────────────────│
0                                          TokenTTL (1h)
                                                 ↑
                                      RefreshBefore (5m)
                                                 │
      │────────────────────────────│             │
      ↑                            ↑             ↑
   首次获取                      触发刷新        过期时间
   (0s)                          (55m)          (60m)

Jitter 机制:
  实际刷新时间 = 55m ± 5% (52m - 58m 随机)
  ✅ 避免惊群效应
  ✅ 分散负载
```

### 状态转换图

```text
          ┌──────────────┐
          │  初始状态     │
          │  (No Token)  │
          └──────┬───────┘
                 │ Start()
                 ↓
          ┌──────────────┐
    ┌────→│  正常运行     │←────┐
    │     │  (Active)    │     │
    │     └──────┬───────┘     │
    │            │             │
    │  刷新成功   │ 刷新失败     │ 熔断恢复
    │            ↓             │
    │     ┌──────────────┐     │
    │     │  退避重试     │     │
    │     │  (Backoff)   │     │
    │     └──────┬───────┘     │
    │            │             │
    │  重试成功   │ 连续失败     │
    └────────────┤             │
                 ↓             │
          ┌──────────────┐     │
          │  熔断打开     │─────┘
          │  (Open)      │
          └──────────────┘
               ↑ ↓
           使用缓存 Token
           (降级保护)
```

### 刷新策略详解

```text
┌──────────────────────────────────────────────────────┐
│ RefreshStrategy                                       │
│                                                       │
│  1️⃣ JitterRatio: 0.05 (±5% 随机抖动)                  │
│     刷新时间 = 55m ± 2.75m                             │
│                                                       │
│  2️⃣ 指数退避:                                          │
│     失败1次: 30s (MinBackoff)                         │
│     失败2次: 60s (30s * 2^1)                          │
│     失败3次: 120s (30s * 2^2)                         │
│     失败4次: 240s (30s * 2^3)                         │
│     失败5次: 300s (MaxBackoff 上限)                    │
│                                                       │
│  3️⃣ 熔断保护:                                          │
│     MaxRetries: 5 (连续失败 5 次)                      │
│     CircuitOpenDuration: 1m (熔断持续 1 分钟)          │
│     熔断后: 使用已缓存 Token (降级)                     │
└──────────────────────────────────────────────────────┘
```

### 故障处理流程

```text
正常流程:
  刷新成功 → 更新 Token → 重置退避 → 继续运行
           (10ms)        (1次)

第一次失败:
  刷新失败 → 30s 后重试 → 成功则恢复
           (MinBackoff)

连续失败:
  失败1次 → 30s 后重试
  失败2次 → 60s 后重试
  失败3次 → 120s 后重试
  失败4次 → 240s 后重试
  失败5次 → 熔断打开 (1分钟)
           ↓
  使用缓存 Token (降级保护)
           ↓
  1 分钟后自动尝试恢复

兜底机制:
  ✅ Token 未过期? → 继续使用
  ❌ Token 已过期? → 返回错误
```

### 3 行代码开始

```go
// 1️⃣ 创建助手
helper, _ := sdk.NewServiceAuthHelper(
    &sdk.ServiceAuthConfig{
        ServiceID: "my-service",
        TargetAudience: []string{"target-service"},
        TokenTTL: time.Hour,
        RefreshBefore: 5 * time.Minute,
    },
    client,
)
defer helper.Stop()

// 2️⃣ 获取 Token
token, _ := helper.GetToken(ctx)

// 3️⃣ 使用 Token
conn, _ := grpc.Dial("target-service:8081",
    grpc.WithPerRPCCredentials(helper)) // 自动注入 Token
```

### 核心优势

| 特性 | 说明 | 收益 |
|-----|------|------|
| 🔄 **自动刷新** | 提前 5 分钟刷新，无需手动管理 | 零感知 Token 更新 |
| 🎲 **Jitter** | 刷新时间 ±5% 随机抖动 | 避免惊群，负载均衡 |
| 📈 **指数退避** | 30s → 60s → 120s → 240s → 300s | 保护 IAM 服务 |
| 🔌 **熔断保护** | 连续失败 5 次后熔断 1 分钟 | 快速失败，不阻塞 |
| 💾 **降级缓存** | 熔断期间使用已有 Token | 服务高可用 |
| 📊 **可观测性** | 统计信息 + 回调钩子 | 实时监控 |

---

## 📖 详细说明

### 为什么需要 ServiceAuthHelper？

**痛点对比:**

| 方式 | 手动管理 | ServiceAuthHelper |
|-----|---------|------------------|
| Token 过期处理 | ❌ 需要手动检查 | ✅ 自动刷新 |
| 刷新时机 | ❌ 难以把握 | ✅ 提前刷新 |
| 并发安全 | ❌ 需要加锁 | ✅ 内置保护 |
| 失败重试 | ❌ 硬编码逻辑 | ✅ 智能退避 |
| 惊群效应 | ❌ 同时刷新 | ✅ Jitter 分散 |
| 熔断降级 | ❌ 需要自己实现 | ✅ 内置熔断器 |

---

## 🚀 快速开始

### 基础用法

```go
import sdk "github.com/FangcunMount/iam-contracts/pkg/sdk"

// 创建 IAM 客户端
client, err := sdk.NewClient(ctx, &sdk.Config{
    Endpoint: "iam.example.com:8081",
    TLS: &sdk.TLSConfig{
        Enabled: true,
        CACert:  "/etc/iam/certs/ca.crt",
    },
})

// 创建服务间认证助手
helper, err := sdk.NewServiceAuthHelper(
    &sdk.ServiceAuthConfig{
        ServiceID:      "my-service",              // 当前服务 ID
        TargetAudience: []string{"iam-service"},   // 目标服务
        TokenTTL:       time.Hour,                 // Token 有效期
        RefreshBefore:  5 * time.Minute,           // 提前 5 分钟刷新
    },
    client,
)
defer helper.Stop()

// 使用方式 1: 获取 Token
token, err := helper.GetToken(ctx)
if err != nil {
    log.Fatal(err)
}
fmt.Println("Token:", token)

// 使用方式 2: 创建认证 Context
authCtx, err := helper.NewAuthenticatedContext(ctx)
if err != nil {
    log.Fatal(err)
}
resp, err := client.Identity().GetUser(authCtx, "user-123")

// 使用方式 3: 包装函数调用
err = helper.CallWithAuth(ctx, func(authCtx context.Context) error {
    _, err := client.Identity().GetUser(authCtx, "user-123")
    return err
})
```

## ServiceAuthConfig 配置

```go
type ServiceAuthConfig struct {
    // ServiceID 当前服务标识
    ServiceID string
    
    // TargetAudience 目标服务 audience 列表
    TargetAudience []string
    
    // TokenTTL Token 有效期
    TokenTTL time.Duration
    
    // RefreshBefore 提前刷新时间（在过期前多久刷新）
    RefreshBefore time.Duration
}
```

### 配置示例

```go
&ServiceAuthConfig{
    ServiceID:      "payment-service",
    TargetAudience: []string{"iam-service", "user-service"},
    TokenTTL:       time.Hour,
    RefreshBefore:  5 * time.Minute,  // 在过期前 5 分钟刷新
}
```

## 刷新策略配置

`ServiceAuthHelper` 支持自定义刷新策略：

```go
type RefreshStrategy struct {
    // JitterRatio 刷新时间抖动比例（0-1）
    JitterRatio float64
    
    // MinBackoff 失败后最小退避时间
    MinBackoff time.Duration
    
    // MaxBackoff 失败后最大退避时间
    MaxBackoff time.Duration
    
    // BackoffMultiplier 退避乘数
    BackoffMultiplier float64
    
    // MaxRetries 最大连续重试次数
    MaxRetries int
    
    // CircuitOpenDuration 熔断持续时间
    CircuitOpenDuration time.Duration
    
    // 回调钩子
    OnRefreshSuccess func(token string, expiresIn time.Duration)
    OnRefreshFailure func(err error, attempt int, nextRetry time.Duration)
    OnCircuitOpen    func()
    OnCircuitClose   func()
}
```

### 默认刷新策略

```go
&RefreshStrategy{
    JitterRatio:         0.1,               // ±10% 抖动
    MinBackoff:          1 * time.Second,
    MaxBackoff:          60 * time.Second,
    BackoffMultiplier:   2.0,               // 指数退避
    MaxRetries:          5,                 // 5 次失败后熔断
    CircuitOpenDuration: 30 * time.Second,  // 熔断 30 秒
}
```

### 自定义刷新策略

```go
import "github.com/FangcunMount/iam-contracts/pkg/sdk/auth"

helper, err := sdk.NewServiceAuthHelper(
    cfg,
    client,
    auth.WithRefreshStrategy(&auth.RefreshStrategy{
        JitterRatio:         0.15,              // 增加抖动
        MinBackoff:          2 * time.Second,
        MaxBackoff:          120 * time.Second,
        BackoffMultiplier:   2.5,
        MaxRetries:          3,                 // 更严格的熔断
        CircuitOpenDuration: 60 * time.Second,
        
        OnRefreshSuccess: func(token string, expiresIn time.Duration) {
            log.Printf("Token 刷新成功，有效期: %v", expiresIn)
            metrics.TokenRefreshSuccess.Inc()
        },
        
        OnRefreshFailure: func(err error, attempt int, nextRetry time.Duration) {
            log.Printf("Token 刷新失败: attempt=%d, next=%v, err=%v", 
                attempt, nextRetry, err)
            metrics.TokenRefreshFailure.Inc()
        },
        
        OnCircuitOpen: func() {
            log.Println("Token 刷新熔断器打开")
            alert.Send("ServiceAuth circuit breaker opened!")
        },
        
        OnCircuitClose: func() {
            log.Println("Token 刷新熔断器关闭")
        },
    }),
)
```

## 刷新机制

### 刷新时机

1. **定时刷新**：在 Token 过期前 `RefreshBefore` 时间刷新
2. **主动刷新**：调用 `GetToken()` 时检查是否需要刷新
3. **启动刷新**：Helper 创建时立即获取 Token

### Jitter（抖动）

避免多个服务同时刷新（惊群效应）：

```text
基础刷新时间: 55 分钟（TokenTTL=1h, RefreshBefore=5m）
Jitter 10%:   55m ± 5.5m
实际刷新:     49.5m ~ 60.5m（随机）
```

### 失败退避

连续失败时使用指数退避 + 抖动：

```text
失败次数  基础退避      实际退避（+jitter）
1        1s           0.9s ~ 1.1s
2        2s           1.8s ~ 2.2s
3        4s           3.6s ~ 4.4s
4        8s           7.2s ~ 8.8s
5        16s          14.4s ~ 17.6s
6        32s          28.8s ~ 35.2s
7+       60s (max)    54s ~ 66s
```

### 熔断保护

连续失败 `MaxRetries` 次后进入熔断状态：

```text
Normal ─┐  连续失败 5 次
        ↓
Retrying ─┐  持续失败
         ↓
CircuitOpen ─┐  熔断 30 秒后
            ↓
Normal (恢复)
```

熔断期间行为：

- ✅ 如有未过期 Token，继续使用
- ❌ 如无有效 Token，返回错误
- 🔄 熔断时间结束后自动尝试恢复

## 使用示例

### 示例 1: HTTP 请求中间件

```go
func authMiddleware(helper *auth.ServiceAuthHelper) gin.HandlerFunc {
    return func(c *gin.Context) {
        token, err := helper.GetToken(c.Request.Context())
        if err != nil {
            c.JSON(500, gin.H{"error": "failed to get service token"})
            c.Abort()
            return
        }
        
        c.Request.Header.Set("Authorization", "Bearer "+token)
        c.Next()
    }
}

// 使用中间件
r := gin.Default()
r.Use(authMiddleware(helper))
```

### 示例 2: gRPC 拦截器

```go
func serviceAuthInterceptor(helper *auth.ServiceAuthHelper) grpc.UnaryClientInterceptor {
    return func(
        ctx context.Context,
        method string,
        req, reply interface{},
        cc *grpc.ClientConn,
        invoker grpc.UnaryInvoker,
        opts ...grpc.CallOption,
    ) error {
        // 自动注入 Token
        authCtx, err := helper.NewAuthenticatedContext(ctx)
        if err != nil {
            return err
        }
        return invoker(authCtx, method, req, reply, cc, opts...)
    }
}

// 使用拦截器
conn, err := grpc.Dial(
    "target-service:8081",
    grpc.WithUnaryInterceptor(serviceAuthInterceptor(helper)),
)
```

### 示例 3: 批量调用

```go
func batchCallWithAuth(ctx context.Context, helper *auth.ServiceAuthHelper, userIDs []string) error {
    return helper.CallWithAuth(ctx, func(authCtx context.Context) error {
        for _, userID := range userIDs {
            user, err := client.Identity().GetUser(authCtx, userID)
            if err != nil {
                return err
            }
            log.Printf("User: %s", user.GetProfile().GetDisplayName())
        }
        return nil
    })
}
```

## 状态和统计

### 获取刷新状态

```go
// 当前状态
state := helper.State()
switch state {
case auth.RefreshStateNormal:
    log.Println("正常状态")
case auth.RefreshStateRetrying:
    log.Println("重试中")
case auth.RefreshStateCircuitOpen:
    log.Println("熔断中")
}

// 详细统计
stats := helper.Stats()
log.Printf("总刷新次数: %d", stats.TotalRefreshes)
log.Printf("成功次数: %d", stats.SuccessfulRefreshes)
log.Printf("失败次数: %d", stats.FailedRefreshes)
log.Printf("连续失败: %d", stats.ConsecutiveFailures)
log.Printf("上次刷新: %v", stats.LastRefreshTime)
log.Printf("上次错误: %v", stats.LastRefreshError)
log.Printf("当前状态: %s", stats.State)
```

### 监控和告警

```go
// 定期检查状态
go func() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        stats := helper.Stats()
        
        // 记录 Metrics
        metrics.TokenRefreshTotal.Set(float64(stats.TotalRefreshes))
        metrics.TokenRefreshFailures.Set(float64(stats.FailedRefreshes))
        metrics.TokenRefreshConsecutiveFailures.Set(float64(stats.ConsecutiveFailures))
        
        // 告警
        if stats.State == auth.RefreshStateCircuitOpen {
            alert.Send("ServiceAuth circuit breaker opened!")
        }
        
        if stats.ConsecutiveFailures >= 3 {
            alert.Send(fmt.Sprintf("ServiceAuth failing: %d consecutive failures", 
                stats.ConsecutiveFailures))
        }
    }
}()
```

## 错误处理

```go
import "github.com/FangcunMount/iam-contracts/pkg/sdk/errors"

token, err := helper.GetToken(ctx)
if err != nil {
    switch {
    case errors.IsServiceUnavailable(err):
        log.Println("IAM 服务不可用，使用降级策略")
        // 降级处理
        
    case strings.Contains(err.Error(), "circuit breaker open"):
        log.Println("熔断中，稍后重试")
        // 等待熔断恢复
        
    case errors.IsUnauthorized(err):
        log.Println("服务认证失败，检查配置")
        // 检查 ServiceID 和 TargetAudience
        
    default:
        log.Printf("获取 Token 失败: %v", err)
    }
    return
}
```

## 生产环境最佳实践

### 1. 合理配置刷新时间

```go
&ServiceAuthConfig{
    TokenTTL:      time.Hour,      // Token 有效期 1 小时
    RefreshBefore: 5 * time.Minute, // 提前 5 分钟刷新（8.3% 的有效期）
}

// 推荐：RefreshBefore = TokenTTL * 0.1 ~ 0.2
```

### 2. 启用回调监控

```go
helper, err := sdk.NewServiceAuthHelper(
    cfg,
    client,
    auth.WithRefreshStrategy(&auth.RefreshStrategy{
        OnRefreshSuccess: func(token string, expiresIn time.Duration) {
            metrics.TokenRefreshSuccess.Inc()
            log.Printf("Token 刷新成功: expires_in=%v", expiresIn)
        },
        
        OnRefreshFailure: func(err error, attempt int, nextRetry time.Duration) {
            metrics.TokenRefreshFailure.Inc()
            log.Printf("Token 刷新失败: attempt=%d, err=%v, next_retry=%v",
                attempt, err, nextRetry)
            
            // 严重失败时告警
            if attempt >= 3 {
                alert.Send(fmt.Sprintf("ServiceAuth failing: %d attempts", attempt))
            }
        },
        
        OnCircuitOpen: func() {
            metrics.TokenRefreshCircuitOpen.Set(1)
            alert.Send("CRITICAL: ServiceAuth circuit breaker opened!")
        },
        
        OnCircuitClose: func() {
            metrics.TokenRefreshCircuitOpen.Set(0)
            log.Println("ServiceAuth circuit breaker closed")
        },
    }),
)
```

### 3. 优雅关闭

```go
func gracefulShutdown(helper *auth.ServiceAuthHelper) {
    // 监听关闭信号
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    
    <-sigCh
    log.Println("Shutting down...")
    
    // 停止 Helper
    helper.Stop()
    
    // 等待正在进行的刷新完成
    time.Sleep(time.Second)
    
    log.Println("Shutdown complete")
}
```

### 4. 健康检查

```go
func healthCheck(helper *auth.ServiceAuthHelper) bool {
    stats := helper.Stats()
    
    // 检查状态
    if stats.State == auth.RefreshStateCircuitOpen {
        return false
    }
    
    // 检查连续失败次数
    if stats.ConsecutiveFailures >= 3 {
        return false
    }
    
    // 检查是否有有效 Token
    token, err := helper.GetToken(context.Background())
    if err != nil || token == "" {
        return false
    }
    
    return true
}

// HTTP 健康检查端点
http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
    if healthCheck(helper) {
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
    } else {
        w.WriteHeader(http.StatusServiceUnavailable)
        json.NewEncoder(w).Encode(map[string]string{"status": "unhealthy"})
    }
})
```

## 常见问题

### Q: Token 刷新失败会怎样？

A: 如果还有未过期的 Token，会继续使用。连续失败 `MaxRetries` 次后进入熔断状态。

### Q: 如何减少刷新频率？

A: 增加 `TokenTTL` 和减少 `JitterRatio`：

```go
TokenTTL:      24 * time.Hour,  // 24 小时
RefreshBefore: 1 * time.Hour,    // 提前 1 小时
JitterRatio:   0.05,             // ±5% 抖动
```

### Q: 多个 Helper 会互相影响吗？

A: 不会。每个 Helper 独立管理自己的 Token 和刷新状态。

### Q: 如何在测试中使用？

A: 使用 mock 客户端：

```go
// 测试时禁用自动刷新
helper, _ := auth.NewServiceAuthHelperWithCallbacks(
    cfg,
    mockClient,
    nil, // 不需要回调
    nil,
)
```

## 下一步

- [可观测性](./05-observability.md)
- [错误处理](./06-error-handling.md)
- [高级重试配置](./07-advanced-retry.md)

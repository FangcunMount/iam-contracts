# IAM SDK for Go

[![Go Reference](https://pkg.go.dev/badge/github.com/FangcunMount/iam-contracts/pkg/sdk.svg)](https://pkg.go.dev/github.com/FangcunMount/iam-contracts/pkg/sdk)
[![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

IAM 服务的 Go 客户端 SDK，采用**模块化设计**和**企业级最佳实践**，提供统一的 API 访问入口。

## 🚀 功能特性

### 核心功能

- ✅ **模块化架构**：config / transport / auth / identity / errors / observability 子包独立可用
- ✅ **统一客户端入口**：通过 `sdk.Client` 访问所有 IAM 服务
- ✅ **mTLS 双向认证**：内置 TLS 1.2/1.3 配置，支持证书文件/PEM 内容
- ✅ **JWT 本地验证**：JWKS 缓存 + HTTP/gRPC 双通道 + 本地种子备份
- ✅ **服务间认证**：自动 Token 刷新，带 jitter/熔断/退避的生产级实现

### 可靠性

- ✅ **方法级重试/超时**：支持按 gRPC 方法定制重试策略和超时时间
- ✅ **自定义可重试码**：幂等/非幂等操作分离，支持自定义重试判断函数
- ✅ **熔断保护**：内置熔断器，自动限流保护
- ✅ **连接池**：gRPC Keepalive，自动重连
- ✅ **负载均衡**：支持 round_robin / pick_first

### 可观测性

- ✅ **Prometheus Metrics**：请求计数、延迟、错误率
- ✅ **OpenTelemetry Tracing**：分布式链路追踪桥接
- ✅ **请求 ID 传播**：自动生成和传递 X-Request-ID
- ✅ **统一错误分类**：gRPC 错误自动分类为业务错误（Auth/NotFound/Validation 等）

### 开发体验

- ✅ **统一错误处理**：语义化错误类型和错误匹配器
- ✅ **配置加载**：环境变量 / Viper / 代码配置多源支持
- ✅ **设计模式**：Chain of Responsibility (JWKS) / Strategy (TokenVerifier)
- ✅ **丰富示例**：basic / mTLS / verifier / service_auth 完整示例

## 安装

```bash
go get github.com/FangcunMount/iam-contracts/pkg/sdk
```

## 📦 包结构

```text
pkg/sdk/
├── sdk.go                     # 主入口（类型别名 + 便捷函数）
├── config/                    # 配置模块
│   ├── config.go              # Config、TLSConfig、RetryConfig、CircuitBreakerConfig
│   ├── loader.go              # 环境变量、Viper 加载器
│   ├── options.go             # ClientOption 函数式选项
│   └── errors.go              # 配置验证错误
├── transport/                 # 传输层（企业级可靠性）
│   ├── dial.go                # gRPC 连接建立、TLS 配置、ServiceConfig
│   ├── interceptors.go        # RequestID、Metadata 注入
│   ├── retry.go               # 方法级重试/超时配置、自定义重试判断
│   └── errors.go              # 错误包装拦截器、错误处理链
├── observability/             # 可观测性（生产级监控）
│   ├── observability.go       # MetricsCollector、TracingHook 接口
│   ├── prometheus.go          # Prometheus 实现
│   ├── otel.go                # OpenTelemetry 桥接
│   └── circuit_breaker.go     # 熔断器实现
├── errors/                    # 错误处理（统一错误分类）
│   └── errors.go              # IAMError、ErrorCategory、ErrorMatcher、ErrorHandler
├── auth/                      # 认证模块（JWT + 服务间认证）
│   ├── client.go              # AuthnClient（VerifyToken、IssueServiceToken）
│   ├── verifier.go            # TokenVerifier（Strategy 模式：Local/Remote/Fallback/Caching）
│   ├── jwks.go                # JWKSManager（Chain of Responsibility：Cache→HTTP→gRPC→Seed）
│   └── service_auth.go        # ServiceAuthHelper（自动刷新 + jitter + 熔断 + 退避）
├── identity/                  # 身份模块
│   ├── client.go              # IdentityClient（用户/角色/部门 CRUD）
│   └── guardianship.go        # GuardianshipClient（监护关系）
└── _examples/                 # 完整示例代码
    ├── basic/                 # 基础用法
    ├── mtls/                  # mTLS 双向认证
    ├── verifier/              # JWT 本地验证
    ├── service_auth/          # 服务间认证
    ├── retry/                 # 方法级重试配置
    └── observability/         # Metrics + Tracing
```

### 设计亮点

| 模块 | 设计模式 | 说明 |
| ------ | --------- | ------ |
| `auth/jwks.go` | **Chain of Responsibility** | Cache → CircuitBreaker → HTTP → gRPC → Seed，自动降级 |
| `auth/verifier.go` | **Strategy** | LocalVerify / RemoteVerify / FallbackVerify / CachingVerify |
| `transport/retry.go` | **Builder + Predicate** | 方法级配置 + 自定义重试判断 |
| `errors/errors.go` | **Matcher + Handler** | 错误匹配器 + 链式错误处理器 |
| `observability/*` | **Hook + Adapter** | MetricsCollector / TracingHook 抽象层 |

## 快速开始

### 基础用法

```go
import sdk "github.com/FangcunMount/iam-contracts/pkg/sdk"

func main() {
    ctx := context.Background()

    client, err := sdk.NewClient(ctx, &sdk.Config{
        Endpoint: "localhost:8081",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // 使用认证服务
    resp, err := client.Auth().VerifyToken(ctx, &authnv1.VerifyTokenRequest{
        AccessToken: "xxx",
    })

    // 使用身份服务
    user, err := client.Identity().GetUser(ctx, "user-123")

    // 使用监护关系服务
    result, err := client.Guardianship().IsGuardian(ctx, "user-1", "child-1")
}
```

### 从环境变量加载配置

```go
// 设置环境变量:
// IAM_ENDPOINT=iam.example.com:8081
// IAM_TLS_CA_CERT=/etc/iam/certs/ca.crt

cfg, err := sdk.ConfigFromEnv()
if err != nil {
    log.Fatal(err)
}

client, err := sdk.NewClient(ctx, cfg)
```

### 生产环境（mTLS）

```go
client, err := sdk.NewClient(ctx, &sdk.Config{
    Endpoint: "iam.example.com:8081",
    TLS: &sdk.TLSConfig{
        Enabled:    true,
        CACert:     "/path/to/ca.crt",
        ClientCert: "/path/to/client.crt",
        ClientKey:  "/path/to/client.key",
        ServerName: "iam.example.com",
        MinVersion: tls.VersionTLS12,
    },
    Retry: &sdk.RetryConfig{
        Enabled:     true,
        MaxAttempts: 3,
    },
})
```

## 子模块使用

### 直接使用子包

如果只需要部分功能，可以直接导入子包：

```go
import (
    "github.com/FangcunMount/iam-contracts/pkg/sdk/config"
    "github.com/FangcunMount/iam-contracts/pkg/sdk/auth"
    "github.com/FangcunMount/iam-contracts/pkg/sdk/errors"
)

// 使用 config 包
cfg, _ := config.FromEnv()

// 使用 errors 包
if errors.IsNotFound(err) {
    // 处理未找到错误
}
```

### Token 验证器

```go
import sdk "github.com/FangcunMount/iam-contracts/pkg/sdk"

verifier, err := sdk.NewTokenVerifier(
    &sdk.TokenVerifyConfig{
        AllowedAudience: []string{"my-app"},
        AllowedIssuer:   "https://iam.example.com",
    },
    &sdk.JWKSConfig{
        URL:             "https://iam.example.com/.well-known/jwks.json",
        RefreshInterval: 5 * time.Minute,
    },
    client, // 可选：用于远程验证降级
)

result, err := verifier.Verify(ctx, token, nil)
if result.Valid {
    fmt.Printf("用户 ID: %s\n", result.Claims.UserID)
}
```

### 服务间认证

```go
import sdk "github.com/FangcunMount/iam-contracts/pkg/sdk"

helper, err := sdk.NewServiceAuthHelper(&sdk.ServiceAuthConfig{
    ServiceID:      "qs-service",
    TargetAudience: []string{"iam-service"},
    TokenTTL:       time.Hour,
    RefreshBefore:  5 * time.Minute,
}, client)
defer helper.Stop()

// 自动注入认证上下文
authCtx, _ := helper.NewAuthenticatedContext(ctx)
resp, _ := client.Identity().GetUser(authCtx, "user-123")
```

### 错误处理

```go
import (
    sdk "github.com/FangcunMount/iam-contracts/pkg/sdk"
    "github.com/FangcunMount/iam-contracts/pkg/sdk/errors"
)

resp, err := client.Identity().GetUser(ctx, "user-123")
if err != nil {
    switch {
    case errors.IsNotFound(err):
        log.Println("用户不存在")
    case errors.IsUnauthorized(err):
        log.Println("未认证")
    case errors.IsPermissionDenied(err):
        log.Println("权限不足")
    case errors.IsServiceUnavailable(err):
        log.Println("服务不可用")
    default:
        log.Printf("其他错误: %v", err)
    }
}
```

### 可观测性

```go
import (
    sdk "github.com/FangcunMount/iam-contracts/pkg/sdk"
    "github.com/FangcunMount/iam-contracts/pkg/sdk/observability"
)

// 实现 MetricsCollector 接口
type myMetrics struct{}
func (m *myMetrics) RecordRequest(method, code string, duration time.Duration) {
    // 上报到 Prometheus
}

// 添加拦截器
client, _ := sdk.NewClient(ctx, cfg,
    sdk.WithUnaryInterceptors(
        observability.MetricsUnaryInterceptor(&myMetrics{}),
    ),
)
```

## 配置说明

### 环境变量

| 变量名 | 说明 | 默认值 |
| -------- | ------ | -------- |
| `IAM_ENDPOINT` | gRPC 服务地址 | (必填) |
| `IAM_TIMEOUT` | 请求超时 | 30s |
| `IAM_TLS_ENABLED` | 启用 TLS | true |
| `IAM_TLS_CA_CERT` | CA 证书路径 | - |
| `IAM_TLS_CLIENT_CERT` | 客户端证书路径 | - |
| `IAM_TLS_CLIENT_KEY` | 客户端私钥路径 | - |
| `IAM_RETRY_ENABLED` | 启用重试 | true |
| `IAM_RETRY_MAX_ATTEMPTS` | 最大重试次数 | 3 |
| `IAM_JWKS_URL` | JWKS 端点 | - |

### YAML 配置示例

```yaml
iam:
  endpoint: "iam.example.com:8081"
  timeout: 30s
  
  tls:
    enabled: true
    ca_cert: "/etc/iam/certs/ca.crt"
    client_cert: "/etc/iam/certs/client.crt"
    client_key: "/etc/iam/certs/client.key"
    server_name: "iam.example.com"

  retry:
    enabled: true
    max_attempts: 3

  jwks:
    url: "https://iam.example.com/.well-known/jwks.json"
    refresh_interval: 5m
```

## 生产环境完整示例

以下是一个完整的生产环境配置示例，涵盖 mTLS、JWKS、服务间认证、熔断器、重试/超时、可观测性等所有特性：

```go
package main

import (
    "context"
    "crypto/tls"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"

    sdk "github.com/FangcunMount/iam-contracts/pkg/sdk"
    "github.com/FangcunMount/iam-contracts/pkg/sdk/auth"
    "github.com/FangcunMount/iam-contracts/pkg/sdk/config"
    "github.com/FangcunMount/iam-contracts/pkg/sdk/errors"
    "github.com/FangcunMount/iam-contracts/pkg/sdk/observability"
    "github.com/FangcunMount/iam-contracts/pkg/sdk/transport"
    "github.com/prometheus/client_golang/prometheus"
)

// =============================================================================
// 1. Prometheus Metrics 实现
// =============================================================================

type prometheusMetrics struct {
    requestCounter  *prometheus.CounterVec
    requestDuration *prometheus.HistogramVec
}

func newPrometheusMetrics() *prometheusMetrics {
    m := &prometheusMetrics{
        requestCounter: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "iam_sdk_requests_total",
                Help: "Total number of IAM SDK requests",
            },
            []string{"method", "code"},
        ),
        requestDuration: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Name:    "iam_sdk_request_duration_seconds",
                Help:    "IAM SDK request duration in seconds",
                Buckets: prometheus.DefBuckets,
            },
            []string{"method"},
        ),
    }
    prometheus.MustRegister(m.requestCounter, m.requestDuration)
    return m
}

func (m *prometheusMetrics) RecordRequest(method, code string, duration time.Duration) {
    m.requestCounter.WithLabelValues(method, code).Inc()
    m.requestDuration.WithLabelValues(method).Observe(duration.Seconds())
}

// =============================================================================
// 2. 完整配置
// =============================================================================

func buildProductionConfig() *config.Config {
    return &config.Config{
        Endpoint:    os.Getenv("IAM_ENDPOINT"), // e.g., "iam.example.com:8081"
        DialTimeout: 10 * time.Second,
        Timeout:     30 * time.Second,

        // mTLS 配置
        TLS: &config.TLSConfig{
            Enabled:    true,
            CACert:     os.Getenv("IAM_TLS_CA_CERT"),
            ClientCert: os.Getenv("IAM_TLS_CLIENT_CERT"),
            ClientKey:  os.Getenv("IAM_TLS_CLIENT_KEY"),
            ServerName: os.Getenv("IAM_TLS_SERVER_NAME"),
            MinVersion: tls.VersionTLS13,
        },

        // 全局重试配置
        Retry: &config.RetryConfig{
            Enabled:           true,
            MaxAttempts:       3,
            InitialBackoff:    100 * time.Millisecond,
            MaxBackoff:        5 * time.Second,
            BackoffMultiplier: 2.0,
            RetryableCodes:    []string{"UNAVAILABLE", "RESOURCE_EXHAUSTED", "ABORTED"},
        },

        // JWKS 配置（支持 HTTP + gRPC 降级 + 本地种子缓存）
        JWKS: &config.JWKSConfig{
            URL:             os.Getenv("IAM_JWKS_URL"), // e.g., "https://iam.example.com/.well-known/jwks.json"
            RefreshInterval: 5 * time.Minute,
            Timeout:         10 * time.Second,
            GRPCFallback:    true,                                     // 启用 gRPC 降级
            SeedCachePath:   "/var/cache/iam-sdk/jwks-seed-cache.json", // 本地种子缓存
        },

        // Keepalive 配置
        Keepalive: &config.KeepaliveConfig{
            Time:                30 * time.Second,
            Timeout:             10 * time.Second,
            PermitWithoutStream: true,
        },

        // 熔断器配置
        CircuitBreaker: &config.CircuitBreakerConfig{
            FailureThreshold: 5,
            OpenDuration:     30 * time.Second,
            HalfOpenRequests: 3,
            SuccessThreshold: 2,
        },
    }
}

// =============================================================================
// 3. 按方法定制重试/超时
// =============================================================================

func buildMethodConfigs() *transport.RetryConfig {
    return &transport.RetryConfig{
        MaxRetries:     3,
        InitialBackoff: 100 * time.Millisecond,
        MaxBackoff:     5 * time.Second,
        Multiplier:     2.0,
        Jitter:         0.2,
        MethodConfigs: map[string]transport.MethodConfig{
            // Token 验证：快速超时，不重试（幂等但时间敏感）
            "/iam.authn.v1.AuthnService/VerifyToken": {
                Timeout:    2 * time.Second,
                MaxRetries: 0,
            },
            // 用户创建：较长超时，少量重试
            "/iam.identity.v1.IdentityService/CreateUser": {
                Timeout:        10 * time.Second,
                MaxRetries:     2,
                RetryableCodes: []string{"UNAVAILABLE"},
            },
            // 用户查询：中等超时，标准重试
            "/iam.identity.v1.IdentityService/GetUser": {
                Timeout:    5 * time.Second,
                MaxRetries: 3,
            },
            // 批量查询：较长超时
            "/iam.identity.v1.IdentityService/BatchGetUsers": {
                Timeout:    15 * time.Second,
                MaxRetries: 2,
            },
        },
    }
}

// =============================================================================
// 4. 主程序
// =============================================================================

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // 监听终止信号
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    go func() {
        <-sigCh
        log.Println("Shutting down...")
        cancel()
    }()

    // 1. 构建配置
    cfg := buildProductionConfig()

    // 2. 初始化 Metrics
    metrics := newPrometheusMetrics()

    // 3. 初始化熔断器
    circuitBreaker := observability.NewCircuitBreaker(cfg.CircuitBreaker)

    // 4. 构建方法级重试配置
    retryCfg := buildMethodConfigs()

    // 5. 创建客户端（注入拦截器）
    client, err := sdk.NewClient(ctx, cfg,
        config.WithUnaryInterceptors(
            transport.RequestIDInterceptor(),                         // 请求 ID
            observability.MetricsUnaryInterceptor(metrics),           // Metrics
            observability.CircuitBreakerInterceptor(circuitBreaker),  // 熔断器
            transport.TimeoutInterceptor(retryCfg),                   // 方法级超时
            transport.RetryInterceptor(retryCfg),                     // 方法级重试
        ),
    )
    if err != nil {
        log.Fatalf("Failed to create IAM client: %v", err)
    }
    defer client.Close()

    // 6. 初始化 JWKS 管理器（带本地种子缓存 + gRPC 降级）
    jwksManager, err := auth.NewJWKSManager(cfg.JWKS, client.Auth())
    if err != nil {
        log.Fatalf("Failed to create JWKS manager: %v", err)
    }
    defer jwksManager.Stop()

    // 7. 初始化 Token 验证器
    verifier := auth.NewTokenVerifier(
        &auth.TokenVerifyConfig{
            AllowedAudience: []string{"my-service"},
            AllowedIssuer:   "https://iam.example.com",
            ClockSkew:       time.Minute,
        },
        jwksManager,
        client.Auth(), // 降级到远程验证
    )

    // 8. 初始化服务间认证助手
    serviceAuth, err := auth.NewServiceAuthHelper(&auth.ServiceAuthConfig{
        ServiceID:      "my-service",
        TargetAudience: []string{"iam-service"},
        TokenTTL:       time.Hour,
        RefreshBefore:  5 * time.Minute,
    }, client.Auth())
    if err != nil {
        log.Fatalf("Failed to create service auth helper: %v", err)
    }
    defer serviceAuth.Stop()

    // ==========================================================================
    // 使用示例
    // ==========================================================================

    // 示例 1: 验证外部 Token
    token := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..." // 用户传入的 token
    result, err := verifier.Verify(ctx, token, nil)
    if err != nil {
        handleError(err)
        return
    }
    if result.Valid {
        log.Printf("Token valid, user: %s, roles: %v", result.Claims.UserID, result.Claims.Roles)
    }

    // 示例 2: 服务间调用（自动注入认证 Token）
    authCtx, err := serviceAuth.NewAuthenticatedContext(ctx)
    if err != nil {
        handleError(err)
        return
    }
    user, err := client.Identity().GetUser(authCtx, "user-123")
    if err != nil {
        handleError(err)
        return
    }
    log.Printf("User: %s, Status: %s", user.GetProfile().GetDisplayName(), user.GetStatus())

    // 示例 3: 监护关系检查
    isGuardian, err := client.Guardianship().IsGuardian(ctx, "parent-123", "child-456")
    if err != nil {
        handleError(err)
        return
    }
    log.Printf("Is guardian: %v", isGuardian)

    log.Println("IAM SDK initialized successfully")
    <-ctx.Done()
}

// =============================================================================
// 5. 统一错误处理
// =============================================================================

func handleError(err error) {
    // 错误分类
    kind := errors.Classify(err)
    log.Printf("Error kind: %s", kind)

    switch {
    case errors.IsNotFound(err):
        log.Println("Resource not found")
    case errors.IsUnauthorized(err):
        log.Println("Authentication failed, please re-login")
    case errors.IsPermissionDenied(err):
        log.Println("Permission denied")
    case errors.IsTokenExpired(err):
        log.Println("Token expired, please refresh")
    case errors.IsServiceUnavailable(err):
        log.Println("Service temporarily unavailable, please retry later")
    case errors.IsRateLimited(err):
        log.Println("Rate limited, please slow down")
    case errors.IsRetryable(err):
        log.Println("Transient error, can retry")
    default:
        log.Printf("Unknown error: %v", err)
    }

    // 获取错误码
    if code := errors.ErrorCode(err); code != "" {
        log.Printf("Error code: %s", code)
    }
}
```

### 生产环境 YAML 配置

```yaml
iam:
  endpoint: "iam.example.com:8081"
  dial_timeout: 10s
  timeout: 30s

  tls:
    enabled: true
    ca_cert: "/etc/iam/certs/ca.crt"
    client_cert: "/etc/iam/certs/client.crt"
    client_key: "/etc/iam/certs/client.key"
    server_name: "iam.example.com"
    min_version: "1.3"

  retry:
    enabled: true
    max_attempts: 3
    initial_backoff: 100ms
    max_backoff: 5s
    backoff_multiplier: 2.0
    retryable_codes:
      - UNAVAILABLE
      - RESOURCE_EXHAUSTED
      - ABORTED

  jwks:
    url: "https://iam.example.com/.well-known/jwks.json"
    refresh_interval: 5m
    timeout: 10s
    grpc_fallback: true
    seed_cache_path: "/var/cache/iam-sdk/jwks-seed-cache.json"

  keepalive:
    time: 30s
    timeout: 10s
    permit_without_stream: true

  circuit_breaker:
    failure_threshold: 5
    open_duration: 30s
    half_open_requests: 3
    success_threshold: 2
```

## 架构设计

```text
┌─────────────────────────────────────────────────────────────────┐
│                        sdk.Client                               │
├─────────────────────────────────────────────────────────────────┤
│  Auth()          Identity()        Guardianship()               │
│  (AuthnClient)   (IdentityClient)  (GuardianshipClient)         │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Interceptor Chain                          │
├─────────────────────────────────────────────────────────────────┤
│  RequestID → Metrics → CircuitBreaker → Timeout → Retry         │
│            → Tracing → ErrorWrapping → ErrorHandler             │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      gRPC Transport                             │
├─────────────────────────────────────────────────────────────────┤
│  mTLS │ Keepalive │ LoadBalancer │ ServiceConfig                │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      IAM Services                               │
├─────────────────────────────────────────────────────────────────┤
│  AuthnService │ IdentityService │ GuardianshipService           │
└─────────────────────────────────────────────────────────────────┘
```

## 📚 完整文档

详细文档请查看 [docs/](./docs/) 目录：

### 快速导航

| 文档 | 说明 |
| ------ | ------ |
| [快速开始](./docs/01-quick-start.md) | 安装、基础示例、常见配置 |
| [配置详解](./docs/02-configuration.md) | 所有配置选项的完整说明 |
| [JWT 验证](./docs/03-jwt-verification.md) | 本地 JWT 验证、JWKS 管理 |
| [服务间认证](./docs/04-service-auth.md) | 自动 Token 刷新、熔断保护 |
| [可观测性](./docs/05-observability.md) | Metrics、Tracing、监控 |
| [错误处理](./docs/06-error-handling.md) | 统一错误处理最佳实践 |
| [方法级重试](./docs/07-advanced-retry.md) | 按方法定制重试策略 |

### 示例代码

完整可运行的示例位于 [_examples/](./examples/) 目录：

- `basic/` - 基础用法示例
- `mtls/` - mTLS 双向认证
- `verifier/` - JWT 本地验证
- `service_auth/` - 服务间认证
- `retry/` - 方法级重试配置
- `observability/` - Metrics 和 Tracing

## 🎯 核心特性详解

### 1. JWKS 职责链（Chain of Responsibility）

```go
// JWKS 获取职责链：Cache → HTTP → gRPC → Seed
jwksManager, _ := auth.NewJWKSManager(cfg,
    auth.WithCacheEnabled(true),          // 1. 内存缓存
    auth.WithAuthClient(client.Auth()),   // 2. gRPC 降级
    auth.WithSeedCache("/var/cache/jwks"), // 3. 本地种子
)
```

**优势**：

- 多级降级，高可用
- 自动切换，无需人工干预
- 支持本地种子备份

### 2. TokenVerifier 策略模式（Strategy）

```go
// 自动选择最佳验证策略
verifier := auth.NewTokenVerifier(cfg, jwksManager, client.Auth())

// 策略：LocalVerify → RemoteVerify → CachingVerify
result, _ := verifier.Verify(ctx, token, nil)
```

**优势**：

- 本地验证快速（<1ms）
- 远程降级保证可用性
- 结果缓存优化性能

### 3. ServiceAuthHelper 生产级实现

```go
helper, _ := sdk.NewServiceAuthHelper(cfg, client,
    auth.WithRefreshStrategy(&auth.RefreshStrategy{
        JitterRatio:         0.1,   // ±10% 抖动防惊群
        MaxRetries:          5,     // 5 次失败后熔断
        CircuitOpenDuration: 30s,   // 熔断 30 秒
        OnRefreshFailure: func(err error, attempt int, nextRetry time.Duration) {
            log.Printf("Token refresh failed: %v", err)
        },
    }),
)
```

**特性**：

- ✅ Jitter 防惊群
- ✅ 指数退避 + 抖动
- ✅ 熔断保护
- ✅ 降级处理
- ✅ 可观测性

### 4. 统一错误分类

```go
import "github.com/FangcunMount/iam-contracts/pkg/sdk/errors"

// 错误分类
details := errors.Analyze(err)
log.Printf("Category: %s", details.Category)       // auth/network/validation
log.Printf("Action: %s", details.SuggestedAction)  // retry/reauth/bad_request

// 错误匹配
if errors.AuthErrors.Match(err) {
    // 处理认证错误
}
```

**优势**：

- 语义化错误类型
- 自动分类和建议
- 统一错误处理

### 5. 方法级重试/超时

```go
configs := transport.NewMethodConfigs()

// 按方法定制策略
configs.SetMethodConfig("/iam.authn.v1.AuthnService/VerifyToken", &transport.MethodConfig{
    Timeout:    2 * time.Second,  // 快速超时
    MaxRetries: 0,                // 不重试
})

configs.SetMethodConfig("/iam.identity.v1.IdentityService/CreateUser", &transport.MethodConfig{
    Timeout:        10 * time.Second,
    MaxRetries:     2,
    RetryableCodes: []string{"UNAVAILABLE"},
})
```

**优势**：

- 精细控制
- 幂等/非幂等分离
- 自定义重试判断

## 🚀 性能优化建议

### 1. JWT 验证优化

```go
// ✅ 启用本地验证 + 缓存
verifier := auth.NewTokenVerifier(cfg, jwksManager, client)
verifier.SetStrategy("caching") // 验证结果缓存 5 分钟

// ✅ JWKS 配置
&JWKSConfig{
    RefreshInterval: 5 * time.Minute,  // 定期刷新
    CacheTTL:        1 * time.Hour,    // 长期缓存
    FallbackOnError: true,             // 失败时用缓存
}
```

### 2. 连接复用

```go
// ✅ 全局共享一个 Client
var globalClient *sdk.Client

func init() {
    globalClient, _ = sdk.NewClient(context.Background(), cfg)
}

// ❌ 避免每次请求创建新 Client
func badHandler(w http.ResponseWriter, r *http.Request) {
    client, _ := sdk.NewClient(r.Context(), cfg) // 不要这样做！
    defer client.Close()
}
```

### 3. 并发控制

```go
// ✅ 使用 Keepalive
&KeepaliveConfig{
    Time:                30 * time.Second,
    Timeout:             10 * time.Second,
    PermitWithoutStream: true,
}

// ✅ 设置合理的超时
&Config{
    Timeout:     30 * time.Second,  // 全局超时
    DialTimeout: 10 * time.Second,  // 连接超时
}
```

## 🔒 安全最佳实践

### 1. 生产环境必须使用 TLS

```go
&TLSConfig{
    Enabled:    true,
    CACert:     "/etc/iam/certs/ca.crt",
    ClientCert: "/etc/iam/certs/client.crt",  // mTLS
    ClientKey:  "/etc/iam/certs/client.key",
    ServerName: "iam.example.com",
    MinVersion: tls.VersionTLS13,  // 强制 TLS 1.3
}
```

### 2. 敏感信息不要硬编码

```go
// ❌ 不要这样做
&Config{
    TLS: &TLSConfig{
        ClientKeyPEM: []byte("-----BEGIN PRIVATE KEY-----..."),
    },
}

// ✅ 使用环境变量或 Secret 管理
cfg, _ := sdk.ConfigFromEnv()
```

### 3. 限制 Token 有效期

```go
&ServiceAuthConfig{
    TokenTTL:      time.Hour,      // 不要设置过长
    RefreshBefore: 5 * time.Minute,
}
```

## 📊 监控指标

推荐收集的 Metrics：

| 指标 | 说明 | 标签 |
| ------ | ------ | ------ |
| `iam_sdk_requests_total` | 请求总数 | `method`, `code` |
| `iam_sdk_request_duration_seconds` | 请求延迟 | `method` |
| `iam_sdk_requests_in_flight` | 并发请求数 | - |
| `iam_sdk_errors_total` | 错误总数 | `method`, `category` |
| `iam_sdk_circuit_breaker_state` | 熔断器状态 | `name` |
| `iam_sdk_jwks_refresh_total` | JWKS 刷新次数 | `source` |
| `iam_sdk_token_refresh_total` | Token 刷新次数 | `result` |

详见 [可观测性文档](./docs/05-observability.md)

## 🐛 故障排查

### 常见问题

#### 1. 连接超时

```bash
Error: dial iam.example.com:8081: i/o timeout
```

**解决方案**：

- 检查网络连通性
- 增加 `DialTimeout`
- 检查防火墙规则

#### 2. TLS 验证失败

```bash
Error: x509: certificate signed by unknown authority
```

**解决方案**：

- 检查 CA 证书路径
- 检查证书有效期
- 测试环境可临时设置 `InsecureSkipVerify: true`

#### 3. Token 刷新失败

```bash
Error: service_auth: circuit breaker opened after 5 failures
```

**解决方案**：

- 检查 ServiceID 和 TargetAudience 配置
- 查看 IAM 服务日志
- 增加 `MaxRetries` 或 `CircuitOpenDuration`

详见 [故障排查指南](./docs/10-troubleshooting.md)

## 🤝 贡献

欢迎贡献！请参考：

- [贡献指南](../../CONTRIBUTING.md)
- [开发文档](../../docs/development/)

## 📄 License

MIT License

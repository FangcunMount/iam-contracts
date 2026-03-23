# 配置详解

## 🎯 30 秒搞懂

### 金字塔模型

```text
                    Config (配置根)
                        │
        ┌───────────────┼───────────────┐
        ↓               ↓               ↓
    基础配置          连接配置         可靠性配置
    Endpoint         TLS              Retry
    Timeout          Keepalive        CircuitBreaker
                                      
        ↓               ↓               ↓
    高级功能         可观测性          特性开关
    JWKS             Metrics          LoadBalancer
    Metadata         Tracing          
```

### 配置层次

```text
┌─────────────────────────────────────────────────┐
│ Level 1: 必填配置 (开始使用)                      │
│  • Endpoint: "iam.example.com:8081"             │
└─────────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────┐
│ Level 2: 安全配置 (生产环境)                      │
│  • TLS: {CACert, ClientCert, ClientKey}        │
└─────────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────┐
│ Level 3: 可靠性配置 (企业级)                      │
│  • Retry: {MaxAttempts, Backoff}               │
│  • CircuitBreaker: {FailureThreshold}          │
│  • Keepalive: {Time, Timeout}                  │
└─────────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────┐
│ Level 4: 高级特性 (按需启用)                      │
│  • JWKS: 本地 JWT 验证                          │
│  • Observability: Metrics/Tracing              │
│  • LoadBalancer: 负载均衡策略                    │
└─────────────────────────────────────────────────┘
```

### 配置场景速查

| 场景 | 配置重点 | 耗时 |
| ------ | --------- | ------ |
| 🚀 **本地开发** | `Endpoint` | 10秒 |
| 🧪 **测试环境** | `Endpoint` + `TLS.InsecureSkipVerify` | 30秒 |
| 🏢 **生产环境** | `Endpoint` + `TLS` + `Retry` + `CircuitBreaker` | 5分钟 |
| ⚡ **高性能** | + `JWKS` + `Keepalive` | 10分钟 |
| 🔍 **可观测** | + `Observability` + `Metrics` | 15分钟 |

### 配置模板

```go
// 🚀 开发环境 (最简)
&Config{Endpoint: "localhost:8081"}

// 🧪 测试环境 (快速)
&Config{
    Endpoint: "iam-test.example.com:8081",
    TLS: &TLSConfig{Enabled: true, InsecureSkipVerify: true},
}

// 🏢 生产环境 (完整)
&Config{
    Endpoint: "iam.example.com:8081",
    TLS: &TLSConfig{Enabled: true, CACert: "/etc/certs/ca.crt"},
    Retry: &RetryConfig{Enabled: true, MaxAttempts: 3},
    CircuitBreaker: &CircuitBreakerConfig{FailureThreshold: 5},
}
```

### 配置优先级

```text
代码配置 (最高)
    ↓
环境变量 (IAM_ENDPOINT, IAM_TIMEOUT...)
    ↓
配置文件 (config.yaml)
    ↓
默认值 (最低)
```

---

## 📋 配置结构

```go
type Config struct {
    // 基础配置
    Endpoint        string                // gRPC 服务地址 (必填)
    Timeout         time.Duration         // 请求超时时间
    DialTimeout     time.Duration         // 连接超时时间
    
    // TLS 配置
    TLS             *TLSConfig
    
    // 连接保活
    Keepalive       *KeepaliveConfig
    
    // 重试配置
    Retry           *RetryConfig
    
    // JWKS 配置（用于本地 JWT 验证）
    JWKS            *JWKSConfig
    
    // 负载均衡
    LoadBalancer    string                // "round_robin" 或 "pick_first"
    
    // 熔断器配置
    CircuitBreaker  *CircuitBreakerConfig
    
    // 可观测性配置
    Observability   *ObservabilityConfig
    
    // 默认元数据
    Metadata        map[string]string
}
```

## 基础配置

### Endpoint（必填）

gRPC 服务地址，格式：`host:port`

```go
&Config{
    Endpoint: "iam.example.com:8081",
}
```

### Timeout

全局请求超时时间，默认 30 秒。可被方法级超时覆盖。

```go
&Config{
    Timeout: 30 * time.Second,
}
```

### DialTimeout

连接超时时间，默认 10 秒。

```go
&Config{
    DialTimeout: 10 * time.Second,
}
```

## TLS 配置

```go
type TLSConfig struct {
    Enabled            bool     // 是否启用 TLS
    CACert             string   // CA 证书文件路径
    CACertPEM          []byte   // CA 证书 PEM 内容（优先级高于文件）
    ClientCert         string   // 客户端证书文件路径（mTLS）
    ClientCertPEM      []byte   // 客户端证书 PEM 内容
    ClientKey          string   // 客户端私钥文件路径（mTLS）
    ClientKeyPEM       []byte   // 客户端私钥 PEM 内容
    ServerName         string   // 服务端名称（SNI）
    InsecureSkipVerify bool     // 跳过证书验证（仅测试）
    MinVersion         uint16   // 最低 TLS 版本（默认 TLS 1.2）
}
```

### 示例：单向 TLS

```go
&Config{
    TLS: &TLSConfig{
        Enabled:    true,
        CACert:     "/etc/iam/certs/ca.crt",
        ServerName: "iam.example.com",
        MinVersion: tls.VersionTLS12,
    },
}
```

### 示例：双向 mTLS

```go
&Config{
    TLS: &TLSConfig{
        Enabled:    true,
        CACert:     "/etc/iam/certs/ca.crt",
        ClientCert: "/etc/iam/certs/client.crt",
        ClientKey:  "/etc/iam/certs/client.key",
        ServerName: "iam.example.com",
        MinVersion: tls.VersionTLS13,
    },
}
```

### 示例：使用 PEM 内容（不使用文件）

```go
caCertPEM := []byte(`-----BEGIN CERTIFICATE-----
MIIDXTCCAkWgAwIBAgIJAL...
-----END CERTIFICATE-----`)

&Config{
    TLS: &TLSConfig{
        Enabled:   true,
        CACertPEM: caCertPEM,
    },
}
```

## Keepalive 配置

```go
type KeepaliveConfig struct {
    Time                time.Duration // 发送 keepalive ping 的间隔
    Timeout             time.Duration // 等待 keepalive ping 响应的超时时间
    PermitWithoutStream bool          // 是否在没有活跃 stream 时发送 keepalive
}
```

### 默认配置

```go
&Config{
    Keepalive: &KeepaliveConfig{
        Time:                30 * time.Second,
        Timeout:             10 * time.Second,
        PermitWithoutStream: true,
    },
}
```

## 重试配置

### 全局重试配置

```go
type RetryConfig struct {
    Enabled           bool          // 是否启用重试
    MaxAttempts       int           // 最大重试次数
    InitialBackoff    time.Duration // 初始退避时间
    MaxBackoff        time.Duration // 最大退避时间
    BackoffMultiplier float64       // 退避时间乘数
    RetryableCodes    []string      // 可重试的 gRPC 状态码
}
```

### 示例：标准重试配置

```go
&Config{
    Retry: &RetryConfig{
        Enabled:           true,
        MaxAttempts:       3,
        InitialBackoff:    100 * time.Millisecond,
        MaxBackoff:        10 * time.Second,
        BackoffMultiplier: 2.0,
        RetryableCodes:    []string{"UNAVAILABLE", "RESOURCE_EXHAUSTED", "ABORTED"},
    },
}
```

### 方法级重试配置（高级）

详见 [方法级重试配置](./07-advanced-retry.md)

## JWKS 配置

用于本地 JWT 验证的 JWKS 配置。

```go
type JWKSConfig struct {
    URL              string            // JWKS 端点 URL (HTTP/HTTPS)
    GRPCEndpoint     string            // gRPC 降级端点
    RefreshInterval  time.Duration     // 刷新间隔
    RequestTimeout   time.Duration     // HTTP 请求超时
    CacheTTL         time.Duration     // 缓存 TTL
    HTTPClient       *http.Client      // 自定义 HTTP 客户端
    CustomHeaders    map[string]string // 自定义请求头
    FallbackOnError  bool              // 失败时使用缓存
}
```

### 示例：标准 JWKS 配置

```go
&Config{
    JWKS: &JWKSConfig{
        URL:             "https://iam.example.com/.well-known/jwks.json",
        RefreshInterval: 5 * time.Minute,
        RequestTimeout:  10 * time.Second,
        CacheTTL:        1 * time.Hour,
        FallbackOnError: true,
    },
}
```

### 示例：JWKS + gRPC 降级

```go
&Config{
    JWKS: &JWKSConfig{
        URL:             "https://iam.example.com/.well-known/jwks.json",
        GRPCEndpoint:    "iam.example.com:8081", // HTTP 失败时使用 gRPC
        RefreshInterval: 5 * time.Minute,
    },
}
```

## 熔断器配置

```go
type CircuitBreakerConfig struct {
    FailureThreshold int           // 触发熔断的连续失败次数
    OpenDuration     time.Duration // 熔断器打开持续时间
    HalfOpenRequests int           // 半开状态允许的请求数
    SuccessThreshold int           // 半开→关闭所需的连续成功次数
}
```

### 示例：标准熔断器配置

```go
&Config{
    CircuitBreaker: &CircuitBreakerConfig{
        FailureThreshold: 5,
        OpenDuration:     30 * time.Second,
        HalfOpenRequests: 3,
        SuccessThreshold: 2,
    },
}
```

## 可观测性配置

```go
type ObservabilityConfig struct {
    EnableMetrics        bool   // 启用指标收集
    EnableTracing        bool   // 启用链路追踪
    EnableCircuitBreaker bool   // 启用熔断器
    EnableRequestID      bool   // 启用请求 ID 注入
    MetricsNamespace     string // Prometheus 指标命名空间
    MetricsSubsystem     string // Prometheus 指标子系统
    ServiceName          string // 服务名称（用于 tracing）
}
```

### 示例：完整可观测性配置

```go
&Config{
    Observability: &ObservabilityConfig{
        EnableMetrics:        true,
        EnableTracing:        true,
        EnableCircuitBreaker: true,
        EnableRequestID:      true,
        MetricsNamespace:     "myapp",
        MetricsSubsystem:     "iam_client",
        ServiceName:          "my-service",
    },
}
```

## 负载均衡

支持两种负载均衡策略：

- `round_robin`：轮询（默认）
- `pick_first`：选择第一个可用连接

```go
&Config{
    LoadBalancer: "round_robin",
}
```

## 环境变量映射

| 环境变量 | 配置字段 | 默认值 |
| --------- | --------- | -------- |
| `IAM_ENDPOINT` | `Endpoint` | - |
| `IAM_TIMEOUT` | `Timeout` | `30s` |
| `IAM_DIAL_TIMEOUT` | `DialTimeout` | `10s` |
| `IAM_TLS_ENABLED` | `TLS.Enabled` | `true` |
| `IAM_TLS_CA_CERT` | `TLS.CACert` | - |
| `IAM_TLS_CLIENT_CERT` | `TLS.ClientCert` | - |
| `IAM_TLS_CLIENT_KEY` | `TLS.ClientKey` | - |
| `IAM_TLS_SERVER_NAME` | `TLS.ServerName` | - |
| `IAM_TLS_INSECURE_SKIP_VERIFY` | `TLS.InsecureSkipVerify` | `false` |
| `IAM_RETRY_ENABLED` | `Retry.Enabled` | `true` |
| `IAM_RETRY_MAX_ATTEMPTS` | `Retry.MaxAttempts` | `3` |
| `IAM_JWKS_URL` | `JWKS.URL` | - |
| `IAM_JWKS_REFRESH_INTERVAL` | `JWKS.RefreshInterval` | `5m` |
| `IAM_LOAD_BALANCER` | `LoadBalancer` | `round_robin` |

### 使用环境变量

```bash
export IAM_ENDPOINT="iam.example.com:8081"
export IAM_TLS_ENABLED="true"
export IAM_TLS_CA_CERT="/etc/iam/certs/ca.crt"
export IAM_TIMEOUT="30s"
export IAM_RETRY_MAX_ATTEMPTS="5"
```

```go
cfg, err := sdk.ConfigFromEnv()
if err != nil {
    log.Fatal(err)
}

client, err := sdk.NewClient(ctx, cfg)
```

## YAML 配置文件

```yaml
iam:
  endpoint: "iam.example.com:8081"
  timeout: 30s
  dial_timeout: 10s
  load_balancer: "round_robin"
  
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
    max_backoff: 10s
    backoff_multiplier: 2.0
    retryable_codes: ["UNAVAILABLE", "RESOURCE_EXHAUSTED"]
  
  jwks:
    url: "https://iam.example.com/.well-known/jwks.json"
    refresh_interval: 5m
    request_timeout: 10s
  
  circuit_breaker:
    failure_threshold: 5
    open_duration: 30s
    half_open_requests: 3
  
  observability:
    enable_metrics: true
    enable_tracing: true
    enable_circuit_breaker: true
    enable_request_id: true
    metrics_namespace: "myapp"
    service_name: "my-service"
```

### 使用 Viper 加载

```go
import (
    "github.com/spf13/viper"
    sdk "github.com/FangcunMount/iam-contracts/pkg/sdk"
    "github.com/FangcunMount/iam-contracts/pkg/sdk/config"
)

func main() {
    viper.SetConfigName("config")
    viper.SetConfigType("yaml")
    viper.AddConfigPath(".")
    
    if err := viper.ReadInConfig(); err != nil {
        log.Fatal(err)
    }
    
    cfg, err := config.FromViper(viper.GetViper())
    if err != nil {
        log.Fatal(err)
    }
    
    client, err := sdk.NewClient(context.Background(), cfg)
    // ...
}
```

## 配置验证

SDK 会在创建客户端时自动验证配置：

```go
client, err := sdk.NewClient(ctx, cfg)
if err != nil {
    // 配置验证失败
    log.Fatal(err)
}
```

常见验证错误：

- `Endpoint` 为空
- TLS 证书文件不存在
- 超时时间为负数
- 重试次数小于 1

## 下一步

- [JWT 验证](./03-jwt-verification.md)
- [方法级重试配置](./07-advanced-retry.md)
- [可观测性配置](./05-observability.md)

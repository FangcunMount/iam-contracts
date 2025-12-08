// Package config 提供 IAM SDK 的配置定义
package config

import (
	"crypto/tls"
	"net/http"
	"time"
)

// Config 是 IAM SDK 的主配置结构
type Config struct {
	// Endpoint gRPC 服务地址，格式: host:port
	Endpoint string

	// TLS 配置，nil 表示使用不安全连接
	TLS *TLSConfig

	// Timeout 请求超时时间
	Timeout time.Duration

	// DialTimeout 连接超时时间
	DialTimeout time.Duration

	// Keepalive 连接保活配置
	Keepalive *KeepaliveConfig

	// Retry 重试配置
	Retry *RetryConfig

	// JWKS 配置（用于本地 Token 验证）
	JWKS *JWKSConfig

	// Metadata 默认请求元数据
	Metadata map[string]string

	// LoadBalancer 负载均衡策略: round_robin, pick_first
	LoadBalancer string

	// CircuitBreaker 熔断器配置
	CircuitBreaker *CircuitBreakerConfig

	// Observability 可观测性配置
	Observability *ObservabilityConfig
}

// TLSConfig TLS/mTLS 配置
type TLSConfig struct {
	// Enabled 是否启用 TLS
	Enabled bool

	// CACert CA 证书文件路径
	CACert string

	// CACertPEM CA 证书 PEM 内容（优先级高于文件路径）
	CACertPEM []byte

	// ClientCert 客户端证书文件路径（mTLS）
	ClientCert string

	// ClientCertPEM 客户端证书 PEM 内容
	ClientCertPEM []byte

	// ClientKey 客户端私钥文件路径（mTLS）
	ClientKey string

	// ClientKeyPEM 客户端私钥 PEM 内容
	ClientKeyPEM []byte

	// ServerName 服务端名称（SNI）
	ServerName string

	// InsecureSkipVerify 跳过证书验证（仅用于测试）
	InsecureSkipVerify bool

	// MinVersion 最低 TLS 版本，默认 TLS 1.2
	MinVersion uint16
}

// KeepaliveConfig gRPC 连接保活配置
type KeepaliveConfig struct {
	// Time 发送 keepalive ping 的间隔
	Time time.Duration

	// Timeout 等待 keepalive ping 响应的超时时间
	Timeout time.Duration

	// PermitWithoutStream 是否在没有活跃 stream 时发送 keepalive
	PermitWithoutStream bool
}

// RetryConfig 重试配置
type RetryConfig struct {
	// Enabled 是否启用重试
	Enabled bool

	// MaxAttempts 最大重试次数
	MaxAttempts int

	// InitialBackoff 初始退避时间
	InitialBackoff time.Duration

	// MaxBackoff 最大退避时间
	MaxBackoff time.Duration

	// BackoffMultiplier 退避时间乘数
	BackoffMultiplier float64

	// RetryableCodes 可重试的状态码
	RetryableCodes []string
}

// JWKSConfig JWKS 配置
type JWKSConfig struct {
	// URL JWKS 端点 URL
	URL string

	// GRPCEndpoint gRPC 降级端点（当 HTTP 失败时使用）
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

// TokenVerifyConfig Token 验证配置
type TokenVerifyConfig struct {
	// AllowedAudience 允许的 audience 列表
	AllowedAudience []string

	// AllowedIssuer 允许的 issuer
	AllowedIssuer string

	// ClockSkew 时钟偏差容忍度
	ClockSkew time.Duration

	// RequireExpirationTime 是否要求 exp 声明
	RequireExpirationTime bool

	// ForceRemoteVerification 强制使用远程验证
	ForceRemoteVerification bool
}

// CircuitBreakerConfig 熔断器配置
type CircuitBreakerConfig struct {
	// FailureThreshold 触发熔断的连续失败次数
	FailureThreshold int

	// OpenDuration 熔断器打开持续时间
	OpenDuration time.Duration

	// HalfOpenRequests 半开状态允许的请求数
	HalfOpenRequests int

	// SuccessThreshold 半开状态恢复到关闭状态所需的连续成功次数
	SuccessThreshold int
}

// ObservabilityConfig 可观测性配置
type ObservabilityConfig struct {
	// EnableMetrics 启用指标收集
	EnableMetrics bool

	// EnableTracing 启用链路追踪
	EnableTracing bool

	// EnableCircuitBreaker 启用熔断器
	EnableCircuitBreaker bool

	// EnableRequestID 启用请求 ID 注入
	EnableRequestID bool

	// MetricsNamespace Prometheus 指标命名空间
	MetricsNamespace string

	// MetricsSubsystem Prometheus 指标子系统
	MetricsSubsystem string

	// ServiceName 服务名称（用于 tracing）
	ServiceName string
}

// DefaultObservabilityConfig 默认可观测性配置
func DefaultObservabilityConfig() *ObservabilityConfig {
	return &ObservabilityConfig{
		EnableMetrics:        true,
		EnableTracing:        false, // 默认关闭，需要用户配置 tracer
		EnableCircuitBreaker: true,
		EnableRequestID:      true,
		MetricsNamespace:     "iam",
		MetricsSubsystem:     "sdk",
		ServiceName:          "iam-sdk",
	}
}

// ServiceAuthConfig 服务间认证配置
type ServiceAuthConfig struct {
	// ServiceID 当前服务标识
	ServiceID string

	// TargetAudience 目标服务 audience
	TargetAudience []string

	// TokenTTL Token 有效期
	TokenTTL time.Duration

	// RefreshBefore 提前刷新时间
	RefreshBefore time.Duration
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Timeout:      30 * time.Second,
		DialTimeout:  10 * time.Second,
		LoadBalancer: "round_robin",
		TLS: &TLSConfig{
			Enabled:    true,
			MinVersion: tls.VersionTLS12,
		},
		Retry: &RetryConfig{
			Enabled:           true,
			MaxAttempts:       3,
			InitialBackoff:    100 * time.Millisecond,
			MaxBackoff:        10 * time.Second,
			BackoffMultiplier: 2.0,
			RetryableCodes:    []string{"UNAVAILABLE", "RESOURCE_EXHAUSTED", "ABORTED"},
		},
		Keepalive: &KeepaliveConfig{
			Time:                30 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		},
	}
}

// Validate 验证配置有效性
func (c *Config) Validate() error {
	if c.Endpoint == "" {
		return ErrEndpointRequired
	}
	return nil
}

// WithDefaults 填充默认值
func (c *Config) WithDefaults() *Config {
	defaults := DefaultConfig()

	if c.Timeout == 0 {
		c.Timeout = defaults.Timeout
	}
	if c.DialTimeout == 0 {
		c.DialTimeout = defaults.DialTimeout
	}
	if c.LoadBalancer == "" {
		c.LoadBalancer = defaults.LoadBalancer
	}
	if c.Retry == nil {
		c.Retry = defaults.Retry
	}
	if c.Keepalive == nil {
		c.Keepalive = defaults.Keepalive
	}

	return c
}

package config

import (
	"net/http"
	"time"
)

// JWKSConfig JWKS（JSON Web Key Set）配置。
type JWKSConfig struct {
	URL             string
	GRPCEndpoint    string
	RefreshInterval time.Duration
	RequestTimeout  time.Duration
	CacheTTL        time.Duration
	HTTPClient      *http.Client
	CustomHeaders   map[string]string
	FallbackOnError bool
}

// TokenVerifyConfig Token 验证配置。
type TokenVerifyConfig struct {
	AllowedAudience         []string
	AllowedIssuer           string
	ClockSkew               time.Duration
	RequireExpirationTime   bool
	ForceRemoteVerification bool
	RequiredClaims          []string
	Algorithms              []string
}

// CircuitBreakerConfig 熔断器配置。
type CircuitBreakerConfig struct {
	FailureThreshold int
	OpenDuration     time.Duration
	HalfOpenRequests int
	SuccessThreshold int
}

// ObservabilityConfig 控制 SDK 默认 metrics / tracing / circuit breaker 链路。
type ObservabilityConfig struct {
	EnableMetrics        bool
	EnableTracing        bool
	EnableCircuitBreaker bool
	EnableRequestID      bool
	MetricsNamespace     string
	MetricsSubsystem     string
	ServiceName          string
}

// ServiceAuthConfig 服务间认证配置。
type ServiceAuthConfig struct {
	ServiceID      string
	TargetAudience []string
	TokenTTL       time.Duration
	RefreshBefore  time.Duration
}

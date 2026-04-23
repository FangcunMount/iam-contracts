// Package config 提供 IAM SDK 的公开配置结构与加载入口。
package config

import "time"

// Config 是 IAM SDK 的主配置结构。
type Config struct {
	Endpoint string
	TLS      *TLSConfig

	Timeout     time.Duration
	DialTimeout time.Duration
	Keepalive   *KeepaliveConfig
	Retry       *RetryConfig

	JWKS           *JWKSConfig
	Metadata       map[string]string
	LoadBalancer   string
	CircuitBreaker *CircuitBreakerConfig
	Observability  *ObservabilityConfig
}

package config

import (
	"crypto/tls"
	"fmt"
	"time"
)

// EnvLoader 环境变量配置加载器。
type EnvLoader struct {
	prefix string
}

// NewEnvLoader 创建环境变量配置加载器。
func NewEnvLoader(prefix string) *EnvLoader {
	return &EnvLoader{prefix: prefix}
}

// Load 从环境变量加载配置。
func (l *EnvLoader) Load() (*Config, error) {
	cfg := &Config{}

	cfg.Endpoint = l.getString("ENDPOINT", "")
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("%s_ENDPOINT is required", l.prefix)
	}

	cfg.Timeout = l.getDuration("TIMEOUT", 30*time.Second)
	cfg.DialTimeout = l.getDuration("DIAL_TIMEOUT", 10*time.Second)
	cfg.LoadBalancer = l.getString("LOAD_BALANCER", "round_robin")

	cfg.TLS = l.loadTLS()
	cfg.Retry = l.loadRetry()
	cfg.Keepalive = l.loadKeepalive()
	cfg.JWKS = l.loadJWKS()
	cfg.CircuitBreaker = l.loadCircuitBreaker()
	cfg.Observability = l.loadObservability()

	return cfg, nil
}

func (l *EnvLoader) loadTLS() *TLSConfig {
	if !l.getBool("TLS_ENABLED", true) {
		return nil
	}

	return &TLSConfig{
		Enabled:            true,
		CACert:             l.getString("TLS_CA_CERT", ""),
		ClientCert:         l.getString("TLS_CLIENT_CERT", ""),
		ClientKey:          l.getString("TLS_CLIENT_KEY", ""),
		ServerName:         l.getString("TLS_SERVER_NAME", ""),
		InsecureSkipVerify: l.getBool("TLS_SKIP_VERIFY", false),
		MinVersion:         tls.VersionTLS12,
	}
}

func (l *EnvLoader) loadRetry() *RetryConfig {
	if !l.getBool("RETRY_ENABLED", true) {
		return nil
	}

	return &RetryConfig{
		Enabled:           true,
		MaxAttempts:       l.getInt("RETRY_MAX_ATTEMPTS", 3),
		InitialBackoff:    l.getDuration("RETRY_INITIAL_BACKOFF", 100*time.Millisecond),
		MaxBackoff:        l.getDuration("RETRY_MAX_BACKOFF", 10*time.Second),
		BackoffMultiplier: l.getFloat64("RETRY_BACKOFF_MULTIPLIER", 2.0),
	}
}

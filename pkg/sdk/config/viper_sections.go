package config

import (
	"crypto/tls"
	"time"
)

func (l *ViperLoader) loadTLS(prefix string) *TLSConfig {
	if !l.getBool(prefix+".tls.enabled", true) {
		return nil
	}

	return &TLSConfig{
		Enabled:            true,
		CACert:             l.getString(prefix + ".tls.ca_cert"),
		ClientCert:         l.getString(prefix + ".tls.client_cert"),
		ClientKey:          l.getString(prefix + ".tls.client_key"),
		ServerName:         l.getString(prefix + ".tls.server_name"),
		InsecureSkipVerify: l.getBool(prefix+".tls.insecure_skip_verify", false),
		MinVersion:         tls.VersionTLS12,
	}
}

func (l *ViperLoader) loadRetry(prefix string) *RetryConfig {
	if !l.getBool(prefix+".retry.enabled", true) {
		return nil
	}

	return &RetryConfig{
		Enabled:           true,
		MaxAttempts:       l.getInt(prefix+".retry.max_attempts", 3),
		InitialBackoff:    l.getDuration(prefix+".retry.initial_backoff", 100*time.Millisecond),
		MaxBackoff:        l.getDuration(prefix+".retry.max_backoff", 10*time.Second),
		BackoffMultiplier: l.getFloat64(prefix+".retry.backoff_multiplier", 2.0),
	}
}

func (l *ViperLoader) loadKeepalive(prefix string) *KeepaliveConfig {
	if l.getValue(prefix+".keepalive") == nil {
		return nil
	}

	return &KeepaliveConfig{
		Time:                l.getDuration(prefix+".keepalive.time", defaultKeepaliveTime),
		Timeout:             l.getDuration(prefix+".keepalive.timeout", defaultKeepaliveTimeout),
		PermitWithoutStream: l.getBool(prefix+".keepalive.permit_without_stream", defaultKeepalivePermitWithoutStream),
	}
}

func (l *ViperLoader) loadJWKS(prefix string) *JWKSConfig {
	jwksURL := l.getString(prefix + ".jwks.url")
	grpcEndpoint := l.getString(prefix + ".jwks.grpc_endpoint")
	if jwksURL == "" && grpcEndpoint == "" {
		return nil
	}

	return &JWKSConfig{
		URL:             jwksURL,
		GRPCEndpoint:    grpcEndpoint,
		RefreshInterval: l.getDuration(prefix+".jwks.refresh_interval", defaultJWKSRefreshInterval),
		RequestTimeout:  l.getDuration(prefix+".jwks.request_timeout", defaultJWKSRequestTimeout),
		CacheTTL:        l.getDuration(prefix+".jwks.cache_ttl", 0),
		FallbackOnError: l.getBool(prefix+".jwks.fallback_on_error", false),
	}
}

func (l *ViperLoader) loadCircuitBreaker(prefix string) *CircuitBreakerConfig {
	if l.getValue(prefix+".circuit_breaker") == nil {
		return nil
	}

	return &CircuitBreakerConfig{
		FailureThreshold: l.getInt(prefix+".circuit_breaker.failure_threshold", 5),
		OpenDuration:     l.getDuration(prefix+".circuit_breaker.open_duration", 30*time.Second),
		HalfOpenRequests: l.getInt(prefix+".circuit_breaker.half_open_requests", 3),
		SuccessThreshold: l.getInt(prefix+".circuit_breaker.success_threshold", 2),
	}
}

func (l *ViperLoader) loadObservability(prefix string) *ObservabilityConfig {
	if l.getValue(prefix+".observability") == nil {
		return nil
	}

	return &ObservabilityConfig{
		EnableMetrics:        l.getBool(prefix+".observability.enable_metrics", false),
		EnableTracing:        l.getBool(prefix+".observability.enable_tracing", false),
		EnableCircuitBreaker: l.getBool(prefix+".observability.enable_circuit_breaker", false),
		EnableRequestID:      l.getBool(prefix+".observability.enable_request_id", false),
		MetricsNamespace:     l.getStringDefault(prefix+".observability.metrics_namespace", "iam"),
		MetricsSubsystem:     l.getStringDefault(prefix+".observability.metrics_subsystem", "sdk"),
		ServiceName:          l.getStringDefault(prefix+".observability.service_name", "iam-sdk"),
	}
}

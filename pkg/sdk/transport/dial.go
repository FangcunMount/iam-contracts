// Package transport 提供 gRPC 传输层功能
package transport

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/FangcunMount/iam-contracts/pkg/sdk/config"
	"github.com/FangcunMount/iam-contracts/pkg/sdk/observability"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

// DialResult Dial 的返回结果，包含连接和可观测性组件
type DialResult struct {
	Conn           *grpc.ClientConn
	Metrics        *observability.PrometheusMetrics
	CircuitBreaker *observability.CircuitBreaker
}

// Dial 创建 gRPC 连接
func Dial(ctx context.Context, cfg *config.Config, opts *config.ClientOptions) (*grpc.ClientConn, error) {
	result, err := DialWithObservability(ctx, cfg, opts)
	if err != nil {
		return nil, err
	}
	return result.Conn, nil
}

// DialWithObservability 创建 gRPC 连接，并返回可观测性组件
func DialWithObservability(ctx context.Context, cfg *config.Config, opts *config.ClientOptions) (*DialResult, error) {
	result := &DialResult{}

	// 构建可观测性组件
	obsCfg := cfg.Observability
	if obsCfg == nil {
		obsCfg = config.DefaultObservabilityConfig()
	}

	// 创建默认拦截器链
	var defaultInterceptors []grpc.UnaryClientInterceptor

	// 1. RequestID 拦截器（最外层）
	if obsCfg.EnableRequestID {
		defaultInterceptors = append(defaultInterceptors, RequestIDInterceptor())
	}

	// 2. Metrics 拦截器
	if obsCfg.EnableMetrics {
		metrics := observability.NewPrometheusMetrics(
			observability.WithPrometheusNamespace(obsCfg.MetricsNamespace),
			observability.WithPrometheusSubsystem(obsCfg.MetricsSubsystem),
		)
		// 注意：注册应该由用户在应用启动时完成
		result.Metrics = metrics
		defaultInterceptors = append(defaultInterceptors, observability.MetricsUnaryInterceptor(metrics))
	}

	// 3. CircuitBreaker 拦截器
	if obsCfg.EnableCircuitBreaker && cfg.CircuitBreaker != nil {
		cb := observability.NewCircuitBreaker(&observability.CircuitBreakerConfig{
			FailureThreshold: cfg.CircuitBreaker.FailureThreshold,
			OpenDuration:     cfg.CircuitBreaker.OpenDuration,
			HalfOpenRequests: cfg.CircuitBreaker.HalfOpenRequests,
		})
		result.CircuitBreaker = cb
		defaultInterceptors = append(defaultInterceptors, observability.CircuitBreakerInterceptor(cb))
	}

	// 4. Tracing 拦截器（如果用户提供了 TracingHook）
	if obsCfg.EnableTracing && opts != nil && opts.TracingHook != nil {
		if hook, ok := opts.TracingHook.(observability.TracingHook); ok {
			defaultInterceptors = append(defaultInterceptors, observability.TracingUnaryInterceptor(hook))
		}
	}

	// 如果用户禁用了默认拦截器，清空
	if opts != nil && opts.DisableDefaultInterceptors {
		defaultInterceptors = nil
	}

	// 合并用户自定义拦截器
	var allInterceptors []grpc.UnaryClientInterceptor
	allInterceptors = append(allInterceptors, defaultInterceptors...)
	if opts != nil && len(opts.UnaryInterceptors) > 0 {
		allInterceptors = append(allInterceptors, opts.UnaryInterceptors...)
	}

	// 构建 DialOptions
	dialOpts, err := BuildDialOptions(cfg, opts)
	if err != nil {
		return nil, fmt.Errorf("build dial options: %w", err)
	}

	// 添加合并后的拦截器链
	if len(allInterceptors) > 0 {
		dialOpts = append(dialOpts, grpc.WithChainUnaryInterceptor(allInterceptors...))
	}

	dialCtx := ctx
	if cfg.DialTimeout > 0 {
		var cancel context.CancelFunc
		dialCtx, cancel = context.WithTimeout(ctx, cfg.DialTimeout)
		defer cancel()
	}

	conn, err := grpc.DialContext(dialCtx, cfg.Endpoint, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", cfg.Endpoint, err)
	}

	result.Conn = conn
	return result, nil
}

// BuildDialOptions 构建 gRPC DialOption（不包含拦截器，拦截器由 Dial 统一处理）
func BuildDialOptions(cfg *config.Config, opts *config.ClientOptions) ([]grpc.DialOption, error) {
	var dialOpts []grpc.DialOption

	// TLS 配置
	tlsCreds, err := BuildTLSCredentials(cfg.TLS)
	if err != nil {
		return nil, err
	}
	dialOpts = append(dialOpts, tlsCreds)

	// Keepalive 配置
	if cfg.Keepalive != nil {
		dialOpts = append(dialOpts, grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                cfg.Keepalive.Time,
			Timeout:             cfg.Keepalive.Timeout,
			PermitWithoutStream: cfg.Keepalive.PermitWithoutStream,
		}))
	}

	// 重试和负载均衡配置
	serviceConfig := BuildServiceConfig(cfg)
	if serviceConfig != "" {
		dialOpts = append(dialOpts, grpc.WithDefaultServiceConfig(serviceConfig))
	}

	// Stream 拦截器
	if opts != nil {
		if len(opts.StreamInterceptors) > 0 {
			dialOpts = append(dialOpts, grpc.WithChainStreamInterceptor(opts.StreamInterceptors...))
		}
		if len(opts.DialOptions) > 0 {
			dialOpts = append(dialOpts, opts.DialOptions...)
		}
	}

	return dialOpts, nil
}

// BuildTLSCredentials 构建 TLS 凭证
func BuildTLSCredentials(tlsCfg *config.TLSConfig) (grpc.DialOption, error) {
	if tlsCfg == nil || !tlsCfg.Enabled {
		return grpc.WithTransportCredentials(insecure.NewCredentials()), nil
	}

	tlsConfig, err := BuildTLSConfig(tlsCfg)
	if err != nil {
		return nil, err
	}

	return grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)), nil
}

// BuildTLSConfig 构建 tls.Config
func BuildTLSConfig(cfg *config.TLSConfig) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: cfg.InsecureSkipVerify,
		ServerName:         cfg.ServerName,
		MinVersion:         cfg.MinVersion,
	}

	if tlsConfig.MinVersion == 0 {
		tlsConfig.MinVersion = tls.VersionTLS12
	}

	// 加载 CA 证书
	if len(cfg.CACertPEM) > 0 || cfg.CACert != "" {
		certPool := x509.NewCertPool()

		var caCert []byte
		if len(cfg.CACertPEM) > 0 {
			caCert = cfg.CACertPEM
		} else {
			var err error
			caCert, err = os.ReadFile(cfg.CACert)
			if err != nil {
				return nil, fmt.Errorf("read CA cert: %w", err)
			}
		}

		if !certPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to append CA cert")
		}
		tlsConfig.RootCAs = certPool
	}

	// 加载客户端证书（mTLS）
	if (len(cfg.ClientCertPEM) > 0 && len(cfg.ClientKeyPEM) > 0) ||
		(cfg.ClientCert != "" && cfg.ClientKey != "") {

		var cert tls.Certificate
		var err error

		if len(cfg.ClientCertPEM) > 0 {
			cert, err = tls.X509KeyPair(cfg.ClientCertPEM, cfg.ClientKeyPEM)
		} else {
			cert, err = tls.LoadX509KeyPair(cfg.ClientCert, cfg.ClientKey)
		}

		if err != nil {
			return nil, fmt.Errorf("load client cert: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}

// BuildServiceConfig 构建 gRPC ServiceConfig
func BuildServiceConfig(cfg *config.Config) string {
	var parts []string

	// 负载均衡
	lb := cfg.LoadBalancer
	if lb == "" {
		lb = "round_robin"
	}
	parts = append(parts, fmt.Sprintf(`"loadBalancingPolicy": "%s"`, lb))

	// 重试策略
	if cfg.Retry != nil && cfg.Retry.Enabled {
		retryConfig := buildRetryConfig(cfg.Retry)
		parts = append(parts, retryConfig)
	}

	return "{" + strings.Join(parts, ",") + "}"
}

func buildRetryConfig(retry *config.RetryConfig) string {
	maxAttempts := retry.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 3
	}

	initialBackoff := retry.InitialBackoff
	if initialBackoff <= 0 {
		initialBackoff = 100 * time.Millisecond
	}

	maxBackoff := retry.MaxBackoff
	if maxBackoff <= 0 {
		maxBackoff = 10 * time.Second
	}

	multiplier := retry.BackoffMultiplier
	if multiplier <= 0 {
		multiplier = 2.0
	}

	codes := retry.RetryableCodes
	if len(codes) == 0 {
		codes = []string{"UNAVAILABLE", "RESOURCE_EXHAUSTED", "ABORTED"}
	}

	return fmt.Sprintf(`"methodConfig": [{
		"name": [{"service": ""}],
		"retryPolicy": {
			"maxAttempts": %d,
			"initialBackoff": "%s",
			"maxBackoff": "%s",
			"backoffMultiplier": %.1f,
			"retryableStatusCodes": [%s]
		}
	}]`, maxAttempts, initialBackoff, maxBackoff, multiplier, formatCodes(codes))
}

func formatCodes(codes []string) string {
	quoted := make([]string, len(codes))
	for i, c := range codes {
		quoted[i] = fmt.Sprintf(`"%s"`, c)
	}
	return strings.Join(quoted, ",")
}

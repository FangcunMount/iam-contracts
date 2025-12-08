package transport

import (
	"context"
	"math"
	"math/rand"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ========== 按方法配置 ==========

// MethodConfig 方法级别配置
type MethodConfig struct {
	// Timeout 方法超时时间
	Timeout time.Duration

	// Retry 重试配置
	Retry *MethodRetryConfig
}

// MethodRetryConfig 方法级别重试配置
type MethodRetryConfig struct {
	// MaxAttempts 最大重试次数
	MaxAttempts int

	// InitialBackoff 初始退避时间
	InitialBackoff time.Duration

	// MaxBackoff 最大退避时间
	MaxBackoff time.Duration

	// BackoffMultiplier 退避乘数
	BackoffMultiplier float64

	// RetryableCodes 可重试的状态码
	RetryableCodes []codes.Code
}

// DefaultMethodRetryConfig 默认重试配置
func DefaultMethodRetryConfig() *MethodRetryConfig {
	return &MethodRetryConfig{
		MaxAttempts:       3,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        10 * time.Second,
		BackoffMultiplier: 2.0,
		RetryableCodes: []codes.Code{
			codes.Unavailable,
			codes.ResourceExhausted,
			codes.Aborted,
		},
	}
}

// MethodConfigs 方法配置表
type MethodConfigs struct {
	// 默认配置
	Default *MethodConfig

	// 按服务配置
	Services map[string]*MethodConfig

	// 按完整方法名配置（优先级最高）
	Methods map[string]*MethodConfig
}

// NewMethodConfigs 创建方法配置表
func NewMethodConfigs() *MethodConfigs {
	return &MethodConfigs{
		Default: &MethodConfig{
			Timeout: 30 * time.Second,
			Retry:   DefaultMethodRetryConfig(),
		},
		Services: make(map[string]*MethodConfig),
		Methods:  make(map[string]*MethodConfig),
	}
}

// GetConfig 获取方法配置
func (mc *MethodConfigs) GetConfig(fullMethod string) *MethodConfig {
	// 优先级：Methods > Services > Default
	if cfg, ok := mc.Methods[fullMethod]; ok {
		return cfg
	}

	service := extractServiceName(fullMethod)
	if cfg, ok := mc.Services[service]; ok {
		return cfg
	}

	return mc.Default
}

// SetMethodConfig 设置方法配置
func (mc *MethodConfigs) SetMethodConfig(method string, cfg *MethodConfig) {
	mc.Methods[method] = cfg
}

// SetServiceConfig 设置服务配置
func (mc *MethodConfigs) SetServiceConfig(service string, cfg *MethodConfig) {
	mc.Services[service] = cfg
}

// ========== 超时拦截器 ==========

// TimeoutInterceptor 按方法超时拦截器
func TimeoutUnaryInterceptor(configs *MethodConfigs) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{},
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {

		cfg := configs.GetConfig(method)
		if cfg == nil || cfg.Timeout <= 0 {
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		// 检查是否已有超时
		if _, ok := ctx.Deadline(); !ok {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, cfg.Timeout)
			defer cancel()
		}

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// ========== 重试拦截器 ==========

// RetryUnaryInterceptor 按方法重试拦截器
func RetryUnaryInterceptor(configs *MethodConfigs) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{},
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {

		cfg := configs.GetConfig(method)
		if cfg == nil || cfg.Retry == nil || cfg.Retry.MaxAttempts <= 1 {
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		retryCfg := cfg.Retry
		var lastErr error

		for attempt := 0; attempt < retryCfg.MaxAttempts; attempt++ {
			// 执行调用
			err := invoker(ctx, method, req, reply, cc, opts...)
			if err == nil {
				return nil
			}

			lastErr = err

			// 检查是否可重试
			if !isRetryable(err, retryCfg.RetryableCodes) {
				return err
			}

			// 最后一次尝试不等待
			if attempt == retryCfg.MaxAttempts-1 {
				break
			}

			// 计算退避时间
			backoff := calculateBackoff(attempt, retryCfg)

			// 等待或超时
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		return lastErr
	}
}

func isRetryable(err error, retryableCodes []codes.Code) bool {
	st, ok := status.FromError(err)
	if !ok {
		return false
	}

	for _, code := range retryableCodes {
		if st.Code() == code {
			return true
		}
	}
	return false
}

func calculateBackoff(attempt int, cfg *MethodRetryConfig) time.Duration {
	backoff := float64(cfg.InitialBackoff) * math.Pow(cfg.BackoffMultiplier, float64(attempt))
	if backoff > float64(cfg.MaxBackoff) {
		backoff = float64(cfg.MaxBackoff)
	}

	// 添加抖动（±10%）
	jitter := backoff * 0.1 * (rand.Float64()*2 - 1)
	backoff += jitter

	return time.Duration(backoff)
}

func extractServiceName(fullMethod string) string {
	if len(fullMethod) > 0 && fullMethod[0] == '/' {
		fullMethod = fullMethod[1:]
	}
	for i := 0; i < len(fullMethod); i++ {
		if fullMethod[i] == '/' {
			return fullMethod[:i]
		}
	}
	return fullMethod
}

// ========== 预定义配置 ==========

// ReadOnlyMethodConfig 只读方法配置（可重试）
var ReadOnlyMethodConfig = &MethodConfig{
	Timeout: 10 * time.Second,
	Retry: &MethodRetryConfig{
		MaxAttempts:       3,
		InitialBackoff:    50 * time.Millisecond,
		MaxBackoff:        2 * time.Second,
		BackoffMultiplier: 2.0,
		RetryableCodes: []codes.Code{
			codes.Unavailable,
			codes.ResourceExhausted,
			codes.Aborted,
			codes.DeadlineExceeded,
		},
	},
}

// WriteMethodConfig 写方法配置（不重试）
var WriteMethodConfig = &MethodConfig{
	Timeout: 30 * time.Second,
	Retry:   nil, // 不重试
}

// LongRunningMethodConfig 长时间运行方法配置
var LongRunningMethodConfig = &MethodConfig{
	Timeout: 5 * time.Minute,
	Retry: &MethodRetryConfig{
		MaxAttempts:       2,
		InitialBackoff:    1 * time.Second,
		MaxBackoff:        30 * time.Second,
		BackoffMultiplier: 2.0,
		RetryableCodes: []codes.Code{
			codes.Unavailable,
		},
	},
}

// IdempotentMethodConfig 幂等方法配置（激进重试）
var IdempotentMethodConfig = &MethodConfig{
	Timeout: 30 * time.Second,
	Retry: &MethodRetryConfig{
		MaxAttempts:       4,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        5 * time.Second,
		BackoffMultiplier: 2.0,
		RetryableCodes:    IdempotentRetryableCodes(),
	},
}

// =============================================================================
// 可重试状态码集合
// =============================================================================

// IdempotentRetryableCodes 幂等操作可重试的状态码（更宽松）
func IdempotentRetryableCodes() []codes.Code {
	return []codes.Code{
		codes.Unavailable,
		codes.ResourceExhausted,
		codes.Aborted,
		codes.DeadlineExceeded,
		codes.Unknown,  // 幂等操作可以重试 Unknown
		codes.Internal, // 幂等操作可以重试 Internal
	}
}

// NonIdempotentRetryableCodes 非幂等操作可重试的状态码（更严格）
func NonIdempotentRetryableCodes() []codes.Code {
	return []codes.Code{
		codes.Unavailable, // 只重试明确的连接问题
	}
}

// DefaultRetryableCodes 默认可重试的状态码
func DefaultRetryableCodes() []codes.Code {
	return []codes.Code{
		codes.Unavailable,
		codes.ResourceExhausted,
		codes.Aborted,
		codes.DeadlineExceeded,
	}
}

// =============================================================================
// 自定义重试判断
// =============================================================================

// RetryPredicate 重试判断函数
type RetryPredicate func(err error, attempt int) bool

// RetryPredicateConfig 带自定义判断的重试配置
type RetryPredicateConfig struct {
	*MethodRetryConfig

	// Predicate 自定义重试判断（优先级高于 RetryableCodes）
	Predicate RetryPredicate

	// OnRetry 重试时回调
	OnRetry func(attempt int, err error, backoff time.Duration)
}

// WithRetryPredicate 包装重试配置添加自定义判断
func WithRetryPredicate(cfg *MethodRetryConfig, predicate RetryPredicate) *RetryPredicateConfig {
	return &RetryPredicateConfig{
		MethodRetryConfig: cfg,
		Predicate:         predicate,
	}
}

// RetryPredicateUnaryInterceptor 带自定义判断的重试拦截器
func RetryPredicateUnaryInterceptor(configs map[string]*RetryPredicateConfig, defaultCfg *RetryPredicateConfig) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{},
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {

		cfg := configs[method]
		if cfg == nil {
			cfg = defaultCfg
		}
		if cfg == nil || cfg.MaxAttempts <= 1 {
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		var lastErr error

		for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
			err := invoker(ctx, method, req, reply, cc, opts...)
			if err == nil {
				return nil
			}

			lastErr = err

			// 自定义判断优先
			shouldRetry := false
			if cfg.Predicate != nil {
				shouldRetry = cfg.Predicate(err, attempt)
			} else {
				shouldRetry = isRetryable(err, cfg.RetryableCodes)
			}

			if !shouldRetry {
				return err
			}

			// 最后一次尝试不等待
			if attempt == cfg.MaxAttempts-1 {
				break
			}

			// 计算退避时间
			backoff := calculateBackoff(attempt, cfg.MethodRetryConfig)

			// 回调
			if cfg.OnRetry != nil {
				cfg.OnRetry(attempt+1, err, backoff)
			}

			// 等待或超时
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		return lastErr
	}
}

// =============================================================================
// 常用重试判断函数
// =============================================================================

// RetryOnNetworkError 网络错误时重试
func RetryOnNetworkError() RetryPredicate {
	return func(err error, attempt int) bool {
		st, ok := status.FromError(err)
		if !ok {
			return false
		}
		switch st.Code() {
		case codes.Unavailable, codes.DeadlineExceeded, codes.Canceled:
			return true
		}
		return false
	}
}

// RetryOnServerError 服务端错误时重试
func RetryOnServerError() RetryPredicate {
	return func(err error, attempt int) bool {
		st, ok := status.FromError(err)
		if !ok {
			return false
		}
		switch st.Code() {
		case codes.Internal, codes.Unknown, codes.DataLoss:
			return true
		}
		return false
	}
}

// RetryOnRateLimit 限流时重试（可配置最大尝试次数）
func RetryOnRateLimit(maxAttempts int) RetryPredicate {
	return func(err error, attempt int) bool {
		if attempt >= maxAttempts {
			return false
		}
		st, ok := status.FromError(err)
		if !ok {
			return false
		}
		return st.Code() == codes.ResourceExhausted
	}
}

// CombinePredicates 组合多个重试判断
func CombinePredicates(predicates ...RetryPredicate) RetryPredicate {
	return func(err error, attempt int) bool {
		for _, p := range predicates {
			if p(err, attempt) {
				return true
			}
		}
		return false
	}
}

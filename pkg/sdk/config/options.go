package config

import "google.golang.org/grpc"

// ClientOption 客户端选项函数
type ClientOption func(*ClientOptions)

// TracingHook Tracing 钩子接口（与 observability 包解耦）
type TracingHook interface {
	StartSpan(ctx interface{}, name string) (interface{}, func())
	SetAttributes(ctx interface{}, attrs map[string]string)
	RecordError(ctx interface{}, err error)
}

// ClientOptions 客户端选项集合
type ClientOptions struct {
	UnaryInterceptors  []grpc.UnaryClientInterceptor
	StreamInterceptors []grpc.StreamClientInterceptor
	DialOptions        []grpc.DialOption

	// TracingHook 用户提供的 Tracing 钩子
	TracingHook interface{}

	// MetricsCollector 用户提供的 Metrics 收集器（覆盖默认）
	MetricsCollector interface{}

	// DisableDefaultInterceptors 禁用默认拦截器
	DisableDefaultInterceptors bool
}

// WithUnaryInterceptors 添加 Unary 拦截器
func WithUnaryInterceptors(interceptors ...grpc.UnaryClientInterceptor) ClientOption {
	return func(o *ClientOptions) {
		o.UnaryInterceptors = append(o.UnaryInterceptors, interceptors...)
	}
}

// WithStreamInterceptors 添加 Stream 拦截器
func WithStreamInterceptors(interceptors ...grpc.StreamClientInterceptor) ClientOption {
	return func(o *ClientOptions) {
		o.StreamInterceptors = append(o.StreamInterceptors, interceptors...)
	}
}

// WithDialOptions 添加 gRPC DialOption
func WithDialOptions(opts ...grpc.DialOption) ClientOption {
	return func(o *ClientOptions) {
		o.DialOptions = append(o.DialOptions, opts...)
	}
}

// WithTracingHook 设置 Tracing 钩子
func WithTracingHook(hook interface{}) ClientOption {
	return func(o *ClientOptions) {
		o.TracingHook = hook
	}
}

// WithMetricsCollector 设置自定义 Metrics 收集器
func WithMetricsCollector(collector interface{}) ClientOption {
	return func(o *ClientOptions) {
		o.MetricsCollector = collector
	}
}

// WithDisableDefaultInterceptors 禁用默认拦截器
func WithDisableDefaultInterceptors() ClientOption {
	return func(o *ClientOptions) {
		o.DisableDefaultInterceptors = true
	}
}

// ApplyOptions 应用选项
func ApplyOptions(opts ...ClientOption) *ClientOptions {
	options := &ClientOptions{}
	for _, opt := range opts {
		opt(options)
	}
	return options
}

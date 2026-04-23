package config

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

// ClientOption 客户端选项函数
type ClientOption func(*ClientOptions)

// MetricsCollector 自定义指标收集器接口。
type MetricsCollector interface {
	RecordRequest(method string, code string, duration time.Duration)
}

// TracingHook Tracing 钩子接口。
type TracingHook interface {
	StartSpan(ctx context.Context, name string) (context.Context, func())
	SetAttributes(ctx context.Context, attrs map[string]string)
	RecordError(ctx context.Context, err error)
}

// ClientOptions 客户端选项集合
type ClientOptions struct {
	UnaryInterceptors  []grpc.UnaryClientInterceptor
	StreamInterceptors []grpc.StreamClientInterceptor
	DialOptions        []grpc.DialOption

	// TracingHook 用户提供的 Tracing 钩子
	TracingHook TracingHook

	// MetricsCollector 用户提供的 Metrics 收集器（覆盖默认）
	MetricsCollector MetricsCollector

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
func WithTracingHook(hook TracingHook) ClientOption {
	return func(o *ClientOptions) {
		o.TracingHook = hook
	}
}

// WithMetricsCollector 设置自定义 Metrics 收集器
func WithMetricsCollector(collector MetricsCollector) ClientOption {
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

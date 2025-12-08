// Package observability 提供可观测性支持
package observability

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// ========== Metrics ==========

// MetricsCollector 指标收集器接口
// 实现者可以对接 Prometheus、OpenTelemetry 等监控系统
type MetricsCollector interface {
	// RecordRequest 记录请求指标
	RecordRequest(method string, code string, duration time.Duration)
}

// MetricsUnaryInterceptor 返回 Metrics 收集拦截器
func MetricsUnaryInterceptor(collector MetricsCollector) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{},
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {

		start := time.Now()
		err := invoker(ctx, method, req, reply, cc, opts...)
		duration := time.Since(start)

		code := "OK"
		if err != nil {
			if st, ok := status.FromError(err); ok {
				code = st.Code().String()
			} else {
				code = "UNKNOWN"
			}
		}

		collector.RecordRequest(method, code, duration)
		return err
	}
}

// ========== Tracing ==========

// TracingHook Tracing 钩子接口
// 实现者可以对接 OpenTelemetry、Jaeger 等追踪系统
type TracingHook interface {
	// StartSpan 开始一个 span
	StartSpan(ctx context.Context, name string) (context.Context, func())

	// SetAttributes 设置 span 属性
	SetAttributes(ctx context.Context, attrs map[string]string)

	// RecordError 记录错误
	RecordError(ctx context.Context, err error)
}

// TracingUnaryInterceptor 返回 Tracing 拦截器
func TracingUnaryInterceptor(hook TracingHook) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{},
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {

		ctx, endSpan := hook.StartSpan(ctx, method)
		defer endSpan()

		hook.SetAttributes(ctx, map[string]string{
			"rpc.system":  "grpc",
			"rpc.service": extractService(method),
			"rpc.method":  extractMethod(method),
		})

		err := invoker(ctx, method, req, reply, cc, opts...)
		if err != nil {
			hook.RecordError(ctx, err)
		}

		return err
	}
}

// ========== 工具函数 ==========

func extractService(fullMethod string) string {
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

func extractMethod(fullMethod string) string {
	if len(fullMethod) > 0 && fullMethod[0] == '/' {
		fullMethod = fullMethod[1:]
	}
	for i := 0; i < len(fullMethod); i++ {
		if fullMethod[i] == '/' {
			return fullMethod[i+1:]
		}
	}
	return fullMethod
}

// ========== Noop 实现 ==========

// NoopMetricsCollector 空操作指标收集器
type NoopMetricsCollector struct{}

func (NoopMetricsCollector) RecordRequest(method string, code string, duration time.Duration) {}

// NoopTracingHook 空操作追踪钩子
type NoopTracingHook struct{}

func (NoopTracingHook) StartSpan(ctx context.Context, name string) (context.Context, func()) {
	return ctx, func() {}
}

func (NoopTracingHook) SetAttributes(ctx context.Context, attrs map[string]string) {}

func (NoopTracingHook) RecordError(ctx context.Context, err error) {}

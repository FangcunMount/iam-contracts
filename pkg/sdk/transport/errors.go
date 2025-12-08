// Package transport 提供 gRPC 传输层功能
package transport

import (
	"context"

	sdkerrors "github.com/FangcunMount/iam-contracts/pkg/sdk/errors"
	"google.golang.org/grpc"
)

// =============================================================================
// 错误包装拦截器
// =============================================================================

// ErrorWrappingInterceptor 错误包装拦截器，将 gRPC 错误转换为 IAMError
func ErrorWrappingInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		err := invoker(ctx, method, req, reply, cc, opts...)
		if err != nil {
			return sdkerrors.Wrap(err)
		}
		return nil
	}
}

// ErrorWrappingStreamInterceptor 流式调用的错误包装拦截器
func ErrorWrappingStreamInterceptor() grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		stream, err := streamer(ctx, desc, cc, method, opts...)
		if err != nil {
			return nil, sdkerrors.Wrap(err)
		}
		return &wrappedClientStream{ClientStream: stream}, nil
	}
}

type wrappedClientStream struct {
	grpc.ClientStream
}

func (w *wrappedClientStream) SendMsg(m interface{}) error {
	err := w.ClientStream.SendMsg(m)
	if err != nil {
		return sdkerrors.Wrap(err)
	}
	return nil
}

func (w *wrappedClientStream) RecvMsg(m interface{}) error {
	err := w.ClientStream.RecvMsg(m)
	if err != nil {
		return sdkerrors.Wrap(err)
	}
	return nil
}

// =============================================================================
// 错误处理拦截器
// =============================================================================

// ErrorHandlerConfig 错误处理拦截器配置
type ErrorHandlerConfig struct {
	// OnError 错误处理回调
	OnError func(ctx context.Context, method string, err error)

	// TransformError 错误转换函数
	TransformError func(err error) error

	// IgnoreErrors 忽略的错误匹配器
	IgnoreErrors sdkerrors.ErrorMatcher
}

// ErrorHandlerInterceptor 错误处理拦截器
func ErrorHandlerInterceptor(cfg *ErrorHandlerConfig) grpc.UnaryClientInterceptor {
	if cfg == nil {
		cfg = &ErrorHandlerConfig{}
	}

	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		err := invoker(ctx, method, req, reply, cc, opts...)
		if err == nil {
			return nil
		}

		// 忽略特定错误
		if cfg.IgnoreErrors != nil && cfg.IgnoreErrors.Match(err) {
			return nil
		}

		// 回调
		if cfg.OnError != nil {
			cfg.OnError(ctx, method, err)
		}

		// 转换
		if cfg.TransformError != nil {
			return cfg.TransformError(err)
		}

		return sdkerrors.Wrap(err)
	}
}

// =============================================================================
// 错误分析拦截器
// =============================================================================

// ErrorAnalyzer 错误分析回调
type ErrorAnalyzer func(ctx context.Context, method string, details *sdkerrors.ErrorDetails)

// ErrorAnalysisInterceptor 错误分析拦截器（用于观测）
func ErrorAnalysisInterceptor(analyzer ErrorAnalyzer) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		err := invoker(ctx, method, req, reply, cc, opts...)
		if err != nil && analyzer != nil {
			details := sdkerrors.Analyze(err)
			analyzer(ctx, method, details)
		}
		return err
	}
}

// =============================================================================
// 拦截器链构建器
// =============================================================================

// InterceptorChainBuilder 拦截器链构建器
type InterceptorChainBuilder struct {
	interceptors []grpc.UnaryClientInterceptor
}

// NewInterceptorChainBuilder 创建拦截器链构建器
func NewInterceptorChainBuilder() *InterceptorChainBuilder {
	return &InterceptorChainBuilder{
		interceptors: make([]grpc.UnaryClientInterceptor, 0),
	}
}

// Add 添加拦截器
func (b *InterceptorChainBuilder) Add(interceptors ...grpc.UnaryClientInterceptor) *InterceptorChainBuilder {
	b.interceptors = append(b.interceptors, interceptors...)
	return b
}

// WithErrorWrapping 添加错误包装
func (b *InterceptorChainBuilder) WithErrorWrapping() *InterceptorChainBuilder {
	return b.Add(ErrorWrappingInterceptor())
}

// WithTimeout 添加超时拦截器
func (b *InterceptorChainBuilder) WithTimeout(configs *MethodConfigs) *InterceptorChainBuilder {
	return b.Add(TimeoutUnaryInterceptor(configs))
}

// WithRetry 添加重试拦截器
func (b *InterceptorChainBuilder) WithRetry(configs *MethodConfigs) *InterceptorChainBuilder {
	return b.Add(RetryUnaryInterceptor(configs))
}

// WithErrorHandler 添加错误处理
func (b *InterceptorChainBuilder) WithErrorHandler(cfg *ErrorHandlerConfig) *InterceptorChainBuilder {
	return b.Add(ErrorHandlerInterceptor(cfg))
}

// WithErrorAnalysis 添加错误分析
func (b *InterceptorChainBuilder) WithErrorAnalysis(analyzer ErrorAnalyzer) *InterceptorChainBuilder {
	return b.Add(ErrorAnalysisInterceptor(analyzer))
}

// Build 构建拦截器切片
func (b *InterceptorChainBuilder) Build() []grpc.UnaryClientInterceptor {
	return b.interceptors
}

// BuildDialOption 构建为 DialOption
func (b *InterceptorChainBuilder) BuildDialOption() grpc.DialOption {
	return grpc.WithChainUnaryInterceptor(b.interceptors...)
}

// =============================================================================
// 预定义错误处理配置
// =============================================================================

// DefaultErrorHandlerConfig 默认错误处理配置
func DefaultErrorHandlerConfig() *ErrorHandlerConfig {
	return &ErrorHandlerConfig{
		TransformError: sdkerrors.Wrap,
	}
}

// LoggingErrorHandlerConfig 带日志的错误处理配置
func LoggingErrorHandlerConfig(logger func(method string, err error)) *ErrorHandlerConfig {
	return &ErrorHandlerConfig{
		OnError: func(ctx context.Context, method string, err error) {
			if logger != nil {
				logger(method, err)
			}
		},
		TransformError: sdkerrors.Wrap,
	}
}

// IgnoreNotFoundConfig 忽略 NotFound 错误的配置
func IgnoreNotFoundConfig() *ErrorHandlerConfig {
	return &ErrorHandlerConfig{
		IgnoreErrors:   sdkerrors.ResourceErrors,
		TransformError: sdkerrors.Wrap,
	}
}

package transport

import "google.golang.org/grpc"

// InterceptorChainBuilder 拦截器链构建器。
type InterceptorChainBuilder struct {
	interceptors []grpc.UnaryClientInterceptor
}

// NewInterceptorChainBuilder 创建拦截器链构建器。
func NewInterceptorChainBuilder() *InterceptorChainBuilder {
	return &InterceptorChainBuilder{
		interceptors: make([]grpc.UnaryClientInterceptor, 0),
	}
}

// Add 添加拦截器。
func (b *InterceptorChainBuilder) Add(interceptors ...grpc.UnaryClientInterceptor) *InterceptorChainBuilder {
	b.interceptors = append(b.interceptors, interceptors...)
	return b
}

// WithErrorWrapping 添加错误包装。
func (b *InterceptorChainBuilder) WithErrorWrapping() *InterceptorChainBuilder {
	return b.Add(ErrorWrappingInterceptor())
}

// WithTimeout 添加超时拦截器。
func (b *InterceptorChainBuilder) WithTimeout(configs *MethodConfigs) *InterceptorChainBuilder {
	return b.Add(TimeoutUnaryInterceptor(configs))
}

// WithRetry 添加重试拦截器。
func (b *InterceptorChainBuilder) WithRetry(configs *MethodConfigs) *InterceptorChainBuilder {
	return b.Add(RetryUnaryInterceptor(configs))
}

// WithErrorHandler 添加错误处理。
func (b *InterceptorChainBuilder) WithErrorHandler(cfg *ErrorHandlerConfig) *InterceptorChainBuilder {
	return b.Add(ErrorHandlerInterceptor(cfg))
}

// WithErrorAnalysis 添加错误分析。
func (b *InterceptorChainBuilder) WithErrorAnalysis(analyzer ErrorAnalyzer) *InterceptorChainBuilder {
	return b.Add(ErrorAnalysisInterceptor(analyzer))
}

// Build 构建拦截器切片。
func (b *InterceptorChainBuilder) Build() []grpc.UnaryClientInterceptor {
	return b.interceptors
}

// BuildDialOption 构建为 DialOption。
func (b *InterceptorChainBuilder) BuildDialOption() grpc.DialOption {
	return grpc.WithChainUnaryInterceptor(b.interceptors...)
}

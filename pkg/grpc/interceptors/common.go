// Package interceptors 提供通用 gRPC 拦截器
package interceptors

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// ===== 通用拦截器 =====

// RecoveryInterceptor 恢复拦截器，防止 panic 导致服务崩溃
func RecoveryInterceptor(opts ...RecoveryOption) grpc.UnaryServerInterceptor {
	options := defaultRecoveryOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		// 上下文预处理
		if options.contextPreprocess != nil {
			ctx = options.contextPreprocess(ctx)
		}

		defer func() {
			if r := recover(); r != nil {
				if options.logger != nil {
					options.logger.LogError("gRPC request panic recovered",
						map[string]interface{}{
							"method": info.FullMethod,
							"panic":  r,
							"stack":  string(debug.Stack()),
						})
				}
				err = status.Error(codes.Internal, fmt.Sprintf("internal server error: %v", r))
			}
		}()

		return handler(ctx, req)
	}
}

// RecoveryStreamInterceptor 流式恢复拦截器
func RecoveryStreamInterceptor(opts ...RecoveryOption) grpc.StreamServerInterceptor {
	options := defaultRecoveryOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		defer func() {
			if r := recover(); r != nil {
				if options.logger != nil {
					options.logger.LogError("gRPC stream panic recovered",
						map[string]interface{}{
							"method": info.FullMethod,
							"panic":  r,
							"stack":  string(debug.Stack()),
						})
				}
				err = status.Error(codes.Internal, fmt.Sprintf("internal server error: %v", r))
			}
		}()

		return handler(srv, ss)
	}
}

// RequestIDInterceptor 请求ID拦截器，为每个请求生成唯一ID
func RequestIDInterceptor(opts ...RequestIDOption) grpc.UnaryServerInterceptor {
	options := defaultRequestIDOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		requestID := options.generator()
		ctx = context.WithValue(ctx, RequestIDContextKey, requestID)

		// 如果有 metadata 注入回调，执行它
		if options.metadataInjector != nil {
			ctx = options.metadataInjector(ctx, requestID)
		}

		return handler(ctx, req)
	}
}

// RequestIDStreamInterceptor 流式请求ID拦截器
func RequestIDStreamInterceptor(opts ...RequestIDOption) grpc.StreamServerInterceptor {
	options := defaultRequestIDOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()
		requestID := options.generator()
		ctx = context.WithValue(ctx, RequestIDContextKey, requestID)

		if options.metadataInjector != nil {
			ctx = options.metadataInjector(ctx, requestID)
		}

		wrappedStream := &WrappedServerStream{
			ServerStream: ss,
			Ctx:          ctx,
		}

		return handler(srv, wrappedStream)
	}
}

// LoggingInterceptor 日志拦截器
func LoggingInterceptor(logger InterceptorLogger, opts ...LoggingOption) grpc.UnaryServerInterceptor {
	options := defaultLoggingOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// 跳过不需要记录的方法
		if options.skipMatcher.Match(info.FullMethod) {
			return handler(ctx, req)
		}

		// 上下文预处理（如注入追踪上下文）
		if options.contextPreprocess != nil {
			ctx = options.contextPreprocess(ctx)
		}

		start := time.Now()

		// 获取客户端信息
		clientIP := GetClientIP(ctx)
		userAgent := GetUserAgent(ctx)
		requestID := RequestIDFromContext(ctx)

		if logger != nil {
			logger.LogInfo("gRPC request started",
				map[string]interface{}{
					"method":     info.FullMethod,
					"client_ip":  clientIP,
					"user_agent": userAgent,
					"request_id": requestID,
				})
		}

		// 执行实际的处理器
		resp, err := handler(ctx, req)

		// 计算执行时间
		duration := time.Since(start)

		// 获取状态码和错误信息
		statusCode := codes.OK
		errorMsg := ""
		if err != nil {
			if st, ok := status.FromError(err); ok {
				statusCode = st.Code()
				errorMsg = st.Message()
			} else {
				statusCode = codes.Internal
				errorMsg = err.Error()
			}
		}

		if logger != nil {
			fields := map[string]interface{}{
				"method":      info.FullMethod,
				"request_id":  requestID,
				"duration_ms": duration.Milliseconds(),
				"status_code": statusCode.String(),
			}

			if err != nil {
				fields["error"] = errorMsg
				logger.LogError("gRPC request failed", fields)
			} else {
				if options.logResponse {
					fields["response_summary"] = generateResponseSummary(resp, options.maxResponseLen)
				}
				logger.LogInfo("gRPC request completed", fields)
			}
		}

		return resp, err
	}
}

// LoggingStreamInterceptor 流式日志拦截器
func LoggingStreamInterceptor(logger InterceptorLogger, opts ...LoggingOption) grpc.StreamServerInterceptor {
	options := defaultLoggingOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if options.skipMatcher.Match(info.FullMethod) {
			return handler(srv, ss)
		}

		start := time.Now()
		ctx := ss.Context()
		clientIP := GetClientIP(ctx)
		requestID := RequestIDFromContext(ctx)

		if logger != nil {
			logger.LogInfo("gRPC stream started",
				map[string]interface{}{
					"method":     info.FullMethod,
					"client_ip":  clientIP,
					"request_id": requestID,
				})
		}

		err := handler(srv, ss)

		duration := time.Since(start)
		statusCode := codes.OK
		if err != nil {
			if st, ok := status.FromError(err); ok {
				statusCode = st.Code()
			} else {
				statusCode = codes.Internal
			}
		}

		if logger != nil {
			fields := map[string]interface{}{
				"method":      info.FullMethod,
				"request_id":  requestID,
				"duration_ms": duration.Milliseconds(),
				"status_code": statusCode.String(),
			}

			if err != nil {
				fields["error"] = err.Error()
				logger.LogError("gRPC stream failed", fields)
			} else {
				logger.LogInfo("gRPC stream completed", fields)
			}
		}

		return err
	}
}

// ===== 上下文工具函数 =====

// RequestIDContextKey 请求ID上下文键
const RequestIDContextKey contextKey = "grpc_request_id"

// RequestIDFromContext 从上下文获取请求ID
func RequestIDFromContext(ctx context.Context) string {
	if requestID, ok := ctx.Value(RequestIDContextKey).(string); ok {
		return requestID
	}
	return ""
}

// GetClientIP 获取客户端IP地址
func GetClientIP(ctx context.Context) string {
	if p, ok := peer.FromContext(ctx); ok {
		return p.Addr.String()
	}
	return "unknown"
}

// GetUserAgent 获取用户代理信息
func GetUserAgent(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if userAgent := md.Get("user-agent"); len(userAgent) > 0 {
			return userAgent[0]
		}
	}
	return "unknown"
}

// GetMetadataValue 获取 metadata 中的值
func GetMetadataValue(ctx context.Context, key string) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		values := md.Get(key)
		if len(values) > 0 {
			return values[0]
		}
	}
	return ""
}

// ===== 选项定义 =====

// recoveryOptions 恢复拦截器选项
type recoveryOptions struct {
	logger            InterceptorLogger
	contextPreprocess func(ctx context.Context) context.Context // 上下文预处理钩子
}

func defaultRecoveryOptions() *recoveryOptions {
	return &recoveryOptions{}
}

// RecoveryOption 恢复拦截器选项函数
type RecoveryOption func(*recoveryOptions)

// WithRecoveryLogger 设置恢复拦截器的日志记录器
func WithRecoveryLogger(logger InterceptorLogger) RecoveryOption {
	return func(o *recoveryOptions) {
		o.logger = logger
	}
}

// WithRecoveryContextPreprocess 设置恢复拦截器的上下文预处理钩子
func WithRecoveryContextPreprocess(fn func(ctx context.Context) context.Context) RecoveryOption {
	return func(o *recoveryOptions) {
		o.contextPreprocess = fn
	}
}

// requestIDOptions 请求ID拦截器选项
type requestIDOptions struct {
	generator        func() string
	metadataInjector func(ctx context.Context, requestID string) context.Context
}

func defaultRequestIDOptions() *requestIDOptions {
	return &requestIDOptions{
		generator: DefaultRequestIDGenerator,
	}
}

// RequestIDOption 请求ID拦截器选项函数
type RequestIDOption func(*requestIDOptions)

// WithRequestIDGenerator 设置请求ID生成器
func WithRequestIDGenerator(generator func() string) RequestIDOption {
	return func(o *requestIDOptions) {
		o.generator = generator
	}
}

// WithMetadataInjector 设置 metadata 注入回调
func WithMetadataInjector(injector func(ctx context.Context, requestID string) context.Context) RequestIDOption {
	return func(o *requestIDOptions) {
		o.metadataInjector = injector
	}
}

// DefaultRequestIDGenerator 默认请求ID生成器
func DefaultRequestIDGenerator() string {
	return fmt.Sprintf("req-%d", time.Now().UnixNano())
}

// loggingOptions 日志拦截器选项
type loggingOptions struct {
	skipMatcher       *SkipMethodMatcher
	logResponse       bool
	maxResponseLen    int
	contextPreprocess func(ctx context.Context) context.Context // 上下文预处理钩子
}

func defaultLoggingOptions() *loggingOptions {
	return &loggingOptions{
		skipMatcher:    NewSkipMethodMatcher(DefaultSkipMethods()...),
		logResponse:    true,
		maxResponseLen: 300,
	}
}

// LoggingOption 日志拦截器选项函数
type LoggingOption func(*loggingOptions)

// WithLoggingSkipMethods 设置跳过日志记录的方法
func WithLoggingSkipMethods(methods ...string) LoggingOption {
	return func(o *loggingOptions) {
		o.skipMatcher.Add(methods...)
	}
}

// WithLogResponse 设置是否记录响应
func WithLogResponse(logResponse bool) LoggingOption {
	return func(o *loggingOptions) {
		o.logResponse = logResponse
	}
}

// WithMaxResponseLen 设置响应摘要最大长度
func WithMaxResponseLen(maxLen int) LoggingOption {
	return func(o *loggingOptions) {
		o.maxResponseLen = maxLen
	}
}

// WithContextPreprocess 设置上下文预处理钩子（用于注入追踪上下文等）
func WithContextPreprocess(fn func(ctx context.Context) context.Context) LoggingOption {
	return func(o *loggingOptions) {
		o.contextPreprocess = fn
	}
}

// ===== 辅助函数 =====

// generateResponseSummary 生成响应摘要
func generateResponseSummary(resp interface{}, maxLength int) string {
	if resp == nil {
		return "nil"
	}

	respStr := fmt.Sprintf("%+v", resp)
	if respStr == "" {
		return "empty"
	}

	if len(respStr) > maxLength {
		return respStr[:maxLength] + "..."
	}

	return respStr
}

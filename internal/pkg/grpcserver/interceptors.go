package grpcserver

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

	"github.com/FangcunMount/component-base/pkg/log"
	"github.com/FangcunMount/component-base/pkg/util/idutil"
)

// LoggingInterceptor 统一的 gRPC 日志拦截器
func LoggingInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		ctx = ensureTraceContext(ctx)
		start := time.Now()

		// 获取客户端信息
		clientIP := getClientIP(ctx)
		userAgent := getUserAgent(ctx)
		requestID := getRequestID(ctx)
		headers := getHeaders(ctx)

		startFields := append([]log.Field{
			log.String("method", info.FullMethod),
			log.String("client_ip", clientIP),
			log.String("user_agent", userAgent),
			log.Any("metadata", headers),
		}, log.TraceFields(ctx)...)
		log.GRPC("gRPC request started", startFields...)

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

		fields := []log.Field{
			log.String("method", info.FullMethod),
			log.String("request_id", requestID),
			log.Duration("duration", duration),
			log.String("status_code", statusCode.String()),
		}

		if err != nil {
			fields = append(fields, log.String("error", errorMsg))
			fields = append(fields, log.TraceFields(ctx)...)
			log.GRPCError("gRPC request failed", fields...)
		} else {
			responseSummary := generateResponseSummary(resp)
			fields = append(fields, log.String("response_summary", responseSummary))
			fields = append(fields, log.TraceFields(ctx)...)
			log.GRPC("gRPC request completed", fields...)
		}

		return resp, err
	}
}

// RecoveryInterceptor 恢复拦截器，防止 panic 导致服务崩溃
func RecoveryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		ctx = ensureTraceContext(ctx)
		defer func() {
			if r := recover(); r != nil {
				fields := append([]log.Field{
					log.String("method", info.FullMethod),
					log.Any("panic", r),
					log.String("stack", string(debug.Stack())),
				}, log.TraceFields(ctx)...)
				log.GRPCError("gRPC request panic recovered", fields...)
				err = status.Error(codes.Internal, fmt.Sprintf("内部服务器错误: %v", r))
			}
		}()

		return handler(ctx, req)
	}
}

// RequestIDInterceptor 请求ID拦截器，为每个请求生成唯一ID
func RequestIDInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		requestID := generateRequestID()
		ctx = log.WithRequestID(ctx, requestID)

		return handler(ctx, req)
	}
}

// getClientIP 获取客户端IP地址
func getClientIP(ctx context.Context) string {
	if peer, ok := peer.FromContext(ctx); ok {
		return peer.Addr.String()
	}
	return "unknown"
}

// getUserAgent 获取用户代理信息
func getUserAgent(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if userAgent := md.Get("user-agent"); len(userAgent) > 0 {
			return userAgent[0]
		}
	}
	return "unknown"
}

// getHeaders 获取请求头信息
func getHeaders(ctx context.Context) map[string][]string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		// 只记录关键的头信息，避免日志过长
		headers := make(map[string][]string)
		for key, values := range md {
			// 过滤敏感信息
			if key != "authorization" && key != "cookie" && key != "x-api-key" {
				headers[key] = values
			}
		}
		return headers
	}
	return map[string][]string{}
}

// getRequestID 从上下文获取请求ID
func getRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value("request_id").(string); ok {
		return requestID
	}
	if requestID := log.ExtractRequestID(ctx); requestID != "" {
		return requestID
	}
	return "unknown"
}

// generateRequestID 生成请求ID
func generateRequestID() string {
	return idutil.NewRequestID()
}

// generateResponseSummary 生成响应摘要
func generateResponseSummary(resp interface{}) string {
	if resp == nil {
		return "nil"
	}

	// 将响应转换为字符串
	respStr := fmt.Sprintf("%+v", resp)

	// 如果响应为空字符串，返回特殊标记
	if respStr == "" {
		return "empty_string"
	}

	// 如果响应只包含默认值，也要显示
	if len(respStr) == 0 {
		return "zero_length"
	}

	// 如果响应太长，截断并添加省略号
	maxLength := 300 // 增加长度限制
	if len(respStr) > maxLength {
		return respStr[:maxLength] + "..."
	}

	return respStr
}

func ensureTraceContext(ctx context.Context) context.Context {
	traceID := metadataValue(ctx, "x-trace-id")
	if traceID == "" {
		traceID = idutil.NewTraceID()
	}

	requestID := metadataValue(ctx, "x-request-id")
	if requestID == "" {
		requestID = idutil.NewRequestID()
	}

	spanID := idutil.NewSpanID()

	return log.WithTraceContext(ctx, traceID, spanID, requestID)
}

func metadataValue(ctx context.Context, key string) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		values := md.Get(key)
		if len(values) > 0 {
			return values[0]
		}
	}
	return ""
}

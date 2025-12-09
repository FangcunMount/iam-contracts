package transport

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// 元数据 key 常量
const (
	MetadataKeyRequestID = "x-request-id"
	MetadataKeyTraceID   = "x-trace-id"
	MetadataKeySpanID    = "x-span-id"
)

// RequestIDInterceptor 自动注入 request-id 的拦截器。
//
// 如果请求上下文中没有 request-id，则自动生成 UUID。
//
// 返回：
//   - grpc.UnaryClientInterceptor: gRPC 客户端拦截器
func RequestIDInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{},
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {

		// 检查是否已有 request-id
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		}

		if len(md.Get(MetadataKeyRequestID)) == 0 {
			md = md.Copy()
			md.Set(MetadataKeyRequestID, uuid.New().String())
			ctx = metadata.NewOutgoingContext(ctx, md)
		}

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// MetadataInterceptor 注入默认元数据的拦截器。
//
// 将配置中的默认 metadata 注入到每个请求中，已存在的 key 不会被覆盖。
//
// 参数：
//   - defaultMeta: 默认元数据键值对
//
// 返回：
//   - grpc.UnaryClientInterceptor: gRPC 客户端拦截器
func MetadataInterceptor(defaultMeta map[string]string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{},
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {

		if len(defaultMeta) > 0 {
			md, ok := metadata.FromOutgoingContext(ctx)
			if !ok {
				md = metadata.New(nil)
			}
			md = md.Copy()

			for k, v := range defaultMeta {
				if len(md.Get(k)) == 0 {
					md.Set(k, v)
				}
			}
			ctx = metadata.NewOutgoingContext(ctx, md)
		}

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// TimeoutInterceptor 请求超时拦截器
func TimeoutInterceptor(timeout interface{}) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{},
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// 超时由 gRPC 本身处理，这里可以扩展自定义逻辑
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// ========== Context 工具函数 ==========

type contextKey string

const (
	requestIDKey contextKey = "request_id"
	traceIDKey   contextKey = "trace_id"
)

// WithRequestID 设置请求 ID。
//
// 将 request-id 添加到 context 和 gRPC metadata 中。
//
// 参数：
//   - ctx: 上下文
//   - requestID: 请求 ID
//
// 返回：
//   - context.Context: 包含 request-id 的新上下文
//
// 示例：
//
//	ctx := sdk.WithRequestID(ctx, "req-123456")
//	resp, err := client.Auth().VerifyToken(ctx, req)
func WithRequestID(ctx context.Context, requestID string) context.Context {
	ctx = context.WithValue(ctx, requestIDKey, requestID)

	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	}
	md = md.Copy()
	md.Set(MetadataKeyRequestID, requestID)

	return metadata.NewOutgoingContext(ctx, md)
}

// WithTraceID 设置追踪 ID。
//
// 将 trace-id 添加到 context 和 gRPC metadata 中，用于分布式追踪。
//
// 参数：
//   - ctx: 上下文
//   - traceID: 追踪 ID
//
// 返回：
//   - context.Context: 包含 trace-id 的新上下文
//
// 示例：
//
//	ctx := sdk.WithTraceID(ctx, "trace-abc123")
//	resp, err := client.Identity().GetUser(ctx, "user-123")
func WithTraceID(ctx context.Context, traceID string) context.Context {
	ctx = context.WithValue(ctx, traceIDKey, traceID)

	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	}
	md = md.Copy()
	md.Set(MetadataKeyTraceID, traceID)

	return metadata.NewOutgoingContext(ctx, md)
}

// GetRequestID 获取请求 ID。
//
// 从 context 或 gRPC metadata 中提取 request-id。
//
// 参数：
//   - ctx: 上下文
//
// 返回：
//   - string: 请求 ID，如果不存在则返回空字符串
func GetRequestID(ctx context.Context) string {
	if v := ctx.Value(requestIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}

	if md, ok := metadata.FromOutgoingContext(ctx); ok {
		if vals := md.Get(MetadataKeyRequestID); len(vals) > 0 {
			return vals[0]
		}
	}

	return ""
}

// GetTraceID 获取追踪 ID。
//
// 从 context 或 gRPC metadata 中提取 trace-id。
//
// 参数：
//   - ctx: 上下文
//
// 返回：
//   - string: 追踪 ID，如果不存在则返回空字符串
func GetTraceID(ctx context.Context) string {
	if v := ctx.Value(traceIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}

	if md, ok := metadata.FromOutgoingContext(ctx); ok {
		if vals := md.Get(MetadataKeyTraceID); len(vals) > 0 {
			return vals[0]
		}
	}

	return ""
}

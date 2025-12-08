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

// RequestIDInterceptor 自动注入 request-id 的拦截器
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

// MetadataInterceptor 注入默认元数据的拦截器
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

// WithRequestID 设置 request-id
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

// WithTraceID 设置 trace-id
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

// GetRequestID 获取 request-id
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

// GetTraceID 获取 trace-id
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

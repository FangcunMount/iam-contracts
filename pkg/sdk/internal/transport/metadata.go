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
func RequestIDInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{},
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
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

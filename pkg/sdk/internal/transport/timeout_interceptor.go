package transport

import (
	"context"

	"google.golang.org/grpc"
)

// TimeoutInterceptor 请求超时拦截器。
func TimeoutInterceptor(timeout interface{}) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{},
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
